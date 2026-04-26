//go:build !windows

package utils

import (
	"path/filepath"
	"testing"
)

func TestUserConfigDir_AppendsProjectNameToXDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
	want := filepath.Join(xdg, ProjectName)
	if got != want {
		t.Errorf("UserConfigDir() = %q, want %q (XDG branch must append %q)", got, want, ProjectName)
	}
}

func TestUserConfigDir_FallsBackToHomeWhenXDGUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	got, err := UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
	want := filepath.Join(home, ".config", ProjectName)
	if got != want {
		t.Errorf("UserConfigDir() = %q, want %q", got, want)
	}
}

func TestUserConfigDir_ErrorsOnRelativeXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/path")

	_, err := UserConfigDir()
	if err == nil {
		t.Fatal("expected error for relative XDG_CONFIG_HOME, got nil")
	}
}
