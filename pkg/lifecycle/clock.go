package lifecycle

import "time"

// BuildClock supplies the current time to build-time plugins.  Production
// managers use SystemBuildClock; the Build Lab can inject a fixed clock so
// generated metadata is reproducible.
type BuildClock interface {
	Now() time.Time
}

// BuildClockFunc adapts a function to BuildClock.
type BuildClockFunc func() time.Time

func (f BuildClockFunc) Now() time.Time { return f() }

type systemBuildClock struct{}

func (systemBuildClock) Now() time.Time { return time.Now() }

func newSystemBuildClock() BuildClock { return systemBuildClock{} }
