package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/config"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	kitruntime "github.com/orot-dev/orot-kit/internal/runtime"
	"github.com/spf13/cobra"
)

type runtimeInstallOptions struct {
	source        string
	serverBaseURL string
	sha256        string
	use           bool
}

func registerRuntimeCommands(root *cobra.Command) {
	runtimeCmd := &cobra.Command{
		Use:   "runtime",
		Short: "Manage development runtimes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return writer(cmd).Write(output.Result{
				Title:   "Runtime",
				Summary: "Supported runtimes: " + strings.Join(kitruntime.Supported, ", "),
				Hint:    []string{"kit runtime list", "kit node current", "kit node use <version>"},
			})
		},
	}
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "list [runtime]",
		Short: "List installed runtime versions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runRuntimeList(cmd, name)
		},
	})
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "available [runtime]",
		Short: "List seed installable runtime versions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			return runRuntimeAvailable(cmd, name)
		},
	})
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "current <runtime>",
		Short: "Detect current runtime version from the executable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeCurrent(cmd, args[0])
		},
	})
	runtimeCmd.AddCommand(newRuntimeInstallCommand("install <runtime> [version]", "", true))
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "use <runtime> [version]",
		Short: "Switch current runtime symlink and shims",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 1 {
				version = args[1]
			}
			return runRuntimeUse(cmd, args[0], version)
		},
	})
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "remove <runtime> <version>",
		Short: "Remove an installed runtime version",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeRemove(cmd, args[0], args[1])
		},
	})
	runtimeCmd.AddCommand(&cobra.Command{
		Use:   "cache [runtime] [version]",
		Short: "Show runtime server cache download endpoint",
		Args:  cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "node"
			version := "22.3.0"
			if len(args) > 0 {
				name = args[0]
			}
			if len(args) > 1 {
				version = args[1]
			}
			return runRuntimeCache(cmd, name, version)
		},
	})
	runtimeCmd.AddCommand(newRuntimeServeCommand())
	root.AddCommand(runtimeCmd)

	for _, name := range kitruntime.Supported {
		root.AddCommand(newRuntimeAliasCommand(name))
	}
}

func newRuntimeAliasCommand(name string) *cobra.Command {
	command := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Manage %s runtime", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeCurrent(cmd, name)
		},
	}
	command.AddCommand(&cobra.Command{
		Use:   "current",
		Short: "Detect current version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeCurrent(cmd, name)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeList(cmd, name)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "available",
		Short: "List seed installable versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeAvailable(cmd, name)
		},
	})
	command.AddCommand(newRuntimeInstallCommand("install [version]", name, false))
	command.AddCommand(&cobra.Command{
		Use:   "use [version]",
		Short: "Switch runtime version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			return runRuntimeUse(cmd, name, version)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "remove <version>",
		Short: "Remove runtime version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuntimeRemove(cmd, name, args[0])
		},
	})
	return command
}

func newRuntimeInstallCommand(use string, fixedRuntime string, runtimeArg bool) *cobra.Command {
	options := runtimeInstallOptions{use: true}
	command := &cobra.Command{
		Use:   use,
		Short: "Install a runtime from a local archive/directory or kit runtime server",
		Args: func(cmd *cobra.Command, args []string) error {
			if runtimeArg {
				return cobra.RangeArgs(1, 2)(cmd, args)
			}
			return cobra.MaximumNArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := fixedRuntime
			version := ""
			if runtimeArg {
				name = args[0]
				if len(args) > 1 {
					version = args[1]
				}
			} else if len(args) > 0 {
				version = args[0]
			}
			return runRuntimeInstall(cmd, name, version, options)
		},
	}
	command.Flags().StringVar(&options.source, "from", "", "local archive, local directory, or URL source")
	command.Flags().StringVar(&options.serverBaseURL, "from-server", "", "kit runtime server base URL")
	command.Flags().StringVar(&options.sha256, "sha256", "", "expected source SHA256")
	command.Flags().BoolVar(&options.use, "use", true, "switch to this version after install")
	return command
}

