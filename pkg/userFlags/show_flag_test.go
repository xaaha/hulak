package userflags

import (
	"flag"
	"testing"
)

func TestRegisterShowFlag_DefaultFalse(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	got := registerShowFlag(fs, "reveal values")
	if got == nil {
		t.Fatal("registerShowFlag returned nil pointer")
	}
	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("parse empty args: %v", err)
	}
	if *got {
		t.Errorf("default value should be false, got true")
	}
}

func TestRegisterShowFlag_SetTrue(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	got := registerShowFlag(fs, "reveal values")
	if err := fs.Parse([]string{"--show"}); err != nil {
		t.Fatalf("parse --show: %v", err)
	}
	if !*got {
		t.Errorf("--show should set value to true, got false")
	}
}

// TestRegisterShowFlag_RegistersOnFlagSet ensures the flag is discoverable
// by name on the FlagSet (so help output includes it).
func TestRegisterShowFlag_RegistersOnFlagSet(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	_ = registerShowFlag(fs, "reveal values")
	if fs.Lookup("show") == nil {
		t.Error("expected --show flag to be registered on the FlagSet")
	}
}
