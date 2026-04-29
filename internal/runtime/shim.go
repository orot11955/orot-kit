package runtime

import (
	"fmt"
	"os"
	"path/filepath"
)

type Shim struct {
	Name   string
	Target string
}

func (m Manager) Use(name string, version string) ([]Shim, error) {
	if !SupportedRuntime(name) {
		return nil, fmt.Errorf("unsupported runtime: %s", name)
	}
	versionDir := m.VersionDir(name, version)
	if info, err := os.Stat(versionDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s %s is not installed at %s", name, version, versionDir)
	}
	if err := os.MkdirAll(m.RuntimeDir(name), 0o755); err != nil {
		return nil, err
	}
	if err := os.Remove(m.CurrentLink(name)); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.Symlink(version, m.CurrentLink(name)); err != nil {
		return nil, err
	}
	return m.EnsureShims(name)
}

func (m Manager) EnsureShims(name string) ([]Shim, error) {
	spec, ok := Specs()[name]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime: %s", name)
	}
	if err := os.MkdirAll(m.ShimsDir(), 0o755); err != nil {
		return nil, err
	}
	shims := make([]Shim, 0, len(spec.ShimNames))
	for _, shimName := range spec.ShimNames {
		target := filepath.Join(m.CurrentLink(name), "bin", shimName)
		shimPath := filepath.Join(m.ShimsDir(), shimName)
		script := fmt.Sprintf("#!/usr/bin/env sh\nexec %q \"$@\"\n", target)
		if err := os.WriteFile(shimPath, []byte(script), 0o755); err != nil {
			return nil, err
		}
		shims = append(shims, Shim{Name: shimName, Target: target})
	}
	return shims, nil
}
