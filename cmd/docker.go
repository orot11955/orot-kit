package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type dockerOptions struct {
	composeFile string
	projectDir  string
	tail        int
	follow      bool
	all         bool
	target      string
}

func registerDockerCommands(root *cobra.Command) {
	options := &dockerOptions{tail: 100}
	docker := &cobra.Command{
		Use:   "docker",
		Short: "Common Docker and Docker Compose tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDockerPS(cmd, options)
		},
	}
	docker.PersistentFlags().StringVarP(&options.composeFile, "file", "f", "", "compose file")
	docker.PersistentFlags().StringVar(&options.projectDir, "project-directory", "", "compose project directory")
	docker.AddCommand(newDockerPSCommand(options))
	docker.AddCommand(newDockerUpCommand(options))
	docker.AddCommand(newDockerDownCommand(options))
	docker.AddCommand(newDockerRestartCommand(options))
	docker.AddCommand(newDockerLogsCommand(options))
	docker.AddCommand(newDockerCleanCommand(options))
	root.AddCommand(docker)
}

func newDockerPSCommand(options *dockerOptions) *cobra.Command {
	var compose bool
	command := &cobra.Command{
		Use:     "ps",
		Aliases: []string{"status"},
		Short:   "List Docker containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if compose {
				command := dockerComposeRunner(append(dockerComposeBaseArgs(*options), "ps"))
				return runDockerCommand(cmd, "Docker PS", command, "Docker Compose services.", false)
			}
			return runDockerPS(cmd, options)
		},
	}
	command.Flags().BoolVar(&compose, "compose", false, "show docker compose services")
	command.Flags().BoolVarP(&options.all, "all", "a", false, "show all containers")
	return command
}

func runDockerPS(cmd *cobra.Command, options *dockerOptions) error {
	if !dockerAvailable() {
		return writer(cmd).Write(output.Result{Title: "Docker PS", Summary: "docker command not found."})
	}
	args := []string{"ps"}
	if options.all {
		args = append(args, "-a")
	}
	command := runner.External("docker", args...)
	return runDockerCommand(cmd, "Docker PS", command, "Docker containers.", false)
}

func newDockerUpCommand(options *dockerOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "up [service]",
		Short: "Start Docker Compose services",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeArgs := append(dockerComposeBaseArgs(*options), "up", "-d")
			if len(args) > 0 {
				composeArgs = append(composeArgs, args[0])
			}
			command := dockerComposeRunner(composeArgs)
			return runDockerCommand(cmd, "Docker Up", command, "Docker Compose services will be started in detached mode.", false)
		},
	}
}

func newDockerDownCommand(options *dockerOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop and remove Docker Compose services",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			command := dockerComposeRunner(append(dockerComposeBaseArgs(*options), "down"))
			return runDockerCommand(cmd, "Docker Down", command, "Docker Compose services will be stopped and removed.", false)
		},
	}
}

func newDockerRestartCommand(options *dockerOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "restart [service]",
		Short: "Restart Docker Compose services",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeArgs := append(dockerComposeBaseArgs(*options), "restart")
			if len(args) > 0 {
				composeArgs = append(composeArgs, args[0])
			}
			command := dockerComposeRunner(composeArgs)
			return runDockerCommand(cmd, "Docker Restart", command, "Docker Compose services will be restarted.", false)
		},
	}
}

func newDockerLogsCommand(options *dockerOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "logs [service]",
		Short: "Show Docker Compose logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeArgs := append(dockerComposeBaseArgs(*options), "logs", "--tail", fmt.Sprint(options.tail))
			if options.follow {
				composeArgs = append(composeArgs, "-f")
			}
			if len(args) > 0 {
				composeArgs = append(composeArgs, args[0])
			}
			command := dockerComposeRunner(composeArgs)
			return runDockerCommand(cmd, "Docker Logs", command, "Docker Compose logs.", false)
		},
	}
	command.Flags().IntVar(&options.tail, "tail", 100, "log line count")
	command.Flags().BoolVar(&options.follow, "follow", false, "follow logs")
	return command
}

func newDockerCleanCommand(options *dockerOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "clean",
		Short: "Prune unused Docker resources with confirmation",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := options.target
			if target == "" {
				if opts.dryRun {
					target = "all"
				} else {
					selected, err := selectDockerCleanTarget(cmd)
					if err != nil {
						return err
					}
					target = selected
				}
			}
			command, summary, err := dockerCleanCommand(target)
			if err != nil {
				return err
			}
			return runDockerCommand(cmd, "Docker Clean", command, summary, true)
		},
	}
	command.Flags().StringVar(&options.target, "target", "", "containers, images, volumes, build-cache, or all")
	return command
}

func runDockerCommand(cmd *cobra.Command, title string, command runner.Command, summary string, danger bool) error {
	if !dockerAvailable() && command.Name != "docker-compose" {
		return writer(cmd).Write(output.Result{Title: title, Summary: "docker command not found."})
	}
	if opts.dryRun {
		return writeDryRun(cmd, title, []runner.Command{command}, []string{"kit docker ps", "kit docker logs"})
	}
	if danger && !opts.yes {
		ok, err := confirmExecution(cmd, title, command, summary, false)
		if err != nil {
			return err
		}
		if !ok {
			return canceled(cmd, title, command)
		}
	}
	result := runner.Run(context.Background(), command)
	return writeRunnerResults(cmd, title, summary, []runner.Result{result}, []string{"kit docker ps", "kit docker logs"})
}

func selectDockerCleanTarget(cmd *cobra.Command) (string, error) {
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	return prompt.Select("정리할 대상을 선택하세요", []builder.Choice{
		{Label: "stopped containers", Value: "containers"},
		{Label: "unused images", Value: "images"},
		{Label: "unused volumes", Value: "volumes"},
		{Label: "build cache", Value: "build-cache"},
		{Label: "all unused resources", Value: "all"},
	}, 4)
}

func dockerCleanCommand(target string) (runner.Command, string, error) {
	switch target {
	case "containers", "container":
		return runner.External("docker", "container", "prune"), "Stopped containers will be removed.", nil
	case "images", "image":
		return runner.External("docker", "image", "prune"), "Dangling images will be removed.", nil
	case "volumes", "volume":
		return runner.External("docker", "volume", "prune"), "Unused volumes will be removed.", nil
	case "build-cache", "build":
		return runner.External("docker", "builder", "prune"), "Docker build cache will be removed.", nil
	case "all", "system":
		return runner.External("docker", "system", "prune"), "Unused Docker resources will be removed.", nil
	default:
		return runner.Command{}, "", fmt.Errorf("unsupported docker clean target: %s", target)
	}
}

func dockerComposeBaseArgs(options dockerOptions) []string {
	args := []string{"compose"}
	if options.projectDir != "" {
		args = append(args, "--project-directory", expandPath(options.projectDir))
	}
	if options.composeFile != "" {
		args = append(args, "-f", expandPath(options.composeFile))
	}
	return args
}

func dockerComposeRunner(args []string) runner.Command {
	if detect.CommandExists("docker") {
		return runner.External("docker", args...)
	}
	if detect.CommandExists("docker-compose") {
		legacyArgs := args
		if len(legacyArgs) > 0 && legacyArgs[0] == "compose" {
			legacyArgs = legacyArgs[1:]
		}
		return runner.External("docker-compose", legacyArgs...)
	}
	return runner.External("docker", args...)
}

func dockerAvailable() bool {
	return detect.CommandExists("docker") || detect.CommandExists("docker-compose")
}
