package main

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func TestGenerateFailureExit(t *testing.T) {
	if path := os.Getenv("IPINFO_TEST_GENERATE_FAILURE"); path != "" {
		os.Args = []string{"ipinfoapi", "--gen", "-c", path}
		InitOptionsArgs(0)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestGenerateFailureExit$")
	cmd.Env = append(os.Environ(), "IPINFO_TEST_GENERATE_FAILURE="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("generation failure exited successfully")
	}
	if bytes.Contains(out, []byte("has been generated")) {
		t.Fatalf("false success: %s", out)
	}
}
