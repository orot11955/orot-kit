package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/pkg/version"
	"github.com/spf13/cobra"
)

type globalOptions struct {
	dryRun  bool
	json    bool
	verbose bool
	yes     bool
}

var opts globalOptions

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "kit",
		Short:         "Personal terminal toolkit for developers and system engineers",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "show command without executing")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "print JSON output")
	root.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "print detailed command output")
	root.PersistentFlags().BoolVar(&opts.yes, "yes", false, "skip confirmation for safe commands")

	root.AddCommand(newVersionCommand())
	root.AddCommand(newInfoCommand())
	registerUninstallCommand(root)
	registerFileCommands(root)
	registerArchiveCommands(root)
	registerResourceCommands(root)
	registerNetworkCommands(root)
	registerGitCommands(root)
	registerRuntimeCommands(root)
	registerServiceCommands(root)
	registerDockerCommands(root)
	registerSSHCommands(root)
	registerTransferCommands(root)
	registerFirewallCommands(root)
	registerSecretCommands(root)
	registerInstallServerCommands(root)

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kit version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writer(cmd).Write(output.Result{
				Title:  "Kit Version",
				Result: version.Version,
			})
		},
	}
}

func newInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Print kit build and platform information",
		RunE: func(cmd *cobra.Command, args []string) error {
			executable, _ := os.Executable()
			system := detect.System()
			return writer(cmd).Write(output.Result{
				Title: "Kit Info",
				Result: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s\nOS: %s\nArch: %s\nGo: %s\nInstall Path: %s",
					version.Version,
					version.Commit,
					version.BuildDate,
					system.OS,
					system.Arch,
					runtime.Version(),
					executable,
				),
			})
		},
	}
}

func writer(cmd *cobra.Command) output.Writer {
	return output.NewWriter(cmd.OutOrStdout(), opts.json)
}
