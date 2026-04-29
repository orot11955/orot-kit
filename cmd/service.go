package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/config"
	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type serviceOptions struct {
	tail int
}

func registerServiceCommands(root *cobra.Command) {
	options := &serviceOptions{tail: 100}
	service := &cobra.Command{
		Use:   "service [service] [action]",
		Short: "Control services",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runService(cmd, args, options)
		},
	}
	service.PersistentFlags().IntVar(&options.tail, "tail", 100, "log line count")
	service.AddCommand(newServiceActionCommand("list", "List running services", options))
	service.AddCommand(newServiceActionCommand("status", "Show service status", options))
	service.AddCommand(newServiceActionCommand("up", "Start service", options))
	service.AddCommand(newServiceActionCommand("down", "Stop service", options))
	service.AddCommand(newServiceActionCommand("restart", "Restart service", options))
	service.AddCommand(newServiceActionCommand("logs", "Show service logs", options))
	service.AddCommand(newServiceAddCommand())
	root.AddCommand(service)
	registerServiceAliases(root, options)
}

func newServiceAddCommand() *cobra.Command {
	var alias string
	var serviceType string
	var name string
	var path string
	command := &cobra.Command{
		Use:   "add [alias]",
		Short: "Add a reusable service alias",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				alias = args[0]
			}
			prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
			var err error
			if alias == "" {
				alias, err = prompt.Ask("서비스 alias", "")
				if err != nil {
					return err
				}
			}
			if serviceType == "" {
				serviceType, err = prompt.Select("서비스 타입", []builder.Choice{
					{Label: "systemctl", Value: "systemctl"},
					{Label: "docker-compose", Value: "docker-compose"},
					{Label: "docker", Value: "docker"},
					{Label: "brew", Value: "brew"},
				}, 0)
				if err != nil {
					return err
				}
			}
			if name == "" {
				name, err = prompt.Ask("실제 서비스 이름", alias)
				if err != nil {
					return err
				}
			}
			if path == "" && (serviceType == "docker-compose" || serviceType == "compose") {
				path, err = prompt.Ask("Compose 프로젝트 경로", ".")
				if err != nil {
					return err
				}
			}
			if alias == "" || name == "" || serviceType == "" {
				return fmt.Errorf("alias, type, and name are required")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.Services == nil {
				cfg.Services = map[string]config.Service{}
			}
			cfg.Services[alias] = config.Service{Type: serviceType, Name: name, Path: path}
			command := runner.Shell("write " + runner.Quote(config.DefaultPath()))
			if opts.dryRun {
				return writeDryRun(cmd, "Service Add", []runner.Command{command}, []string{"kit " + alias + " status", "kit service list"})
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			return writer(cmd).Write(output.Result{
				Title:   "Service Add",
				Command: []string{command.String()},
				Summary: "Service alias saved.",
				Result:  fmt.Sprintf("%s -> %s %s", alias, serviceType, name),
				Hint:    []string{"kit " + alias + " status", "kit service list"},
			})
		},
	}
	command.Flags().StringVar(&alias, "alias", "", "service alias")
	command.Flags().StringVar(&serviceType, "type", "", "systemctl, docker-compose, docker, or brew")
	command.Flags().StringVar(&name, "name", "", "actual service name")
	command.Flags().StringVar(&path, "path", "", "service project path")
	return command
}

func newServiceActionCommand(action string, short string, options *serviceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   action + " [service]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if action == "list" {
				return runService(cmd, []string{"list"}, options)
			}
			if len(args) == 0 {
				return fmt.Errorf("service name is required for %s", action)
			}
			return runService(cmd, []string{args[0], action}, options)
		},
	}
}

func runService(cmd *cobra.Command, args []string, options *serviceOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	action, name := parseServiceArgs(args)
	commands, summary, err := serviceCommands(action, name, *options, cfg)
	if err != nil {
		return writer(cmd).Write(output.Result{
			Title:   "Service " + titleAction(action),
			Summary: err.Error(),
			Hint:    []string{"kit service list", "kit docker ps"},
		})
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Service "+titleAction(action), commands, []string{"kit service list", "kit docker ps"})
	}
	results := runner.RunMany(context.Background(), commands)
	return writeRunnerResults(cmd, "Service "+titleAction(action), summary, results, []string{"kit service list", "kit docker ps"})
}

func parseServiceArgs(args []string) (string, string) {
	action := "list"
	name := ""
	if len(args) == 1 {
		if isServiceAction(args[0]) {
			action = normalizeServiceAction(args[0])
		} else {
			name = args[0]
			action = "status"
		}
	}
	if len(args) >= 2 {
		if isServiceAction(args[0]) {
			action = normalizeServiceAction(args[0])
			name = args[1]
		} else {
			name = args[0]
			action = normalizeServiceAction(args[1])
		}
	}
	return action, name
}

