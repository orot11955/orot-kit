package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/config"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type sshAddOptions struct {
	name        string
	host        string
	user        string
	port        int
	identity    string
	generateKey bool
	copyKey     bool
}

func registerSSHCommands(root *cobra.Command) {
	ssh := &cobra.Command{
		Use:   "ssh",
		Short: "Manage SSH hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			names := config.SSHHostNames(cfg)
			if len(names) == 0 {
				return writer(cmd).Write(output.Result{
					Title:   "SSH",
					Summary: "No SSH hosts configured.",
					Hint:    []string{"kit ssh add"},
				})
			}
			rows := make([]string, 0, len(names))
			for _, name := range names {
				host := cfg.SSH.Hosts[name]
				rows = append(rows, fmt.Sprintf("%s  %s@%s:%d  %s", name, host.User, host.Host, host.Port, host.IdentityFile))
			}
			return writer(cmd).Write(output.Result{
				Title:  "SSH Hosts",
				Result: strings.Join(rows, "\n"),
				Hint:   []string{"kit send", "kit receive"},
			})
		},
	}
	ssh.AddCommand(newSSHAddCommand())
	ssh.AddCommand(newSSHKeygenCommand())
	ssh.AddCommand(newSSHCopyCommand())
	root.AddCommand(ssh)
}

func newSSHAddCommand() *cobra.Command {
	options := sshAddOptions{port: 22, identity: "~/.ssh/id_ed25519"}
	command := &cobra.Command{
		Use:   "add [name]",
		Short: "Add an SSH host for reuse by transfer commands",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				options.name = args[0]
			}
			var err error
			options, err = fillSSHAddOptions(cmd, options)
			if err != nil {
				return err
			}
			host := config.SSHHost{
				Host:         options.host,
				User:         options.user,
				Port:         options.port,
				IdentityFile: options.identity,
			}
			commands := sshAddCommands(options)
			if opts.dryRun {
				preview := append([]runner.Command{runner.Shell("write " + runner.Quote(config.DefaultPath()))}, commands...)
				return writeDryRun(cmd, "SSH Add", preview, []string{"kit ssh", "kit send"})
			}
			if !opts.yes {
				preview := runner.Shell("write " + runner.Quote(config.DefaultPath()))
				ok, err := confirmExecution(cmd, "SSH Add", preview, "Host will be stored for reuse by kit send/receive/sync.", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "SSH Add", preview)
				}
			}
			if err := config.UpsertSSHHost(options.name, host); err != nil {
				return err
			}
			results := make([]runner.Result, 0, len(commands))
			for _, command := range commands {
				results = append(results, runner.Run(context.Background(), command))
			}
			if len(results) == 0 {
				return writer(cmd).Write(output.Result{
					Title:   "SSH Add",
					Command: []string{"write " + config.DefaultPath()},
					Summary: "SSH host saved.",
					Result:  fmt.Sprintf("%s -> %s@%s:%d", options.name, options.user, options.host, options.port),
					Hint:    []string{"kit ssh", "kit send"},
				})
			}
			return writeRunnerResults(cmd, "SSH Add", "SSH host saved. Optional SSH commands finished.", results, []string{"kit ssh", "kit send"})
		},
	}
	command.Flags().StringVar(&options.name, "name", "", "server alias")
	command.Flags().StringVar(&options.host, "host", "", "SSH host or IP")
	command.Flags().StringVar(&options.user, "user", "", "SSH username")
	command.Flags().IntVar(&options.port, "port", 22, "SSH port")
	command.Flags().StringVarP(&options.identity, "identity", "i", "~/.ssh/id_ed25519", "identity file")
	command.Flags().BoolVar(&options.generateKey, "generate-key", false, "generate identity file if missing")
	command.Flags().BoolVar(&options.copyKey, "copy-key", false, "copy public key to the server")
	return command
}

