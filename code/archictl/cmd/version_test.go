package cmd

import (
	"errors"
	"testing"
)

func TestRefusedErrorIsDistinguishable(t *testing.T) {
	err := refused("Composition not allowed from %s to %s", "BusinessProcess", "BusinessRole")
	var r *refusedError
	if !errors.As(err, &r) {
		t.Fatal("refused() must return a *refusedError")
	}
	if r.Error() != "Composition not allowed from BusinessProcess to BusinessRole" {
		t.Fatalf("unexpected message %q", r.Error())
	}
}

func TestVersionDefaults(t *testing.T) {
	if version == "" || buildDate == "" || gitCommit == "" {
		t.Fatal("version variables must have non-empty defaults")
	}
}