func isServiceAction(value string) bool {
	switch value {
	case "list", "status", "up", "start", "down", "stop", "restart", "logs":
		return true
	default:
		return false
	}
}

func normalizeServiceAction(value string) string {
	switch value {
	case "start":
		return "up"
	case "stop":
		return "down"
	default:
		return value
	}
}

func serviceCommands(action string, alias string, options serviceOptions, cfg config.Config) ([]runner.Command, string, error) {
	if action == "list" {
		command, err := serviceListCommand()
		if err != nil {
			return nil, "", err
		}
		summary := "Detected service manager list."
		if configured := configuredServicesSummary(cfg); configured != "" {
			summary += "\nConfigured aliases:\n" + configured
		}
		return []runner.Command{command}, summary, nil
	}
	if alias == "" {
		return nil, "", fmt.Errorf("service name is required for %s", action)
	}
	service, configured := cfg.Services[alias]
	if !configured {
		service = config.Service{Name: alias, Type: "auto"}
	}
	if service.Name == "" {
		service.Name = alias
	}
	command, err := serviceCommandForTarget(action, service, options)
	if err != nil {
		return nil, "", err
	}
	summary := fmt.Sprintf("Service: %s\nType: %s\nTarget: %s", alias, resolvedServiceType(service), service.Name)
	if service.Path != "" {
		summary += "\nPath: " + service.Path
	}
	return []runner.Command{command}, summary, nil
}

func serviceListCommand() (runner.Command, error) {
	if detect.CommandExists("systemctl") {
		return runner.External("systemctl", "list-units", "--type=service", "--state=running"), nil
	}
	if detect.IsDarwin() && detect.CommandExists("brew") {
		return runner.External("brew", "services", "list"), nil
	}
	if detect.CommandExists("docker") {
		return runner.External("docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}"), nil
	}
	return runner.Command{}, fmt.Errorf("no supported service manager found")
}

func serviceCommandForTarget(action string, service config.Service, options serviceOptions) (runner.Command, error) {
	serviceType := resolvedServiceType(service)
	switch serviceType {
	case "systemctl", "systemd":
		return systemctlServiceCommand(action, service.Name, options)
	case "brew", "homebrew":
		return brewServiceCommand(action, service.Name, options)
	case "docker-compose", "compose":
		return composeServiceCommand(action, service, options)
	case "docker":
		return dockerServiceCommand(action, service.Name, options)
	default:
		return runner.Command{}, fmt.Errorf("unsupported service type: %s", serviceType)
	}
}

func resolvedServiceType(service config.Service) string {
	if service.Type != "" && service.Type != "auto" {
		return service.Type
	}
	if service.Path != "" {
		return "docker-compose"
	}
	if detect.CommandExists("systemctl") {
		return "systemctl"
	}
	if detect.IsDarwin() && detect.CommandExists("brew") {
		return "brew"
	}
	if detect.CommandExists("docker") {
		return "docker"
	}
	return "auto"
}

func systemctlServiceCommand(action string, name string, options serviceOptions) (runner.Command, error) {
	if name == "" {
		return runner.Command{}, fmt.Errorf("service name is required for %s", action)
	}
	switch action {
	case "status":
		return runner.External("systemctl", "status", name), nil
	case "up":
		return runner.External("systemctl", "start", name), nil
	case "down":
		return runner.External("systemctl", "stop", name), nil
	case "restart":
		return runner.External("systemctl", "restart", name), nil
	case "logs":
		return runner.External("journalctl", "-u", name, "-n", fmt.Sprint(options.tail), "--no-pager"), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported systemctl action: %s", action)
	}
}

func brewServiceCommand(action string, name string, options serviceOptions) (runner.Command, error) {
	if name == "" {
		return runner.Command{}, fmt.Errorf("service name is required for %s", action)
	}
	switch action {
	case "status":
		return runner.External("brew", "services", "info", name), nil
	case "up":
		return runner.External("brew", "services", "start", name), nil
	case "down":
		return runner.External("brew", "services", "stop", name), nil
	case "restart":
		return runner.External("brew", "services", "restart", name), nil
	case "logs":
		return runner.External("brew", "services", "info", name), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported brew service action: %s", action)
	}
}