func fillSSHAddOptions(cmd *cobra.Command, options sshAddOptions) (sshAddOptions, error) {
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	var err error
	prompted := false
	if options.name == "" {
		options.name, err = prompt.Ask("서버 이름", "")
		if err != nil {
			return options, err
		}
		prompted = true
	}
	if options.host == "" {
		options.host, err = prompt.Ask("Host", "")
		if err != nil {
			return options, err
		}
		prompted = true
	}
	if options.user == "" {
		options.user, err = prompt.Ask("User", os.Getenv("USER"))
		if err != nil {
			return options, err
		}
		prompted = true
	}
	if prompted && !cmd.Flags().Changed("port") {
		port, err := prompt.Ask("Port", "22")
		if err != nil {
			return options, err
		}
		options.port, err = strconv.Atoi(port)
		if err != nil {
			return options, err
		}
		prompted = true
	}
	if prompted && !cmd.Flags().Changed("identity") {
		options.identity, err = prompt.Ask("Identity file", "~/.ssh/id_ed25519")
		if err != nil {
			return options, err
		}
		prompted = true
	} else if options.identity == "" {
		options.identity = "~/.ssh/id_ed25519"
	}
	if prompted && !cmd.Flags().Changed("generate-key") {
		options.generateKey, err = prompt.Confirm("SSH key를 생성할까요?", false)
		if err != nil {
			return options, err
		}
	}
	if prompted && !cmd.Flags().Changed("copy-key") {
		options.copyKey, err = prompt.Confirm("서버에 public key를 복사할까요?", false)
		if err != nil {
			return options, err
		}
	}
	if options.name == "" || options.host == "" || options.user == "" {
		return options, fmt.Errorf("name, host, and user are required")
	}
	return options, nil
}

func sshAddCommands(options sshAddOptions) []runner.Command {
	commands := []runner.Command{}
	identity := expandPath(options.identity)
	if options.generateKey {
		if _, err := os.Stat(identity); os.IsNotExist(err) {
			commands = append(commands, runner.External("ssh-keygen", "-t", "ed25519", "-C", "kit-"+options.name, "-f", identity, "-N", ""))
		}
	}
	if options.copyKey {
		args := []string{}
		if options.port != 0 {
			args = append(args, "-p", strconv.Itoa(options.port))
		}
		if options.identity != "" {
			args = append(args, "-i", identity+".pub")
		}
		args = append(args, options.user+"@"+options.host)
		commands = append(commands, runner.External("ssh-copy-id", args...))
	}
	return commands
}

func newSSHKeygenCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "keygen [path]",
		Short: "Generate an ed25519 SSH key",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "~/.ssh/id_ed25519"
			if len(args) > 0 {
				path = args[0]
			}
			command := runner.External("ssh-keygen", "-t", "ed25519", "-f", expandPath(path), "-N", "")
			if opts.dryRun {
				return writeDryRun(cmd, "SSH Keygen", []runner.Command{command}, []string{"kit ssh add"})
			}
			if _, err := os.Stat(expandPath(path)); err == nil {
				return writer(cmd).Write(output.Result{
					Title:   "SSH Keygen",
					Command: []string{command.String()},
					Summary: "Key already exists. Refusing to overwrite.",
				})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "SSH Keygen", command, "A new SSH private key will be created.", false)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "SSH Keygen", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "SSH Keygen", "SSH key generation finished.", []runner.Result{result}, []string{"kit ssh add"})
		},
	}
}

func newSSHCopyCommand() *cobra.Command {
	var port int
	var identity string
	command := &cobra.Command{
		Use:   "copy <user@host>",
		Short: "Copy a public key to an SSH host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			copyArgs := []string{}
			if port != 0 {
				copyArgs = append(copyArgs, "-p", strconv.Itoa(port))
			}
			if identity != "" {
				copyArgs = append(copyArgs, "-i", expandPath(identity)+".pub")
			}
			copyArgs = append(copyArgs, args[0])
			command := runner.External("ssh-copy-id", copyArgs...)
			if opts.dryRun {
				return writeDryRun(cmd, "SSH Copy", []runner.Command{command}, []string{"kit ssh add"})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "SSH Copy", command, "Public key will be installed on the remote server.", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "SSH Copy", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "SSH Copy", "SSH public key copy finished.", []runner.Result{result}, []string{"kit ssh add"})
		},
	}
	command.Flags().IntVar(&port, "port", 22, "SSH port")
	command.Flags().StringVarP(&identity, "identity", "i", "~/.ssh/id_ed25519", "identity file")
	return command
}
