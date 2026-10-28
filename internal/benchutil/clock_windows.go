//go:build windows

package benchutil

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var performanceCounter = windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryPerformanceCounter")
var counterFrequency = func() int64 {
	var frequency int64
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryPerformanceFrequency")
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&frequency)))
	if ok == 0 || frequency <= 0 {
		panic("QueryPerformanceFrequency unavailable")
	}
	return frequency
}()

const ClockName = "Windows QueryPerformanceCounter"

func Tick() int64 {
	var counter int64
	ok, _, _ := performanceCounter.Call(uintptr(unsafe.Pointer(&counter)))
	if ok == 0 {
		panic("QueryPerformanceCounter unavailable")
	}
	return counter
}

func Since(start int64) time.Duration {
	delta := Tick() - start
	return time.Duration(delta/counterFrequency)*time.Second + time.Duration((delta%counterFrequency)*int64(time.Second)/counterFrequency)
}
