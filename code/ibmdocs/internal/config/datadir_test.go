package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDataDir_EnvWins(t *testing.T) {
	t.Setenv(EnvDataFolder, "/custom/data")
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	r := ResolveDataDir()
	if r.Dir != "/custom/data" || r.Source != SourceEnv {
		t.Errorf("unexpected: %+v", r)
	}
}

func TestResolveDataDir_XDG(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv(EnvDataFolder, "")
	t.Setenv("XDG_CACHE_HOME", xdg)
	r := ResolveDataDir()
	if r.Dir != filepath.Join(xdg, DataDirName) || r.Source != SourceXDG {
		t.Errorf("unexpected: %+v", r)
	}
}

func TestResolveDataDir_XDGRelativeIgnored(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvDataFolder, "")
	t.Setenv("XDG_CACHE_HOME", "relative/cache")
	t.Setenv("HOME", home)
	r := ResolveDataDir()
	if r.Source != SourceHomeCache || len(r.Notes) != 1 {
		t.Errorf("unexpected: %+v", r)
	}
}

func TestResolveDataDir_HomeCacheExistingAndCreated(t *testing.T) {
	home := t.TempDir()
	t.Setenv(EnvDataFolder, "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", home)
	want := filepath.Join(home, ".cache", DataDirName)

	// not yet existing: created
	r := ResolveDataDir()
	if r.Dir != want || r.Source != SourceHomeCache {
		t.Fatalf("unexpected: %+v", r)
	}
	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Fatalf("folder not created: %v", err)
	}

	// existing: reused
	r = ResolveDataDir()
	if r.Dir != want || r.Source != SourceHomeCache {
		t.Errorf("unexpected: %+v", r)
	}
}

func TestResolveDataDir_FallbackToCWD(t *testing.T) {
	home := t.TempDir()
	// ~/.cache is a file, so ~/.cache/ibmdocs-data cannot be created
	if err := os.WriteFile(filepath.Join(home, ".cache"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataFolder, "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", home)
	r := ResolveDataDir()
	if r.Dir != DefaultDataDir || r.Source != SourceCWD || len(r.Notes) != 1 {
		t.Errorf("unexpected: %+v", r)
	}
}
