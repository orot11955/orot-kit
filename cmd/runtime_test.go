package cmd

import (
	"bytes"
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
