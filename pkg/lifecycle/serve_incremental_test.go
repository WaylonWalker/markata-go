package lifecycle

import "testing"

func TestServeIncrementalIsIndependentFromFastMode(t *testing.T) {
	m := NewManager()
	if IsServeIncremental(m) {
		t.Fatal("incremental mode enabled by default")
	}
	SetServeIncremental(m, true)
	if !IsServeIncremental(m) {
		t.Fatal("incremental mode was not enabled")
	}
	if IsServeFastMode(m) {
		t.Fatal("incremental mode unexpectedly enabled fast mode")
	}
}

func TestServeFastModeImpliesIncremental(t *testing.T) {
	m := NewManager()
	m.Config().Extra["fast_mode"] = true
	if !IsServeIncremental(m) {
		t.Fatal("fast mode did not enable incremental behavior")
	}
}
