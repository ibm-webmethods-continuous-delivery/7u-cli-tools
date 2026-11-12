package config

import (
	"os"
	"path/filepath"
)

// Sources reported by ResolveDataDir.
const (
	SourceEnv       = "env " + EnvDataFolder
	SourceXDG       = "env XDG_CACHE_HOME"
	SourceHomeCache = "~/.cache"
	SourceCWD       = "current directory (fallback)"
)

// DataDirResolution is the outcome of ResolveDataDir.
type DataDirResolution struct {
	Dir    string   // resolved data folder
	Source string   // which rule produced Dir
	Notes  []string // skipped rules / failures, intended for debug logging
}

// ResolveDataDir picks the data folder when --data is not given:
//  1. $IWCD_IBMDOCS_DATA_FOLDER, if set
//  2. $XDG_CACHE_HOME/ibmdocs-data, if XDG_CACHE_HOME is set (and absolute)
//  3. ~/.cache/ibmdocs-data, if it exists or can be created
//  4. ./ibmdocs-data
//
// Only rule 3 touches the filesystem.
func ResolveDataDir() DataDirResolution {
	var notes []string

	if v := os.Getenv(EnvDataFolder); v != "" {
		return DataDirResolution{Dir: v, Source: SourceEnv, Notes: notes}
	}

	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		if filepath.IsAbs(x) {
			return DataDirResolution{Dir: filepath.Join(x, DataDirName), Source: SourceXDG, Notes: notes}
		}
		notes = append(notes, "XDG_CACHE_HOME is not an absolute path, ignored: "+x)
	}

	if home, err := os.UserHomeDir(); err != nil {
		notes = append(notes, "home directory unavailable, skipping ~/.cache: "+err.Error())
	} else {
		dir := filepath.Join(home, ".cache", DataDirName)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return DataDirResolution{Dir: dir, Source: SourceHomeCache, Notes: notes}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			notes = append(notes, "cannot create "+dir+", skipping ~/.cache: "+err.Error())
		} else {
			return DataDirResolution{Dir: dir, Source: SourceHomeCache, Notes: notes}
		}
	}

	return DataDirResolution{Dir: DefaultDataDir, Source: SourceCWD, Notes: notes}
}
