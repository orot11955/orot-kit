package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallTargetsIncludeBinaryAndState(t *testing.T) {
	home := t.TempDir()
	executable := filepath.Join(home, ".local", "bin", "kit")
	targets := uninstallTargets(home, executable, uninstallOptions{})
	got := uninstallTargetPaths(targets)

	for _, want := range []string{
		filepath.Join(home, ".local", "bin", "kit"),
		filepath.Join(home, "bin", "kit"),
		filepath.Join(home, ".kit"),
		filepath.Join(home, ".kit-server"),
		"/usr/local/bin/kit",
	} {
		if !containsString(got, want) {
			t.Fatalf("targets missing %q in %#v", want, got)
		}
	}
}

func TestUninstallTargetsHonorKeepFlags(t *testing.T) {
	home := t.TempDir()
	targets := uninstallTargets(home, "", uninstallOptions{keepConfig: true, keepServer: true})
	got := uninstallTargetPaths(targets)

	for _, unexpected := range []string{filepath.Join(home, ".kit"), filepath.Join(home, ".kit-server")} {
		if containsString(got, unexpected) {
			t.Fatalf("targets should not include %q in %#v", unexpected, got)
		}
	}
}

func TestUninstallCommandUsesSafeRemoveForms(t *testing.T) {
	home := t.TempDir()
	command := uninstallCommand([]uninstallTarget{
		{Path: filepath.Join(home, "bin", "kit"), Kind: "binary"},
		{Path: filepath.Join(home, ".kit"), Kind: "state"},
	})
	got := command.String()

	if !strings.Contains(got, "rm -f") {
		t.Fatalf("binary remove missing rm -f: %q", got)
	}
	if !strings.Contains(got, "rm -rf") {
		t.Fatalf("state remove missing rm -rf: %q", got)
	}
}

func uninstallTargetPaths(targets []uninstallTarget) []string {
	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		paths = append(paths, target.Path)
	}
	return paths
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
