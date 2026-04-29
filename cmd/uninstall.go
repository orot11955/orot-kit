package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type uninstallOptions struct {
	binPath    string
	keepConfig bool
	keepServer bool
}

type uninstallTarget struct {
	Path string
	Kind string
}

func registerUninstallCommand(root *cobra.Command) {
	options := uninstallOptions{}
	command := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove kit and local kit state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cmd, options)
		},
	}
	command.Flags().StringVar(&options.binPath, "bin", "", "additional kit binary path to remove")
	command.Flags().BoolVar(&options.keepConfig, "keep-config", false, "keep ~/.kit")
	command.Flags().BoolVar(&options.keepServer, "keep-server", false, "keep ~/.kit-server")
	root.AddCommand(command)
}

func runUninstall(cmd *cobra.Command, options uninstallOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	executable, _ := os.Executable()
	targets := uninstallTargets(home, executable, options)
	command := uninstallCommand(targets)
	if opts.dryRun {
		return writeDryRun(cmd, "Uninstall", []runner.Command{command}, []string{"kit uninstall --yes"})
	}
	if !opts.yes {
		ok, err := confirmExecution(cmd, "Uninstall", command, uninstallSummary(targets), false)
		if err != nil {
			return err
		}
		if !ok {
			return canceled(cmd, "Uninstall", command)
		}
	}
	removed := []string{}
	skipped := []string{}
	failed := []string{}
	for _, target := range targets {
		if target.Path == "" {
			continue
		}
		if _, err := os.Lstat(target.Path); os.IsNotExist(err) {
			skipped = append(skipped, target.Path)
			continue
		} else if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", target.Path, err))
			continue
		}
		if err := os.RemoveAll(target.Path); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", target.Path, err))
			continue
		}
		removed = append(removed, target.Path)
	}
	sections := []output.Section{
		{Name: "Removed", Rows: uninstallRowsOrFallback(removed, "(nothing removed)")},
		{Name: "Skipped", Rows: uninstallRowsOrFallback(skipped, "(nothing skipped)")},
	}
	if len(failed) > 0 {
		sections = append(sections, output.Section{Name: "Failed", Rows: failed})
	}
	summary := "kit binary and local kit state were removed."
	if len(failed) > 0 {
		summary = "Uninstall finished, but some paths could not be removed."
	}
	return writer(cmd).Write(output.Result{
		Title:    "Uninstall",
		Command:  []string{command.String()},
		Summary:  summary,
		Sections: sections,
		Hint:     []string{"Remove ~/.kit/shims from PATH if it was added to your shell profile."},
	})
}

func uninstallTargets(home string, executable string, options uninstallOptions) []uninstallTarget {
	targets := []uninstallTarget{}
	add := func(path string, kind string) {
		if path == "" {
			return
		}
		path = expandHome(path)
		clean := filepath.Clean(path)
		for _, target := range targets {
			if target.Path == clean {
				return
			}
		}
		targets = append(targets, uninstallTarget{Path: clean, Kind: kind})
	}

	add(options.binPath, "binary")
	add(executable, "binary")
	add(filepath.Join(home, ".local", "bin", "kit"), "binary")
	add(filepath.Join(home, "bin", "kit"), "binary")
	add("/usr/local/bin/kit", "binary")

	if !options.keepConfig {
		add(filepath.Join(home, ".kit"), "state")
	}
	if !options.keepServer {
		add(filepath.Join(home, ".kit-server"), "state")
	}
	return targets
}

func uninstallCommand(targets []uninstallTarget) runner.Command {
	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		switch target.Kind {
		case "binary":
			parts = append(parts, "rm -f "+runner.Quote(target.Path))
		default:
			parts = append(parts, "rm -rf "+runner.Quote(target.Path))
		}
	}
	if len(parts) == 0 {
		return runner.Shell("true")
	}
	return runner.Shell(strings.Join(parts, " && "))
}

func uninstallSummary(targets []uninstallTarget) string {
	rows := make([]string, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, fmt.Sprintf("%s: %s", target.Kind, target.Path))
	}
	return "The following kit paths will be removed:\n" + strings.Join(rows, "\n")
}

func uninstallRowsOrFallback(values []string, fallback string) []string {
	if len(values) == 0 {
		return []string{fallback}
	}
	return values
}
