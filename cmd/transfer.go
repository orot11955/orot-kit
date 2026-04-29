package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/config"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type transferOptions struct {
	server string
	local  string
	remote string
	method string
}

func registerTransferCommands(root *cobra.Command) {
	root.AddCommand(newTransferCommand("send"))
	root.AddCommand(newTransferCommand("receive"))
	root.AddCommand(newTransferCommand("sync"))
}

func newTransferCommand(mode string) *cobra.Command {
	options := transferOptions{method: defaultTransferMethod(mode)}
	command := &cobra.Command{
		Use:   mode,
		Short: mode + " files through SSH",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && options.local == "" && mode != "receive" {
				options.local = args[0]
			}
			if len(args) > 0 && options.remote == "" && mode == "receive" {
				options.remote = args[0]
			}
			if len(args) > 1 && options.server == "" {
				options.server = args[1]
			}
			if len(args) > 2 {
				if mode == "receive" {
					options.local = args[2]
				} else {
					options.remote = args[2]
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			options, err = fillTransferOptions(cmd, mode, options, cfg)
			if err != nil {
				return err
			}
			command, err := buildTransferCommand(mode, options, cfg)
			if err != nil {
				return err
			}
			title := "Transfer " + titleAction(mode)
			if opts.dryRun {
				return writeDryRun(cmd, title, []runner.Command{command}, []string{"kit ssh add", "kit port"})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, title, command, "Files will be transferred over SSH using "+options.method+".", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, title, command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, title, "Transfer command finished.", []runner.Result{result}, []string{"kit ssh add"})
		},
	}
	command.Flags().StringVar(&options.server, "server", "", "configured SSH server name or raw SSH target")
	command.Flags().StringVar(&options.local, "local", "", "local path")
	command.Flags().StringVar(&options.remote, "remote", "", "remote path")
	command.Flags().StringVar(&options.method, "method", options.method, "scp, rsync, or tar")
	return command
}

func fillTransferOptions(cmd *cobra.Command, mode string, options transferOptions, cfg config.Config) (transferOptions, error) {
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	var err error
	if options.local == "" && mode != "send" && mode != "sync" {
		options.local, err = prompt.Ask("로컬 저장 경로", ".")
		if err != nil {
			return options, err
		}
	}
	if options.local == "" {
		options.local, err = prompt.Ask("로컬 파일 또는 디렉토리", ".")
		if err != nil {
			return options, err
		}
	}
	if options.server == "" {
		options.server, err = selectSSHHost(cmd, cfg)
		if err != nil {
			return options, err
		}
	}
	if options.remote == "" {
		label := "대상 원격 경로"
		if mode == "receive" {
			label = "가져올 원격 경로"
		}
		options.remote, err = prompt.Ask(label, "/tmp")
		if err != nil {
			return options, err
		}
	}
	if options.method == "" {
		options.method = defaultTransferMethod(mode)
	}
	if options.method == "tar+ssh" {
		options.method = "tar"
	}
	return options, nil
}

func selectSSHHost(cmd *cobra.Command, cfg config.Config) (string, error) {
	names := config.SSHHostNames(cfg)
	if len(names) == 0 {
		prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
		return prompt.Ask("대상 서버", "")
	}
	choices := make([]builder.Choice, 0, len(names)+1)
	for _, name := range names {
		host := cfg.SSH.Hosts[name]
		label := name
		if host.Host != "" {
			label += " (" + host.User + "@" + host.Host + ":" + strconv.Itoa(host.Port) + ")"
		}
		choices = append(choices, builder.Choice{Label: label, Value: name})
	}
	choices = append(choices, builder.Choice{Label: "직접 입력", Value: "__custom__"})
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	value, err := prompt.Select("대상 서버를 선택하세요", choices, 0)
	if err != nil {
		return "", err
	}
	if value == "__custom__" {
		return prompt.Ask("대상 서버", "")
	}
	return value, nil
}

func buildTransferCommand(mode string, options transferOptions, cfg config.Config) (runner.Command, error) {
	if options.server == "" {
		return runner.Command{}, fmt.Errorf("server is required")
	}
	if options.local == "" {
		return runner.Command{}, fmt.Errorf("local path is required")
	}
	if options.remote == "" {
		return runner.Command{}, fmt.Errorf("remote path is required")
	}
	host, hasHost := cfg.SSH.Hosts[options.server]
	target := sshTarget(options.server, host, hasHost)
	method := options.method
	if method == "" {
		method = defaultTransferMethod(mode)
	}
	switch method {
	case "scp":
		return scpTransferCommand(mode, options, target, host, hasHost), nil
	case "rsync":
		return rsyncTransferCommand(mode, options, target, host, hasHost), nil
	case "tar":
		return tarTransferCommand(mode, options, target, host, hasHost), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported transfer method: %s", options.method)
	}
}

func scpTransferCommand(mode string, options transferOptions, target string, host config.SSHHost, hasHost bool) runner.Command {
	args := []string{}
	if hasHost {
		if host.Port != 0 {
			args = append(args, "-P", strconv.Itoa(host.Port))
		}
		if host.IdentityFile != "" {
			args = append(args, "-i", expandPath(host.IdentityFile))
		}
	}
	args = append(args, "-r")
	if mode == "receive" {
		args = append(args, target+":"+options.remote, options.local)
	} else {
		args = append(args, options.local, target+":"+options.remote)
	}
	return runner.External("scp", args...)
}

func rsyncTransferCommand(mode string, options transferOptions, target string, host config.SSHHost, hasHost bool) runner.Command {
	args := []string{"-avz"}
	if hasHost {
		ssh := sshShellCommand(host)
		if ssh != "ssh" {
			args = append(args, "-e", ssh)
		}
	}
	if mode == "receive" {
		args = append(args, target+":"+options.remote, options.local)
	} else {
		args = append(args, options.local, target+":"+options.remote)
	}
	return runner.External("rsync", args...)
}

func tarTransferCommand(mode string, options transferOptions, target string, host config.SSHHost, hasHost bool) runner.Command {
	ssh := sshShellCommand(host)
	if !hasHost {
		ssh = "ssh"
	}
	if mode == "receive" {
		remoteDir := filepath.Dir(options.remote)
		remoteBase := filepath.Base(options.remote)
		return runner.Shell("mkdir -p " + runner.Quote(options.local) + " && " + ssh + " " + runner.Quote(target) + " " + runner.Quote("tar -czf - -C "+remoteDir+" "+remoteBase) + " | tar -xzf - -C " + runner.Quote(options.local))
	}
	localDir := filepath.Dir(options.local)
	localBase := filepath.Base(options.local)
	return runner.Shell("tar -czf - -C " + runner.Quote(localDir) + " " + runner.Quote(localBase) + " | " + ssh + " " + runner.Quote(target) + " " + runner.Quote("mkdir -p "+options.remote+" && tar -xzf - -C "+options.remote))
}

func sshTarget(server string, host config.SSHHost, hasHost bool) string {
	if !hasHost || host.Host == "" {
		return server
	}
	if host.User == "" {
		return host.Host
	}
	return host.User + "@" + host.Host
}

func sshShellCommand(host config.SSHHost) string {
	parts := []string{"ssh"}
	if host.Port != 0 {
		parts = append(parts, "-p", strconv.Itoa(host.Port))
	}
	if host.IdentityFile != "" {
		parts = append(parts, "-i", expandPath(host.IdentityFile))
	}
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, runner.Quote(part))
	}
	return strings.Join(quoted, " ")
}

func defaultTransferMethod(mode string) string {
	if mode == "send" {
		return "rsync"
	}
	if mode == "receive" {
		return "rsync"
	}
	return "rsync"
}

func expandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
