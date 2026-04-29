package cmd

import (
	"bytes"
	"strings"
	"testing"
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

func TestHiddenShortcutCommandsStillExecuteHelp(t *testing.T) {
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"node", "--help"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "Manage node runtime") {
		t.Fatalf("hidden node shortcut help was not reachable:\n%s", got)
	}
}
