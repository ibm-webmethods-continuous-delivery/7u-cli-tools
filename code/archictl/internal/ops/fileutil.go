package ops

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a temp file in the target directory, then renames it over path.
// An existing file keeps its permission bits; a new file gets 0644.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	perm := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		perm = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, ".archictl-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// Lock takes the advisory lock <model>.lock (created exclusively) and returns its release func.
// A stale lock is reported, never removed.
func Lock(modelPath string) (func(), error) {
	lp := modelPath + ".lock"
	f, err := os.OpenFile(lp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("model is locked: %s exists (another edit running, or a stale lock to remove by hand)", lp)
		}
		return nil, err
	}
	_ = f.Close()
	return func() { _ = os.Remove(lp) }, nil
}
