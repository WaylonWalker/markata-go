package buildstats

import (
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const sampleInterval = 100 * time.Millisecond

// ResourceBreakdown is an estimated wall-time split for a build.
type ResourceBreakdown struct {
	CPU           time.Duration
	NetworkWait   time.Duration
	DiskReadWait  time.Duration
	DiskWriteWait time.Duration
	Idle          time.Duration
}

// Hotspot identifies a slow plugin hook.
type Hotspot struct {
	Stage    string
	Plugin   string
	Duration time.Duration
}

// RequestTiming records a single outbound HTTP request made during a build.
type RequestTiming struct {
	Stage    string
	Plugin   string
	Method   string
	Host     string
	URL      string
	Status   int
	Error    string
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
	Requests  []RequestTiming
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
	plugin     string
	requests   []RequestTiming

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

	requests := append([]RequestTiming(nil), p.requests...)
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].Duration == requests[j].Duration {
			if requests[i].Stage == requests[j].Stage {
				if requests[i].Plugin == requests[j].Plugin {
					return requests[i].URL < requests[j].URL
				}
				return requests[i].Plugin < requests[j].Plugin
			}
			return requests[i].Stage < requests[j].Stage
		}
		return requests[i].Duration > requests[j].Duration
	})

	return Summary{
		Total:     time.Since(p.start),
		Resources: p.resources,
		Hotspots:  hotspots,
		Requests:  requests,
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

// SetActivePlugin marks the currently running plugin hook for request attribution.
func SetActivePlugin(plugin string) {
	profile := activeProfile.Load()
	if profile == nil {
		return
	}

	profile.mu.Lock()
	defer profile.mu.Unlock()
	profile.plugin = plugin
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
	start := time.Now()
	if profile != nil {
		profile.networkOps.Add(1)
		defer profile.networkOps.Add(-1)
	}
	resp, err := t.base.RoundTrip(req)
	if profile != nil {
		profile.recordRequest(req, resp, err, time.Since(start))
	}
	return resp, err
}

func (p *Profile) recordRequest(req *http.Request, resp *http.Response, err error, duration time.Duration) {
	if p == nil || req == nil {
		return
	}

	request := RequestTiming{
		Method:   requestMethod(req.Method),
		URL:      sanitizeURL(req.URL),
		Duration: duration,
	}
	if req.URL != nil {
		request.Host = req.URL.Host
	}
	if resp != nil {
		request.Status = resp.StatusCode
	}
	if err != nil {
		request.Error = err.Error()
	}

	p.mu.Lock()
	request.Stage = p.current
	request.Plugin = p.plugin
	p.requests = append(p.requests, request)
	p.mu.Unlock()
}

func sanitizeURL(raw *url.URL) string {
	if raw == nil {
		return ""
	}
	clean := *raw
	clean.User = nil
	clean.RawQuery = ""
	clean.ForceQuery = false
	clean.Fragment = ""
	if clean.Path == "" {
		clean.Path = "/"
	}
	return clean.String()
}

func requestMethod(method string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		return http.MethodGet
	}
	return method
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
	p.resources.DiskReadWait += delta.DiskReadWait
	p.resources.DiskWriteWait += delta.DiskWriteWait
	p.resources.Idle += delta.Idle
	if p.current != "" {
		usage := p.stageUsage[p.current]
		usage.CPU += delta.CPU
		usage.NetworkWait += delta.NetworkWait
		usage.DiskReadWait += delta.DiskReadWait
		usage.DiskWriteWait += delta.DiskWriteWait
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

	readDelta := deltaBytes(prev.ReadBytes, current.ReadBytes)
	writeDelta := deltaBytes(prev.WriteBytes, current.WriteBytes)
	switch {
	case networkActive:
		breakdown.NetworkWait = remaining
	case readDelta > 0 || writeDelta > 0:
		breakdown.DiskReadWait, breakdown.DiskWriteWait = splitDiskWait(remaining, readDelta, writeDelta)
	default:
		breakdown.Idle = remaining
	}

	return breakdown
}

func deltaBytes(prev, current uint64) uint64 {
	if current <= prev {
		return 0
	}
	return current - prev
}

func safeInt64(v uint64) int64 {
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}

func splitDiskWait(remaining time.Duration, readDelta, writeDelta uint64) (readWait, writeWait time.Duration) {
	total := readDelta + writeDelta
	if total == 0 {
		return 0, 0
	}
	if readDelta == 0 {
		return 0, remaining
	}
	if writeDelta == 0 {
		return remaining, 0
	}

	readWait = time.Duration(int64(remaining) * safeInt64(readDelta) / safeInt64(total))
	if readWait < 0 {
		readWait = 0
	}
	if readWait > remaining {
		readWait = remaining
	}
	writeWait = remaining - readWait
	return readWait, writeWait
}