func runRuntimeCurrent(cmd *cobra.Command, name string) error {
	manager, err := kitruntime.NewManager()
	if err != nil {
		return err
	}
	current := kitruntime.DetectCurrent(context.Background(), name)
	linked, hasLinked := manager.CurrentVersion(name)
	versionCommand := runtimeVersionCommand(name)
	if current.Version == "" {
		summary := "Runtime executable was not found in PATH."
		if hasLinked {
			summary += "\nKit current: " + linked.Version + "\nKit path: " + linked.Path
		}
		return writer(cmd).Write(output.Result{
			Title:   titleAction(name) + " Current",
			Command: []string{versionCommand, "which " + name},
			Summary: summary,
			Hint:    []string{"kit " + name + " list", "kit " + name + " use <version>"},
		})
	}
	result := "Version: " + current.Version + "\nPath: " + current.Path
	if hasLinked {
		result += "\nKit Current: " + linked.Version + "\nKit Path: " + linked.Path
	}
	return writer(cmd).Write(output.Result{
		Title:   titleAction(name) + " Current",
		Command: []string{versionCommand, "which " + name},
		Result:  result,
		Hint:    []string{"kit " + name + " list", "kit " + name + " use <version>"},
	})
}

func runRuntimeList(cmd *cobra.Command, name string) error {
	manager, err := kitruntime.NewManager()
	if err != nil {
		return err
	}
	names := []string{name}
	if name == "" {
		names = kitruntime.Supported
	}
	rows := []string{}
	for _, runtimeName := range names {
		versions, err := manager.Installed(runtimeName)
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			rows = append(rows, runtimeName+": (none)")
			continue
		}
		for _, version := range versions {
			marker := " "
			if version.Current {
				marker = "*"
			}
			rows = append(rows, fmt.Sprintf("%s %s %s  %s", marker, runtimeName, version.Version, version.Path))
		}
	}
	return writer(cmd).Write(output.Result{
		Title:   "Runtime List",
		Command: []string{"ls ~/.kit/runtimes"},
		Result:  strings.Join(rows, "\n"),
		Hint:    []string{"kit node available", "kit node use <version>"},
	})
}

func runRuntimeAvailable(cmd *cobra.Command, name string) error {
	specs := kitruntime.Specs()
	names := []string{name}
	if name == "" {
		names = kitruntime.Supported
	}
	rows := []string{}
	for _, runtimeName := range names {
		spec, ok := specs[runtimeName]
		if !ok {
			return fmt.Errorf("unsupported runtime: %s", runtimeName)
		}
		rows = append(rows, runtimeName+": "+strings.Join(spec.SeedVersions, ", "))
	}
	return writer(cmd).Write(output.Result{
		Title:   "Runtime Available",
		Command: []string{"kit runtime available"},
		Summary: "Seed registry versions. Use --from or runtime_base_url for actual install sources.",
		Result:  strings.Join(rows, "\n"),
		Hint:    []string{"kit runtime cache node 22.3.0", "kit node install 22.3.0 --from ./runtime.tar.gz"},
	})
}

func runRuntimeUse(cmd *cobra.Command, name string, version string) error {
	manager, err := kitruntime.NewManager()
	if err != nil {
		return err
	}
	if version == "" {
		version, err = selectInstalledRuntimeVersion(cmd, manager, name)
		if err != nil {
			return err
		}
	}
	command := runner.Shell("ln -sfn " + runner.Quote(version) + " " + runner.Quote(manager.CurrentLink(name)) + " && write shims " + runner.Quote(manager.ShimsDir()))
	if opts.dryRun {
		return writeDryRun(cmd, titleAction(name)+" Use", []runner.Command{command}, []string{"export PATH=\"$HOME/.kit/shims:$PATH\"", "kit " + name + " current"})
	}
	shims, err := manager.Use(name, version)
	if err != nil {
		return err
	}
	rows := []string{"Current: " + version, "Path: " + manager.VersionDir(name, version)}
	for _, shim := range shims {
		rows = append(rows, "Shim: "+shim.Name+" -> "+shim.Target)
	}
	return writer(cmd).Write(output.Result{
		Title:   titleAction(name) + " Use",
		Command: []string{command.String()},
		Summary: "Runtime current symlink and shims updated.",
		Result:  strings.Join(rows, "\n"),
		Hint:    []string{"export PATH=\"$HOME/.kit/shims:$PATH\"", "kit " + name + " current"},
	})
}

