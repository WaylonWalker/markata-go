//go:build !linux

package buildstats

import "time"

func readRuntimeSample() runtimeSample {
	return runtimeSample{At: time.Now()}
}
