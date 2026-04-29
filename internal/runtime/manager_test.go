package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUseUpdatesCurrentAndShims(t *testing.T) {
	base := t.TempDir()
	manager := NewManagerWithBase(base)
	versionDir := manager.VersionDir("node", "1.0.0")
	if err := os.MkdirAll(filepath.Join(versionDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(versionDir, "bin", "node"), []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	shims, err := manager.Use("node", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	current, ok := manager.CurrentVersion("node")
	if !ok || current.Version != "1.0.0" {
		t.Fatalf("current = %#v, ok=%t", current, ok)
	}
	if len(shims) == 0 {
		t.Fatal("expected shims")
	}
	shim, err := os.ReadFile(filepath.Join(manager.ShimsDir(), "node"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shim), filepath.Join(manager.CurrentLink("node"), "bin", "node")) {
		t.Fatalf("shim target missing: %s", shim)
	}
}

func TestInstallFromDirectoryPreservesBin(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "source")
	if err := os.MkdirAll(filepath.Join(source, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "bin", "node"), []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager := NewManagerWithBase(filepath.Join(base, "kit"))
	result, err := manager.Install(context.Background(), InstallRequest{
		Runtime: "node",
		Version: "1.0.0",
		Source:  source,
		Use:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.Path != manager.VersionDir("node", "1.0.0") {
		t.Fatalf("path = %s", result.Plan.Path)
	}
	if _, err := os.Stat(filepath.Join(result.Plan.Path, "bin", "node")); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveRefusesCurrentVersion(t *testing.T) {
	base := t.TempDir()
	manager := NewManagerWithBase(base)
	versionDir := manager.VersionDir("node", "1.0.0")
	if err := os.MkdirAll(filepath.Join(versionDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Use("node", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := manager.Remove("node", "1.0.0"); err == nil {
		t.Fatal("expected current remove error")
	}
}

func TestCurlDownloadArgs(t *testing.T) {
	args := strings.Join(CurlDownloadArgs("/tmp/runtime.tar.gz", "https://kit.local/runtime/node"), " ")
	for _, want := range []string{"-fL", "--retry 2", "--connect-timeout 10", "--output /tmp/runtime.tar.gz", "https://kit.local/runtime/node"} {
		if !strings.Contains(args, want) {
			t.Fatalf("curl args missing %q in %q", want, args)
		}
	}
}
