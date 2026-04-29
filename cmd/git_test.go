package cmd

import "testing"

func TestParseGitStatus(t *testing.T) {
	status := parseGitStatus("## main...origin/main [ahead 2, behind 1]\n M README.md\nA  cmd/git.go\n?? new.go\nUU conflict.go\n")
	if status.Branch != "main" {
		t.Fatalf("branch = %q", status.Branch)
	}
	if status.Upstream != "origin/main" {
		t.Fatalf("upstream = %q", status.Upstream)
	}
	if status.Ahead != 2 || status.Behind != 1 {
		t.Fatalf("ahead/behind = %d/%d", status.Ahead, status.Behind)
	}
	if status.Staged != 2 || status.Unstaged != 2 || status.Untracked != 1 || status.Conflicts != 1 {
		t.Fatalf("counts mismatch: %#v", status)
	}
}

func TestGitDiffCommandAgainstPath(t *testing.T) {
	command, summary, err := gitDiffCommand(gitDiffOptions{against: "main", stat: true, context: 5}, []string{"cmd"})
	if err != nil {
		t.Fatal(err)
	}
	if got := command.String(); got != "git diff --color=never --stat main -- cmd" {
		t.Fatalf("command = %q", got)
	}
	if summary != "Working tree compared against main." {
		t.Fatalf("summary = %q", summary)
	}
}

func TestGitDiffCommandRejectsConflictingOptions(t *testing.T) {
	_, _, err := gitDiffCommand(gitDiffOptions{staged: true, base: "main"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
