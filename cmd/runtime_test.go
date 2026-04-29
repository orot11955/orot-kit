package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	kitruntime "github.com/orot-dev/orot-kit/internal/runtime"
)

func TestRuntimeInstallCommandStringUsesCurlForServerDownloads(t *testing.T) {
	manager := kitruntime.NewManagerWithBase(t.TempDir())
	got := runtimeInstallCommandString(manager, kitruntime.InstallRequest{
		Runtime:        "node",
		Version:        "22.3.0",
		RuntimeBaseURL: "https://kit.local/runtime",
		OS:             "linux",
		Arch:           "amd64",
	})

	for _, want := range []string{
		"curl -fL --retry 2 --connect-timeout 10 --output '<temp-file>' https://kit.local/runtime/node/22.3.0/linux/amd64",
		"extract '<temp-file>'",
		manager.VersionDir("node", "22.3.0"),
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("command preview missing %q in %q", want, got)
		}
	}
}

func TestRuntimeInstallCommandStringUsesLocalSourceDirectly(t *testing.T) {
	manager := kitruntime.NewManagerWithBase(t.TempDir())
	got := runtimeInstallCommandString(manager, kitruntime.InstallRequest{
		Runtime: "node",
		Version: "22.3.0",
		Source:  "./node.tar.gz",
	})

	if strings.Contains(got, "curl ") {
		t.Fatalf("local install preview should not use curl: %q", got)
	}
	if !strings.Contains(got, "extract ./node.tar.gz") {
		t.Fatalf("local install preview missing source: %q", got)
	}
}

func TestRunRuntimeInstallRejectsConflictingSources(t *testing.T) {
	command := &cobra.Command{}
	command.SetOut(&bytes.Buffer{})
	err := runRuntimeInstall(command, "node", "22.3.0", runtimeInstallOptions{
		source:        "./node.tar.gz",
		serverBaseURL: "https://kit.local/runtime",
	})
	if err == nil || !strings.Contains(err.Error(), "--from and --from-server") {
		t.Fatalf("expected conflicting source error, got %v", err)
	}
}

func TestRunRuntimeListDetectsCurrentFromRuntimeCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	fakeBin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fakeBin, "node"), []byte("#!/usr/bin/env sh\nprintf 'v22.3.0\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	manager := kitruntime.NewManagerWithBase(filepath.Join(home, ".kit"))
	if err := os.MkdirAll(manager.VersionDir("node", "22.3.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	command := &cobra.Command{}
	command.SetOut(&output)
	if err := runRuntimeList(command, "node"); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if strings.Contains(got, "ls ~/.kit/runtimes") {
		t.Fatalf("runtime list should not preview ls: %s", got)
	}
	for _, want := range []string{"node -v", "which node", "* node 22.3.0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("runtime list output missing %q in %s", want, got)
		}
	}
}

func TestRuntimeVersionMatchesCommandOutput(t *testing.T) {
	cases := []struct {
		name      string
		detected  string
		installed string
	}{
		{name: "node", detected: "v22.3.0", installed: "22.3.0"},
		{name: "go", detected: "go version go1.23.0 linux/amd64", installed: "1.23.0"},
		{name: "python", detected: "Python 3.12.3", installed: "3.12.3"},
		{name: "java", detected: `openjdk version "21.0.2" 2024-01-16`, installed: "21"},
	}
	for _, tc := range cases {
		if !runtimeVersionMatches(tc.name, tc.detected, tc.installed) {
			t.Fatalf("expected %s %q to match %q", tc.name, tc.detected, tc.installed)
		}
	}
}
