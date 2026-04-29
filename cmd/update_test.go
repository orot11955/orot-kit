package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpdateDownloadURLAddsUpdateQuery(t *testing.T) {
	got := updateDownloadURL("https://kit.local", "kit-linux-amd64")
	want := "https://kit.local/bin/kit-linux-amd64?update=1"
	if got != want {
		t.Fatalf("update URL = %q, want %q", got, want)
	}
}

func TestKitBinaryName(t *testing.T) {
	got, err := kitBinaryName("linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if got != "kit-linux-arm64" {
		t.Fatalf("binary name = %q", got)
	}
	if _, err := kitBinaryName("windows", "amd64"); err == nil {
		t.Fatal("unsupported OS should fail")
	}
}

func TestUpdateDryRunUsesUpdateDownloadURL(t *testing.T) {
	previous := opts
	opts = globalOptions{}
	t.Cleanup(func() { opts = previous })

	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"--dry-run", "update", "--base-url", "https://kit.local", "--bin", "/tmp/kit"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"Update",
		"https://kit.local/bin/kit-",
		"?update=1",
		"mv '<temp-file>' /tmp/kit",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, got)
		}
	}
}
