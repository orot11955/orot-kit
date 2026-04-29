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

const (
	rootGroupFiles  = "files"
	rootGroupSystem = "system"
	rootGroupDev    = "dev"
	rootGroupOps    = "ops"
	rootGroupAccess = "access"
	rootGroupKit    = "kit"
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use:           "kit",
		Short:         "Personal terminal toolkit for developers and system engineers",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return writeVersion(cmd)
			}
			return cmd.Help()
		},
	}
	configureRootHelp(root)

	root.Flags().BoolVarP(&showVersion, "version", "v", false, "print kit version")
	root.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "show command without executing")
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "print JSON output")
	root.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "print detailed command output")
	root.PersistentFlags().BoolVar(&opts.yes, "yes", false, "skip confirmation for safe commands")

	root.AddCommand(newVersionCommand())
	root.AddCommand(newInfoCommand())
	registerUninstallCommand(root)
	registerUpdateCommand(root)
	registerFileCommands(root)
	registerArchiveCommands(root)
	registerResourceCommands(root)
	registerNetworkCommands(root)
	registerGitCommands(root)
	registerServiceCommands(root)
	registerSSHCommands(root)
	registerTransferCommands(root)
	registerFirewallCommands(root)
	registerSecretCommands(root)
	registerInstallServerCommands(root)
	organizeRootCommands(root)

	return root
}

func configureRootHelp(root *cobra.Command) {
	root.CompletionOptions.HiddenDefaultCmd = true
	root.SetHelpCommandGroupID(rootGroupKit)
	root.AddGroup(
		&cobra.Group{ID: rootGroupFiles, Title: "Files & Archives:"},
		&cobra.Group{ID: rootGroupSystem, Title: "System & Network:"},
		&cobra.Group{ID: rootGroupDev, Title: "Development:"},
		&cobra.Group{ID: rootGroupOps, Title: "Services & Operations:"},
		&cobra.Group{ID: rootGroupAccess, Title: "Access & Transfer:"},
		&cobra.Group{ID: rootGroupKit, Title: "Kit:"},
	)
}

func organizeRootCommands(root *cobra.Command) {
	groups := map[string]string{
		"ls":        rootGroupFiles,
		"tree":      rootGroupFiles,
		"find":      rootGroupFiles,
		"size":      rootGroupFiles,
		"archive":   rootGroupFiles,
		"extract":   rootGroupFiles,
		"resource":  rootGroupSystem,
		"network":   rootGroupSystem,
		"git":       rootGroupDev,
		"diff":      rootGroupDev,
		"secret":    rootGroupDev,
		"service":   rootGroupOps,
		"fw":        rootGroupOps,
		"ssh":       rootGroupAccess,
		"send":      rootGroupAccess,
		"receive":   rootGroupAccess,
		"sync":      rootGroupAccess,
		"version":   rootGroupKit,
		"info":      rootGroupKit,
		"uninstall": rootGroupKit,
		"update":    rootGroupKit,
	}
	for _, command := range root.Commands() {
		if group, ok := groups[command.Name()]; ok {
			command.GroupID = group
		}
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print kit version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writeVersion(cmd)
		},
	}
}

func writeVersion(cmd *cobra.Command) error {
	return writer(cmd).Write(output.Result{
		Title:  "Kit Version",
		Result: version.Version,
	})
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
