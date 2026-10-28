//go:build windows

package benchutil

import (
	"fmt"
	"golang.org/x/sys/windows"
	"unsafe"
)

type memoryCounters struct {
	Size                                                                                                                       uint32
	PageFaultCount                                                                                                             uint32
	PeakWorkingSet, WorkingSet, PeakPagedPool, PagedPool, PeakNonPagedPool, NonPagedPool, Pagefile, PeakPagefile, PrivateUsage uintptr
}

var processMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")

func ProcessUsage() (Usage, error) {
	var created, exited, kernel, user windows.Filetime
	handle := windows.CurrentProcess()
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return Usage{}, err
	}
	ticks := func(t windows.Filetime) uint64 { return uint64(t.HighDateTime)<<32 | uint64(t.LowDateTime) }
	counters := memoryCounters{}
	counters.Size = uint32(unsafe.Sizeof(counters))
	ok, _, err := processMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
	if ok == 0 {
		return Usage{}, fmt.Errorf("GetProcessMemoryInfo: %w", err)
	}
	return Usage{CPUSeconds: float64(ticks(kernel)+ticks(user)) / 1e7, RSSBytes: uint64(counters.WorkingSet), PeakRSSBytes: uint64(counters.PeakWorkingSet)}, nil
}
