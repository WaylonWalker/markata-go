package buildstats

import (
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const sampleInterval = 100 * time.Millisecond

// ResourceBreakdown is an estimated wall-time split for a build.
type ResourceBreakdown struct {
	CPU         time.Duration
	NetworkWait time.Duration
	DiskWait    time.Duration
	Idle        time.Duration
}

// Hotspot identifies a slow plugin hook.
type Hotspot struct {
	Stage    string
	Plugin   string
	Duration time.Duration
}

// StageTiming records how long a lifecycle stage took.
type StageTiming struct {
	Stage     string
	Duration  time.Duration
	Resources ResourceBreakdown
}

// Summary contains benchmark details for a single build.
type Summary struct {
	Total     time.Duration
	Resources ResourceBreakdown
	Hotspots  []Hotspot
	Stages    []StageTiming
}

type runtimeSample struct {
	At         time.Time
	ProcessCPU time.Duration
	ReadBytes  uint64
	WriteBytes uint64
}

// Profile collects coarse-grained benchmark data for a build.
type Profile struct {
	start time.Time

	mu         sync.Mutex
	lastSample runtimeSample
	resources  ResourceBreakdown
	hotspots   []Hotspot
	stageOrder []string
	stageTimes map[string]time.Duration
	stageUsage map[string]ResourceBreakdown
	current    string

	networkOps atomic.Int64
	stopOnce   sync.Once
	stopCh     chan struct{}
	stoppedCh  chan struct{}
}

var activeProfile atomic.Pointer[Profile]

// Start begins collecting benchmark data for the current process.
func Start() *Profile {
	profile := &Profile{
		start:      time.Now(),
		lastSample: readRuntimeSample(),
		stageTimes: make(map[string]time.Duration),
		stageUsage: make(map[string]ResourceBreakdown),
		stopCh:     make(chan struct{}),
		stoppedCh:  make(chan struct{}),
	}

	activeProfile.Store(profile)
	go profile.sampleLoop()

	return profile
}

// Stop finishes collection and returns a stable summary.
func (p *Profile) Stop() Summary {
	if p == nil {
		return Summary{}
	}

	p.stopOnce.Do(func() {
		activeProfile.CompareAndSwap(p, nil)
		close(p.stopCh)
		<-p.stoppedCh
	})

	p.mu.Lock()
	defer p.mu.Unlock()

	hotspots := append([]Hotspot(nil), p.hotspots...)
	sort.Slice(hotspots, func(i, j int) bool {
		if hotspots[i].Duration == hotspots[j].Duration {
			if hotspots[i].Stage == hotspots[j].Stage {
				return hotspots[i].Plugin < hotspots[j].Plugin
			}
			return hotspots[i].Stage < hotspots[j].Stage
		}
		return hotspots[i].Duration > hotspots[j].Duration
	})

	stages := make([]StageTiming, 0, len(p.stageOrder))
	for _, stage := range p.stageOrder {
		stages = append(stages, StageTiming{Stage: stage, Duration: p.stageTimes[stage], Resources: p.stageUsage[stage]})
	}

	return Summary{
		Total:     time.Since(p.start),
		Resources: p.resources,
		Hotspots:  hotspots,
		Stages:    stages,
	}
}

// RecordPlugin stores a plugin hook timing when a profile is active.
func RecordPlugin(stage, plugin string, duration time.Duration) {
	profile := activeProfile.Load()
	if profile == nil {
		return
	}

	profile.mu.Lock()
	defer profile.mu.Unlock()
	profile.hotspots = append(profile.hotspots, Hotspot{Stage: stage, Plugin: plugin, Duration: duration})
}

// RecordStage stores a stage timing when a profile is active.
func RecordStage(stage string, duration time.Duration) {
	profile := activeProfile.Load()
	if profile == nil {
		return
	}

	profile.mu.Lock()
	defer profile.mu.Unlock()
	if _, ok := profile.stageTimes[stage]; !ok {
		profile.stageOrder = append(profile.stageOrder, stage)
	}
	profile.stageTimes[stage] = duration
}

// SetActiveStage marks the currently running lifecycle stage for resource attribution.
func SetActiveStage(stage string) {
	profile := activeProfile.Load()
	if profile == nil {
		return
	}

	profile.mu.Lock()
	defer profile.mu.Unlock()
	profile.current = stage
	if stage != "" {
		if _, ok := profile.stageTimes[stage]; !ok {
			if _, seen := profile.stageUsage[stage]; !seen {
				profile.stageOrder = append(profile.stageOrder, stage)
			}
		}
	}
}

// InstrumentHTTPClient wraps an HTTP client so active builds can estimate network wait time.
func InstrumentHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return nil
	}
	if _, ok := client.Transport.(*trackingTransport); ok {
		return client
	}

	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = &trackingTransport{base: base}
	return client
}

type trackingTransport struct {
	base http.RoundTripper
}

func (t *trackingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	profile := activeProfile.Load()
	if profile != nil {
		profile.networkOps.Add(1)
		defer profile.networkOps.Add(-1)
	}
	return t.base.RoundTrip(req)
}

func (p *Profile) sampleLoop() {
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()
	defer close(p.stoppedCh)

	for {
		select {
		case <-ticker.C:
			p.sample()
		case <-p.stopCh:
			p.sample()
			return
		}
	}
}

func (p *Profile) sample() {
	current := readRuntimeSample()
	delta := classifyInterval(p.lastSample, current, p.networkOps.Load() > 0)

	p.mu.Lock()
	p.resources.CPU += delta.CPU
	p.resources.NetworkWait += delta.NetworkWait
	p.resources.DiskWait += delta.DiskWait
	p.resources.Idle += delta.Idle
	if p.current != "" {
		usage := p.stageUsage[p.current]
		usage.CPU += delta.CPU
		usage.NetworkWait += delta.NetworkWait
		usage.DiskWait += delta.DiskWait
		usage.Idle += delta.Idle
		p.stageUsage[p.current] = usage
	}
	p.lastSample = current
	p.mu.Unlock()
}

func classifyInterval(prev, current runtimeSample, networkActive bool) ResourceBreakdown {
	elapsed := current.At.Sub(prev.At)
	if elapsed <= 0 {
		return ResourceBreakdown{}
	}

	cpuDelta := current.ProcessCPU - prev.ProcessCPU
	if cpuDelta < 0 {
		cpuDelta = 0
	}
	if cpuDelta > elapsed {
		cpuDelta = elapsed
	}

	breakdown := ResourceBreakdown{CPU: cpuDelta}
	remaining := elapsed - cpuDelta
	if remaining <= 0 {
		return breakdown
	}

	diskChanged := current.ReadBytes > prev.ReadBytes || current.WriteBytes > prev.WriteBytes
	switch {
	case networkActive:
		breakdown.NetworkWait = remaining
	case diskChanged:
		breakdown.DiskWait = remaining
	default:
		breakdown.Idle = remaining
	}

	return breakdown
}
