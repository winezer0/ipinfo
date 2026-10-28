//go:build !windows && !linux

package benchutil

import "fmt"

func ProcessUsage() (Usage, error) {
	return Usage{}, fmt.Errorf("process CPU/RSS measurement supports Windows and Linux")
}
