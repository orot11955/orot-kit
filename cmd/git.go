package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type gitDiffOptions struct {
	staged   bool
	stat     bool
	nameOnly bool
	against  string
	base     string
	context  int
}

func registerGitCommands(root *cobra.Command) {
	git := &cobra.Command{
		Use:   "git",
		Short: "Safe git status, position, and diff commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitStatus(cmd)
		},
	}
	git.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show safe git status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitStatus(cmd)
		},
	})
	git.AddCommand(&cobra.Command{
		Use:   "position",
		Short: "Show local and upstream git position",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitPosition(cmd)
		},
	})
	git.AddCommand(newGitDiffCommand())
	root.AddCommand(git)
	root.AddCommand(newTopLevelDiffCommand())
}

func runGitStatus(cmd *cobra.Command) error {
	commands := []runner.Command{
		runner.External("git", "status", "--porcelain=v1", "--branch"),
		runner.External("git", "log", "--oneline", "--decorate", "-n", "5", "--graph", "--all"),
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Git Status", commands, []string{"kit git position", "kit git diff"})
	}
	ok, err := ensureGitRepository(cmd, "Git Status")
	if err != nil || !ok {
		return err
	}
	results := runner.RunMany(context.Background(), commands)
	status := parseGitStatus(results[0].Stdout)
	summary := status.Summary()
	return writer(cmd).Write(output.Result{
		Title:   "Git Status",
		Command: commandStrings(commands),
		Summary: summary,
		Sections: []output.Section{
			{Name: "Status", Text: emptyFallback(results[0].Stdout, "Working tree is clean.")},
			{Name: "Recent Log", Text: emptyFallback(results[1].Stdout, "(no commits)")},
		},
		Hint: []string{"kit git position", "kit git diff", "kit diff <file-a> <file-b>"},
	})
}

func runGitPosition(cmd *cobra.Command) error {
	commands := []runner.Command{
		runner.External("git", "rev-parse", "--abbrev-ref", "HEAD"),
		runner.External("git", "log", "-1", "--pretty=%h %s"),
		runner.External("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"),
		runner.External("git", "log", "-1", "--pretty=%h %s", "@{u}"),
		runner.External("git", "rev-list", "--left-right", "--count", "HEAD...@{u}"),
		runner.External("git", "log", "--oneline", "--decorate", "-n", "8", "--graph", "--all"),
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Git Position", commands, []string{"kit git status", "kit git diff --stat"})
	}
	ok, err := ensureGitRepository(cmd, "Git Position")
	if err != nil || !ok {
		return err
	}
	results := runner.RunMany(context.Background(), commands)
	branch := cleanGitLine(results[0].Stdout)
	local := cleanGitLine(results[1].Stdout)
	upstream := cleanGitLine(results[2].Stdout)
	remote := cleanGitLine(results[3].Stdout)
	ahead, behind, relation := parseAheadBehind(results[4].Stdout)
	if results[2].Err != nil {
		upstream = "not configured"
		remote = "not available"
		relation = "No upstream branch configured"
	} else if relation == "" {
		relation = describeAheadBehind(ahead, behind)
	}
	summary := fmt.Sprintf("Branch: %s\nLocal: %s\nUpstream: %s\nRemote: %s\nRelation: %s", branch, local, upstream, remote, relation)
	return writer(cmd).Write(output.Result{
		Title:   "Git Position",
		Command: commandStrings(commands),
		Summary: summary,
		Sections: []output.Section{
			{Name: "Recent Graph", Text: emptyFallback(results[5].Stdout, "(no commits)")},
		},
		Hint: []string{"kit git status", "kit git diff --stat"},
	})
}

func newGitDiffCommand() *cobra.Command {
	var diffOptions gitDiffOptions
	command := &cobra.Command{
		Use:   "diff [paths...]",
		Short: "Compare code using git diff without changing the repository",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGitDiff(cmd, diffOptions, args)
		},
	}
	command.Flags().BoolVar(&diffOptions.staged, "staged", false, "compare staged changes")
	command.Flags().BoolVar(&diffOptions.stat, "stat", false, "show diff stat")
	command.Flags().BoolVar(&diffOptions.nameOnly, "name-only", false, "show changed file names only")
	command.Flags().StringVar(&diffOptions.against, "against", "", "compare working tree or HEAD against a ref")
	command.Flags().StringVar(&diffOptions.base, "base", "", "compare base...HEAD")
	command.Flags().IntVarP(&diffOptions.context, "context", "U", 3, "number of context lines")
	return command
}

func runGitDiff(cmd *cobra.Command, diffOptions gitDiffOptions, paths []string) error {
	command, summary, err := gitDiffCommand(diffOptions, paths)
	if err != nil {
		return err
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Git Diff", []runner.Command{command}, []string{"kit git status", "kit git diff --stat"})
	}
	ok, err := ensureGitRepository(cmd, "Git Diff")
	if err != nil || !ok {
		return err
	}
	result := runner.Run(context.Background(), command)
	return writeRunnerResults(cmd, "Git Diff", summary, []runner.Result{result}, []string{"kit git status", "kit git diff --stat"})
}

func newTopLevelDiffCommand() *cobra.Command {
	var contextLines int
	command := &cobra.Command{
		Use:   "diff <file-a> <file-b>",
		Short: "Compare two files with unified diff",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !detect.CommandExists("diff") {
				return writer(cmd).Write(output.Result{Title: "Diff", Summary: "diff command not found."})
			}
			command := runner.External("diff", "-U", strconv.Itoa(contextLines), args[0], args[1])
			if opts.dryRun {
				return writeDryRun(cmd, "Diff", []runner.Command{command}, []string{"kit git diff"})
			}
			result := runner.Run(context.Background(), command)
			if result.ExitCode == 1 {
				result.Err = nil
			}
			summary := "Unified file comparison. Exit code 1 means files differ."
			return writeRunnerResults(cmd, "Diff", summary, []runner.Result{result}, []string{"kit git diff"})
		},
	}
	command.Flags().IntVarP(&contextLines, "context", "U", 3, "number of context lines")
	return command
}

