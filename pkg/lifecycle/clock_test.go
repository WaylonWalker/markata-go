package lifecycle

import (
	"testing"
	"time"
)

func TestManagerBuildClockCanBeInjected(t *testing.T) {
	want := time.Date(2032, time.March, 4, 5, 6, 7, 0, time.UTC)
	m := NewManager()
	m.SetBuildClock(BuildClockFunc(func() time.Time { return want }))

	if got := m.BuildClock().Now(); !got.Equal(want) {
		t.Fatalf("build clock = %v, want %v", got, want)
	}

	m.SetBuildClock(nil)
	if m.BuildClock() == nil {
		t.Fatal("nil reset removed build clock")
	}
}

func TestManagerSerialBuildFlag(t *testing.T) {
	m := NewManager()
	if m.IsSerialBuild() {
		t.Fatal("new manager unexpectedly marked serial")
	}
	m.SetSerialBuild(true)
	if !m.IsSerialBuild() {
		t.Fatal("serial build flag was not set")
	}
}
