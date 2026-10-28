//go:build !windows

package benchutil

import "time"

var clockOrigin = time.Now()

const ClockName = "Go monotonic clock"

func Tick() int64                     { return time.Since(clockOrigin).Nanoseconds() }
func Since(start int64) time.Duration { return time.Duration(Tick() - start) }