func ensureGitRepository(cmd *cobra.Command, title string) (bool, error) {
	if !detect.CommandExists("git") {
		return false, writer(cmd).Write(output.Result{Title: title, Summary: "git command not found."})
	}
	result := runner.Run(context.Background(), runner.External("git", "rev-parse", "--is-inside-work-tree"))
	if result.Err != nil || strings.TrimSpace(result.Stdout) != "true" {
		return false, writer(cmd).Write(output.Result{Title: title, Command: []string{result.Command}, Summary: "Current directory is not inside a git repository."})
	}
	return true, nil
}

func gitDiffCommand(diffOptions gitDiffOptions, paths []string) (runner.Command, string, error) {
	args := []string{"diff", "--color=never"}
	if diffOptions.stat && diffOptions.nameOnly {
		return runner.Command{}, "", fmt.Errorf("--stat and --name-only cannot be used together")
	}
	if diffOptions.stat {
		args = append(args, "--stat")
	}
	if diffOptions.nameOnly {
		args = append(args, "--name-only")
	}
	if diffOptions.context >= 0 && !diffOptions.stat && !diffOptions.nameOnly {
		args = append(args, "-U"+strconv.Itoa(diffOptions.context))
	}
	summary := "Working tree changes compared to the index."
	switch {
	case diffOptions.staged && diffOptions.base != "":
		return runner.Command{}, "", fmt.Errorf("--staged and --base cannot be used together")
	case diffOptions.staged && diffOptions.against != "":
		return runner.Command{}, "", fmt.Errorf("--staged and --against cannot be used together")
	case diffOptions.staged:
		args = append(args, "--cached")
		summary = "Staged changes compared to HEAD."
	case diffOptions.base != "":
		args = append(args, diffOptions.base+"...HEAD")
		summary = "HEAD compared to merge-base with " + diffOptions.base + "."
	case diffOptions.against != "":
		args = append(args, diffOptions.against)
		summary = "Working tree compared against " + diffOptions.against + "."
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return runner.External("git", args...), summary, nil
}

type gitStatusSummary struct {
	Branch    string
	Upstream  string
	Ahead     int
	Behind    int
	Staged    int
	Unstaged  int
	Untracked int
	Conflicts int
	Files     int
}

func parseGitStatus(raw string) gitStatusSummary {
	summary := gitStatusSummary{Branch: "unknown"}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			parseGitBranchLine(strings.TrimPrefix(line, "## "), &summary)
			continue
		}
		summary.Files++
		if strings.HasPrefix(line, "??") {
			summary.Untracked++
			continue
		}
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]
		if isConflictStatus(line[:2]) {
			summary.Conflicts++
		}
		if x != ' ' {
			summary.Staged++
		}
		if y != ' ' {
			summary.Unstaged++
		}
	}
	return summary
}

func isConflictStatus(status string) bool {
	switch status {
	case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
		return true
	default:
		return false
	}
}

func parseGitBranchLine(line string, summary *gitStatusSummary) {
	relation := ""
	if index := strings.Index(line, " ["); index >= 0 {
		relation = strings.TrimSuffix(line[index+2:], "]")
		line = line[:index]
	}
	if parts := strings.SplitN(line, "...", 2); len(parts) == 2 {
		summary.Branch = parts[0]
		summary.Upstream = parts[1]
	} else {
		summary.Branch = line
	}
	for _, part := range strings.Split(relation, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ahead ") {
			summary.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		}
		if strings.HasPrefix(part, "behind ") {
			summary.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
}

func (s gitStatusSummary) Summary() string {
	lines := []string{"Branch: " + s.Branch}
	if s.Upstream != "" {
		lines = append(lines, "Upstream: "+s.Upstream)
		lines = append(lines, "Relation: "+describeAheadBehind(s.Ahead, s.Behind))
	} else {
		lines = append(lines, "Upstream: not configured")
	}
	lines = append(lines, fmt.Sprintf("Changed files: %d", s.Files))
	lines = append(lines, fmt.Sprintf("Staged: %d", s.Staged))
	lines = append(lines, fmt.Sprintf("Unstaged: %d", s.Unstaged))
	lines = append(lines, fmt.Sprintf("Untracked: %d", s.Untracked))
	if s.Conflicts > 0 {
		lines = append(lines, fmt.Sprintf("Conflicts: %d", s.Conflicts))
	}
	return strings.Join(lines, "\n")
}

func parseAheadBehind(raw string) (int, int, string) {
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return 0, 0, ""
	}
	ahead, errAhead := strconv.Atoi(fields[0])
	behind, errBehind := strconv.Atoi(fields[1])
	if errAhead != nil || errBehind != nil {
		return 0, 0, ""
	}
	return ahead, behind, describeAheadBehind(ahead, behind)
}

func describeAheadBehind(ahead int, behind int) string {
	switch {
	case ahead == 0 && behind == 0:
		return "Local and upstream are aligned"
	case ahead > 0 && behind > 0:
		return fmt.Sprintf("Local is %d ahead and %d behind", ahead, behind)
	case ahead > 0:
		return fmt.Sprintf("Local is %d ahead", ahead)
	default:
		return fmt.Sprintf("Local is %d behind", behind)
	}
}

func cleanGitLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "not available"
	}
	return value
}

func emptyFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
