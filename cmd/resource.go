package cmd

import (
	"context"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

func registerResourceCommands(root *cobra.Command) {
	root.AddCommand(newResourceCommand())
	root.AddCommand(newDiskCommand())
	root.AddCommand(newMemoryCommand())
	root.AddCommand(newProcessCommand())
	root.AddCommand(newLogsCommand())
}

func newResourceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resource",
		Short: "Show server resource summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			commands := resourceCommands()
			if opts.dryRun {
				return writeDryRun(cmd, "Resource Summary", commands, []string{"kit disk", "kit memory", "kit process"})
			}
			results := runner.RunMany(context.Background(), commands)
			system := detect.System()
			summary := "Hostname: " + system.Hostname + "\nOS: " + system.OS + "\nArch: " + system.Arch
			if primary := detect.PrimaryIP(); primary != "" {
				summary += "\nPrimary IP: " + primary
			}
			return writeRunnerResults(cmd, "Resource Summary", summary, results, []string{"kit disk", "kit memory", "kit process"})
		},
	}
}

func resourceCommands() []runner.Command {
	commands := []runner.Command{
		runner.External("uname", "-a"),
		runner.External("uptime"),
		runner.External("df", "-h"),
	}
	if detect.CommandExists("free") {
		commands = append(commands, runner.External("free", "-h"))
	} else if detect.IsDarwin() {
		commands = append(commands, runner.External("vm_stat"))
	}
	commands = append(commands, runner.Shell("ps aux | head -n 10"))
	return commands
}

func newDiskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disk",
		Short: "Show disk usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.External("df", "-h")
			if opts.dryRun {
				return writeDryRun(cmd, "Disk", []runner.Command{command}, []string{"kit resource"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Disk", "Filesystem disk usage.", []runner.Result{result}, []string{"kit resource"})
		},
	}
}

func newMemoryCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "memory",
		Short: "Show memory usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.External("free", "-h")
			if detect.IsDarwin() || !detect.CommandExists("free") {
				command = runner.External("vm_stat")
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Memory", []runner.Command{command}, []string{"kit resource"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Memory", "Memory usage.", []runner.Result{result}, []string{"kit resource"})
		},
	}
}

func newProcessCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "process",
		Short: "Show top processes",
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.Shell("ps aux | head -n 15")
			if opts.dryRun {
				return writeDryRun(cmd, "Process", []runner.Command{command}, []string{"kit resource", "kit port"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Process", "Top process snapshot.", []runner.Result{result}, []string{"kit resource", "kit port"})
		},
	}
}

func newLogsCommand() *cobra.Command {
	var unit string
	command := &cobra.Command{
		Use:   "logs [service]",
		Short: "Show recent system logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				unit = args[0]
			}
			var command runner.Command
			if detect.CommandExists("journalctl") {
				if unit != "" {
					command = runner.External("journalctl", "-u", unit, "-n", "80", "--no-pager")
				} else {
					command = runner.External("journalctl", "-n", "80", "--no-pager")
				}
			} else if detect.IsDarwin() {
				command = runner.External("log", "show", "--last", "10m", "--style", "compact")
			} else {
				return writer(cmd).Write(output.Result{
					Title:   "Logs",
					Summary: "No supported log command found.",
					Hint:    []string{"kit service logs"},
				})
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Logs", []runner.Command{command}, []string{"kit service logs"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Logs", "Recent system logs.", []runner.Result{result}, []string{"kit service logs"})
		},
	}
	command.Flags().StringVar(&unit, "unit", "", "systemd unit")
	return command
}