func runRuntimeInstall(cmd *cobra.Command, name string, version string, options runtimeInstallOptions) error {
	if name == "" {
		return fmt.Errorf("runtime name is required")
	}
	if options.source != "" && options.serverBaseURL != "" {
		return fmt.Errorf("--from and --from-server cannot be used together")
	}
	var err error
	if version == "" {
		version, err = selectAvailableRuntimeVersion(cmd, name)
		if err != nil {
			return err
		}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	baseURL := runtimeBaseURL(cfg)
	if options.serverBaseURL != "" {
		baseURL = strings.TrimRight(options.serverBaseURL, "/")
	}
	manager, err := kitruntime.NewManager()
	if err != nil {
		return err
	}
	request := kitruntime.InstallRequest{
		Runtime: name, Version: version, Source: options.source,
		RuntimeBaseURL: baseURL, OS: stdruntime.GOOS, Arch: stdruntime.GOARCH,
		SHA256: options.sha256, Use: options.use,
	}
	planCommand := runtimeInstallCommandString(manager, request)
	if opts.dryRun {
		return writeDryRun(cmd, titleAction(name)+" Install", []runner.Command{runner.Shell(planCommand)}, []string{"kit " + name + " list", "kit " + name + " use " + version})
	}
	result, err := manager.Install(context.Background(), request)
	if err != nil {
		return err
	}
	rows := []string{"Installed: " + name + " " + version, "Path: " + result.Plan.Path}
	if options.use {
		rows = append(rows, "Current: "+version)
	}
	for _, shim := range result.Shims {
		rows = append(rows, "Shim: "+shim.Name+" -> "+shim.Target)
	}
	return writer(cmd).Write(output.Result{
		Title:   titleAction(name) + " Install",
		Command: []string{planCommand},
		Summary: "Runtime installed under ~/.kit/runtimes.",
		Result:  strings.Join(rows, "\n"),
		Hint:    []string{"export PATH=\"$HOME/.kit/shims:$PATH\"", "kit " + name + " current"},
	})
}

func runRuntimeRemove(cmd *cobra.Command, name string, version string) error {
	manager, err := kitruntime.NewManager()
	if err != nil {
		return err
	}
	command := runner.Shell("rm -rf " + runner.Quote(manager.VersionDir(name, version)))
	if opts.dryRun {
		return writeDryRun(cmd, titleAction(name)+" Remove", []runner.Command{command}, []string{"kit " + name + " list"})
	}
	if !opts.yes {
		ok, err := confirmExecution(cmd, titleAction(name)+" Remove", command, "Runtime version directory will be deleted.", false)
		if err != nil {
			return err
		}
		if !ok {
			return canceled(cmd, titleAction(name)+" Remove", command)
		}
	}
	if err := manager.Remove(name, version); err != nil {
		return err
	}
	return writer(cmd).Write(output.Result{
		Title:   titleAction(name) + " Remove",
		Command: []string{command.String()},
		Summary: "Runtime version removed.",
		Result:  name + " " + version,
		Hint:    []string{"kit " + name + " list"},
	})
}

func runRuntimeCache(cmd *cobra.Command, name string, version string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	baseURL := runtimeBaseURL(cfg)
	if baseURL == "" {
		baseURL = "https://kit.example.com/runtime"
	}
	plan := kitruntime.NewDownloadPlan(baseURL, name, version)
	return writer(cmd).Write(output.Result{
		Title:   "Runtime Cache",
		Command: []string{runner.External("curl", "-fsSL", plan.URL).String()},
		Summary: fmt.Sprintf("Runtime: %s\nVersion: %s\nOS: %s\nArch: %s", plan.Name, plan.Version, plan.OS, plan.Arch),
		Result:  plan.URL,
		Hint:    []string{"kit " + name + " install " + version},
	})
}

func newRuntimeServeCommand() *cobra.Command {
	addr := ":8081"
	cacheDir := defaultRuntimeCacheDir()
	command := &cobra.Command{
		Use:   "serve",
		Short: "Serve cached runtime archives",
		RunE: func(cmd *cobra.Command, args []string) error {
			cacheDir = expandPath(cacheDir)
			display := runner.Shell("kit runtime serve --addr " + runner.Quote(addr) + " --cache-dir " + runner.Quote(cacheDir))
			if opts.dryRun {
				return writeDryRun(cmd, "Runtime Serve", []runner.Command{display}, []string{"kit runtime cache node 22.3.0"})
			}
			cmd.Printf("Title\n  Runtime Serve\n\nResult\n  Listening on %s\n  Cache: %s\n\n", addr, cacheDir)
			return http.ListenAndServe(addr, kitruntime.NewCacheServer(cacheDir))
		},
	}
	command.Flags().StringVar(&addr, "addr", addr, "listen address")
	command.Flags().StringVar(&cacheDir, "cache-dir", cacheDir, "runtime cache directory")
	return command
}

func selectAvailableRuntimeVersion(cmd *cobra.Command, name string) (string, error) {
	spec, ok := kitruntime.Specs()[name]
	if !ok {
		return "", fmt.Errorf("unsupported runtime: %s", name)
	}
	choices := make([]builder.Choice, 0, len(spec.SeedVersions))
	for _, version := range spec.SeedVersions {
		choices = append(choices, builder.Choice{Label: version, Value: version})
	}
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	return prompt.Select("설치할 버전을 선택하세요", choices, len(choices)-1)
}

func selectInstalledRuntimeVersion(cmd *cobra.Command, manager kitruntime.Manager, name string) (string, error) {
	versions, err := manager.Installed(name)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no installed %s versions", name)
	}
	choices := make([]builder.Choice, 0, len(versions))
	for _, version := range versions {
		choices = append(choices, builder.Choice{Label: version.Version, Value: version.Version})
	}
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	return prompt.Select("사용할 버전을 선택하세요", choices, len(choices)-1)
}

func runtimeBaseURL(cfg config.Config) string {
	if value := os.Getenv("KIT_RUNTIME_BASE_URL"); value != "" {
		return value
	}
	return cfg.Server.RuntimeBaseURL
}

func runtimeVersionCommand(name string) string {
	switch name {
	case "node":
		return "node -v"
	case "go":
		return "go version"
	case "java":
		return "java -version"
	default:
		return name + " --version"
	}
}

func runtimeInstallCommandString(manager kitruntime.Manager, request kitruntime.InstallRequest) string {
	source := request.Source
	if source == "" {
		source = kitruntime.RuntimeDownloadURL(request.RuntimeBaseURL, request.Runtime, request.Version, request.OS, request.Arch)
	}
	destination := runner.Quote(manager.VersionDir(request.Runtime, request.Version))
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		download := runner.External("curl", kitruntime.CurlDownloadArgs("<temp-file>", source)...).String()
		return download + " && extract '<temp-file>' to " + destination
	}
	return "extract " + runner.Quote(source) + " to " + destination
}

func defaultRuntimeCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".kit-server", "cache", "runtimes")
	}
	return filepath.Join(home, ".kit-server", "cache", "runtimes")
}
