package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

func registerFileCommands(root *cobra.Command) {
	root.AddCommand(hiddenCommand(newListAliasCommand(".", ".")))
	root.AddCommand(hiddenCommand(newListAliasCommand("..", "..")))
	root.AddCommand(hiddenCommand(newListAliasCommand("...", "../..")))
	root.AddCommand(newLSCommand())
	root.AddCommand(newTreeCommand())
	root.AddCommand(newSizeCommand())
	root.AddCommand(newFindCommand())
}

func newListAliasCommand(use string, target string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: "List directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, target)
		},
	}
}

func newLSCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ls [path]",
		Short: "List directory with details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}
			return runList(cmd, target)
		},
	}
}

func runList(cmd *cobra.Command, target string) error {
	command := runner.External("ls", "-al", target)
	if opts.dryRun {
		return writeDryRun(cmd, "List Directory", []runner.Command{command}, []string{"kit find", "kit size"})
	}
	result := runner.Run(context.Background(), command)
	return writeRunnerResults(cmd, "List Directory", "Directory entries with permissions, owner, size, and time.", []runner.Result{result}, []string{"kit find", "kit size"})
}

func newTreeCommand() *cobra.Command {
	var depth int
	command := &cobra.Command{
		Use:   "tree [path]",
		Short: "Show a directory tree",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}
			var command runner.Command
			if detect.CommandExists("tree") {
				command = runner.External("tree", "-a", "-L", strconv.Itoa(depth), target)
			} else {
				command = runner.External("find", target, "-maxdepth", strconv.Itoa(depth), "-print")
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Directory Tree", []runner.Command{command}, []string{"kit ls", "kit find"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Directory Tree", "Directory hierarchy.", []runner.Result{result}, []string{"kit ls", "kit find"})
		},
	}
	command.Flags().IntVar(&depth, "depth", 2, "maximum tree depth")
	return command
}

func newSizeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "size [path]",
		Short: "Show file or directory size",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}
			command := runner.External("du", "-sh", target)
			if opts.dryRun {
				return writeDryRun(cmd, "Size", []runner.Command{command}, []string{"kit ls", "kit resource disk"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Size", "Disk usage summary.", []runner.Result{result}, []string{"kit ls", "kit resource disk"})
		},
	}
}

func newFindCommand() *cobra.Command {
	var rootPath string
	var name string
	var fileType string
	command := &cobra.Command{
		Use:   "find [pattern] [root]",
		Short: "Find files",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && name == "" {
				name = args[0]
			}
			if len(args) > 1 && rootPath == "." {
				rootPath = args[1]
			}

			if name == "" {
				prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
				selected, err := prompt.Select("검색 시작 위치를 선택하세요", []builder.Choice{
					{Label: ". (current directory)", Value: "."},
					{Label: "~ (home directory)", Value: "~"},
					{Label: "/ (root, permission errors will be hidden)", Value: "/"},
					{Label: "직접 입력", Value: "__custom__"},
				}, 0)
				if err != nil {
					return err
				}
				rootPath = selected
				if rootPath == "__custom__" {
					rootPath, err = prompt.Ask("검색 시작 위치", ".")
					if err != nil {
						return err
					}
				}
				name, err = prompt.Ask("찾을 이름 또는 패턴", "")
				if err != nil {
					return err
				}
				fileType, err = prompt.Select("파일 타입을 제한할까요", []builder.Choice{
					{Label: "제한 없음", Value: "any"},
					{Label: "파일만", Value: "file"},
					{Label: "디렉토리만", Value: "dir"},
				}, 0)
				if err != nil {
					return err
				}
			}

			if rootPath == "" {
				rootPath = "."
			}
			if fileType == "" {
				fileType = "any"
			}
			if name == "" {
				return fmt.Errorf("find pattern is required")
			}

			displayRoot := rootPath
			execRoot := expandHome(rootPath)
			pattern := normalizeFindPattern(name)
			parts := []string{"find", runner.Quote(execRoot)}
			switch fileType {
			case "file", "f":
				parts = append(parts, "-type f")
			case "dir", "directory", "d":
				parts = append(parts, "-type d")
			case "any":
			default:
				return fmt.Errorf("unsupported type: %s", fileType)
			}
			parts = append(parts, "-name", runner.Quote(pattern), "2>/dev/null")
			shell := strings.Join(parts, " ")
			command := runner.Shell(shell)
			summary := fmt.Sprintf("Search root: %s, pattern: %s", displayRoot, pattern)
			if rootPath == "/" {
				summary += "\nPermission errors are hidden for root searches."
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Find Files", []runner.Command{command}, []string{"kit ls", "kit tree"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Find Files", summary, []runner.Result{result}, []string{"kit ls", "kit tree"})
		},
	}
	command.Flags().StringVar(&rootPath, "root", ".", "search root")
	command.Flags().StringVar(&name, "name", "", "name or glob pattern")
	command.Flags().StringVar(&fileType, "type", "any", "any, file, or dir")
	return command
}

func expandHome(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + strings.TrimPrefix(path, "~")
		}
	}
	return path
}

func normalizeFindPattern(pattern string) string {
	if strings.ContainsAny(pattern, "*?[]") {
		return pattern
	}
	return "*" + pattern + "*"
}
