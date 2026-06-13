package buildstats

import (
	"testing"
	"time"
)

func TestClassifyInterval_CPUOnly(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 150 * time.Millisecond}

	got := classifyInterval(prev, current, false)

	if got.CPU != 100*time.Millisecond {
		t.Fatalf("CPU = %v, want %v", got.CPU, 100*time.Millisecond)
	}
	if got.NetworkWait != 0 || got.DiskReadWait != 0 || got.DiskWriteWait != 0 || got.Idle != 0 {
		t.Fatalf("unexpected non-CPU time: %+v", got)
	}
}

func TestClassifyInterval_NetworkWait(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 25 * time.Millisecond}

	got := classifyInterval(prev, current, true)

	if got.CPU != 25*time.Millisecond {
		t.Fatalf("CPU = %v, want %v", got.CPU, 25*time.Millisecond)
	}
	if got.NetworkWait != 75*time.Millisecond {
		t.Fatalf("NetworkWait = %v, want %v", got.NetworkWait, 75*time.Millisecond)
	}
}

func TestClassifyInterval_DiskWriteWait(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0, ReadBytes: 10, WriteBytes: 10}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 20 * time.Millisecond, ReadBytes: 10, WriteBytes: 42}

	got := classifyInterval(prev, current, false)

	if got.CPU != 20*time.Millisecond {
		t.Fatalf("CPU = %v, want %v", got.CPU, 20*time.Millisecond)
	}
	if got.DiskWriteWait != 80*time.Millisecond {
		t.Fatalf("DiskWriteWait = %v, want %v", got.DiskWriteWait, 80*time.Millisecond)
	}
	if got.DiskReadWait != 0 {
		t.Fatalf("DiskReadWait = %v, want 0", got.DiskReadWait)
	}
}

func TestClassifyInterval_DiskReadWait(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0, ReadBytes: 10, WriteBytes: 10}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 20 * time.Millisecond, ReadBytes: 42, WriteBytes: 10}

	got := classifyInterval(prev, current, false)

	if got.CPU != 20*time.Millisecond {
		t.Fatalf("CPU = %v, want %v", got.CPU, 20*time.Millisecond)
	}
	if got.DiskReadWait != 80*time.Millisecond {
		t.Fatalf("DiskReadWait = %v, want %v", got.DiskReadWait, 80*time.Millisecond)
	}
	if got.DiskWriteWait != 0 {
		t.Fatalf("DiskWriteWait = %v, want 0", got.DiskWriteWait)
	}
	if got.Idle != 0 {
		t.Fatalf("Idle = %v, want 0", got.Idle)
	}
	if got.NetworkWait != 0 {
		t.Fatalf("NetworkWait = %v, want 0", got.NetworkWait)
	}
	if got.DiskReadWait+got.DiskWriteWait != 80*time.Millisecond {
		t.Fatalf("disk wait total = %v, want %v", got.DiskReadWait+got.DiskWriteWait, 80*time.Millisecond)
	}
}

func TestClassifyInterval_DiskReadWriteWait(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0, ReadBytes: 10, WriteBytes: 10}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 20 * time.Millisecond, ReadBytes: 30, WriteBytes: 70}

	got := classifyInterval(prev, current, false)

	if got.DiskReadWait != 20*time.Millisecond {
		t.Fatalf("DiskReadWait = %v, want %v", got.DiskReadWait, 20*time.Millisecond)
	}
	if got.DiskWriteWait != 60*time.Millisecond {
		t.Fatalf("DiskWriteWait = %v, want %v", got.DiskWriteWait, 60*time.Millisecond)
	}
}

func TestClassifyInterval_Idle(t *testing.T) {
	prev := runtimeSample{At: time.Unix(0, 0), ProcessCPU: 0}
	current := runtimeSample{At: time.Unix(0, int64(100*time.Millisecond)), ProcessCPU: 10 * time.Millisecond}

	got := classifyInterval(prev, current, false)

	if got.CPU != 10*time.Millisecond {
		t.Fatalf("CPU = %v, want %v", got.CPU, 10*time.Millisecond)
	}
	if got.Idle != 90*time.Millisecond {
		t.Fatalf("Idle = %v, want %v", got.Idle, 90*time.Millisecond)
	}
}
