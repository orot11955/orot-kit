package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Manager struct {
	BaseDir string
}

type InstalledVersion struct {
	Runtime string
	Version string
	Path    string
	Current bool
}

func NewManager() (Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Manager{}, err
	}
	return Manager{BaseDir: filepath.Join(home, ".kit")}, nil
}

func NewManagerWithBase(baseDir string) Manager {
	return Manager{BaseDir: baseDir}
}

func (m Manager) RuntimeRoot() string {
	return filepath.Join(m.BaseDir, "runtimes")
}

func (m Manager) ShimsDir() string {
	return filepath.Join(m.BaseDir, "shims")
}

func (m Manager) RuntimeDir(name string) string {
	return filepath.Join(m.RuntimeRoot(), name)
}

func (m Manager) VersionDir(name string, version string) string {
	return filepath.Join(m.RuntimeDir(name), version)
}

func (m Manager) CurrentLink(name string) string {
	return filepath.Join(m.RuntimeDir(name), "current")
}

func (m Manager) Installed(name string) ([]InstalledVersion, error) {
	if !SupportedRuntime(name) {
		return nil, fmt.Errorf("unsupported runtime: %s", name)
	}
	entries, err := os.ReadDir(m.RuntimeDir(name))
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledVersion{}, nil
		}
		return nil, err
	}
	currentTarget := ""
	if target, err := os.Readlink(m.CurrentLink(name)); err == nil {
		currentTarget = filepath.Clean(target)
	}
	versions := []InstalledVersion{}
	for _, entry := range entries {
		if entry.Name() == "current" || stringsHasPrefix(entry.Name(), ".") {
			continue
		}
		if !entry.IsDir() {
			continue
		}
		path := m.VersionDir(name, entry.Name())
		current := currentTarget == entry.Name() || currentTarget == filepath.Clean(path)
		versions = append(versions, InstalledVersion{Runtime: name, Version: entry.Name(), Path: path, Current: current})
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version < versions[j].Version
	})
	return versions, nil
}

func (m Manager) CurrentVersion(name string) (InstalledVersion, bool) {
	target, err := os.Readlink(m.CurrentLink(name))
	if err != nil {
		return InstalledVersion{}, false
	}
	version := filepath.Base(target)
	if filepath.IsAbs(target) {
		version = filepath.Base(filepath.Clean(target))
	}
	path := m.VersionDir(name, version)
	if filepath.IsAbs(target) {
		path = filepath.Clean(target)
	}
	return InstalledVersion{Runtime: name, Version: version, Path: path, Current: true}, true
}

func stringsHasPrefix(value string, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}