func composeServiceCommand(action string, service config.Service, options serviceOptions) (runner.Command, error) {
	compose := dockerComposeBaseArgs(dockerOptions{projectDir: service.Path})
	name := service.Name
	switch action {
	case "status":
		args := append(compose, "ps")
		if name != "" {
			args = append(args, name)
		}
		return dockerComposeRunner(args), nil
	case "up":
		args := append(compose, "up", "-d")
		if name != "" {
			args = append(args, name)
		}
		return dockerComposeRunner(args), nil
	case "down":
		if name != "" {
			return dockerComposeRunner(append(compose, "stop", name)), nil
		}
		return dockerComposeRunner(append(compose, "down")), nil
	case "restart":
		args := append(compose, "restart")
		if name != "" {
			args = append(args, name)
		}
		return dockerComposeRunner(args), nil
	case "logs":
		args := append(compose, "logs", "--tail", fmt.Sprint(options.tail))
		if name != "" {
			args = append(args, name)
		}
		return dockerComposeRunner(args), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported docker compose service action: %s", action)
	}
}

func dockerServiceCommand(action string, name string, options serviceOptions) (runner.Command, error) {
	if name == "" {
		return runner.Command{}, fmt.Errorf("container name is required for %s", action)
	}
	switch action {
	case "status":
		return runner.External("docker", "ps", "--filter", "name="+name), nil
	case "up":
		return runner.External("docker", "start", name), nil
	case "down":
		return runner.External("docker", "stop", name), nil
	case "restart":
		return runner.External("docker", "restart", name), nil
	case "logs":
		return runner.External("docker", "logs", "--tail", fmt.Sprint(options.tail), name), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported docker service action: %s", action)
	}
}

func configuredServicesSummary(cfg config.Config) string {
	names := config.ServiceNames(cfg)
	if len(names) == 0 {
		return ""
	}
	rows := make([]string, 0, len(names))
	for _, name := range names {
		service := cfg.Services[name]
		target := service.Name
		if target == "" {
			target = name
		}
		parts := []string{name, service.Type, target}
		if service.Path != "" {
			parts = append(parts, filepath.Clean(expandPath(service.Path)))
		}
		rows = append(rows, strings.Join(parts, "  "))
	}
	return strings.Join(rows, "\n")
}

func registerServiceAliases(root *cobra.Command, options *serviceOptions) {
	cfg, _ := config.Load()
	aliases := map[string]config.Service{}
	for _, name := range commonServiceAliases() {
		aliases[name] = config.Service{Name: name, Type: "auto"}
	}
	for name, service := range cfg.Services {
		aliases[name] = service
	}
	for alias, service := range aliases {
		if rootCommandExists(root, alias) {
			continue
		}
		root.AddCommand(newServiceAliasCommand(alias, service, options))
	}
}

func newServiceAliasCommand(alias string, service config.Service, options *serviceOptions) *cobra.Command {
	command := &cobra.Command{
		Use:    alias + " [status|up|down|restart|logs]",
		Short:  "Service alias for " + alias,
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "status"
			if len(args) > 0 {
				action = normalizeServiceAction(args[0])
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if configured, ok := cfg.Services[alias]; ok {
				service = configured
			}
			if service.Name == "" {
				service.Name = alias
			}
			command, err := serviceCommandForTarget(action, service, *options)
			if err != nil {
				return err
			}
			title := "Service " + titleAction(action)
			if opts.dryRun {
				return writeDryRun(cmd, title, []runner.Command{command}, []string{"kit service " + alias + " " + action})
			}
			result := runner.Run(context.Background(), command)
			summary := fmt.Sprintf("Alias: %s\nType: %s\nTarget: %s", alias, resolvedServiceType(service), service.Name)
			return writeRunnerResults(cmd, title, summary, []runner.Result{result}, []string{"kit service " + alias + " " + action})
		},
	}
	command.Flags().IntVar(&options.tail, "tail", 100, "log line count")
	return command
}

func rootCommandExists(root *cobra.Command, name string) bool {
	if reservedRootCommand(name) {
		return true
	}
	for _, command := range root.Commands() {
		if command.Name() == name {
			return true
		}
	}
	return false
}

func reservedRootCommand(name string) bool {
	switch name {
	case "docker", "ssh", "send", "receive", "sync", "service", "git", "runtime", "node", "go", "python", "java", "secret", "fw", "install-server", "uninstall", "diff":
		return true
	default:
		return false
	}
}

func commonServiceAliases() []string {
	return []string{"nginx", "apache", "httpd", "mysql", "mariadb", "postgres", "postgresql", "redis", "sshd", "cron"}
}

func titleAction(action string) string {
	if action == "" {
		return ""
	}
	return strings.ToUpper(action[:1]) + action[1:]
}
