package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/orot-dev/orot-kit/pkg/version"
)

func TestRootHelpIsGroupedAndHidesShortcuts(t *testing.T) {
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"--help"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	for _, want := range []string{
		"Files & Archives:",
		"System & Network:",
		"Development:",
		"Services & Operations:",
		"Access & Transfer:",
		"Kit:",
		"\n  network ",
		"\n  resource ",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("root help missing %q:\n%s", want, got)
		}
	}
	for _, hidden := range []string{
		"\n  . ",
		"\n  .. ",
		"\n  nginx ",
		"\n  node ",
		"\n  runtime ",
		"\n  docker ",
		"\n  ip ",
		"\n  disk ",
		"\n  install-server ",
		"\n  completion ",
	} {
		if strings.Contains(got, hidden) {
			t.Fatalf("root help should hide %q:\n%s", hidden, got)
		}
	}
}

func TestRemovedRuntimeAndDockerCommandsAreUnavailable(t *testing.T) {
	for _, name := range []string{"runtime", "node", "docker"} {
		command := NewRootCommand()
		output := &bytes.Buffer{}
		command.SetOut(output)
		command.SetErr(output)
		command.SetArgs([]string{name, "--help"})

		if err := command.Execute(); err == nil {
			t.Fatalf("%s command should be removed", name)
		}
	}
}

func TestRootVersionFlag(t *testing.T) {
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"-v"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "Kit Version") || !strings.Contains(got, version.Version) {
		t.Fatalf("version flag output = %q", got)
	}
	if strings.Contains(got, "Files & Archives:") {
		t.Fatalf("version flag should not print help:\n%s", got)
	}
}
