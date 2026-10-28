//go:build linux

package benchutil

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func ProcessUsage() (Usage, error) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return Usage{}, err
	}
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return Usage{}, err
	}
	parts := strings.Fields(string(data))
	if len(parts) < 2 {
		return Usage{}, fmt.Errorf("invalid /proc/self/statm")
	}
	pages, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return Usage{}, err
	}
	return Usage{CPUSeconds: float64(ru.Utime.Sec+ru.Stime.Sec) + float64(ru.Utime.Usec+ru.Stime.Usec)/1e6, RSSBytes: pages * uint64(os.Getpagesize()), PeakRSSBytes: uint64(ru.Maxrss) * 1024}, nil
}
