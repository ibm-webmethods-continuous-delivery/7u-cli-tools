package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("rootCmd.Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "archictl") {
		t.Errorf("expected output to contain 'archictl', got: %s", out)
	}
	if !strings.Contains(out, "version") {
		t.Errorf("expected output to list 'version' command, got: %s", out)
	}
}

func TestVersionCommandOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("version command execution failed: %v", err)
	}
}
