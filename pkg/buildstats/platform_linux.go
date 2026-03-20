//go:build linux

package buildstats

import (
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func readRuntimeSample() runtimeSample {
	sample := runtimeSample{At: time.Now()}

	var usage unix.Rusage
	if err := unix.Getrusage(unix.RUSAGE_SELF, &usage); err == nil {
		sample.ProcessCPU = timevalDuration(usage.Utime) + timevalDuration(usage.Stime)
	}

	if data, err := os.ReadFile("/proc/self/io"); err == nil {
		sample.ReadBytes, sample.WriteBytes = parseProcIO(data)
	}

	return sample
}

func timevalDuration(tv unix.Timeval) time.Duration {
	return time.Duration(tv.Sec)*time.Second + time.Duration(tv.Usec)*time.Microsecond
}

func parseProcIO(data []byte) (uint64, uint64) {
	var readBytes uint64
	var writeBytes uint64

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil {
			continue
		}

		switch strings.TrimSpace(key) {
		case "read_bytes":
			readBytes = parsed
		case "write_bytes":
			writeBytes = parsed
		}
	}

	return readBytes, writeBytes
}
