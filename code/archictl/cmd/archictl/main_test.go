package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestMainInvocation(t *testing.T) {
	if os.Getenv("TEST_RUN_MAIN") == "1" {
		os.Args = []string{"archictl", "version"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMainInvocation")
	cmd.Env = append(os.Environ(), "TEST_RUN_MAIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("main execution failed: %v, output: %s", err, string(out))
	}
}
