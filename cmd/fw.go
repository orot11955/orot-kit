package cmd

import (
	"context"
	"fmt"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

func registerFirewallCommands(root *cobra.Command) {
	fw := &cobra.Command{
		Use:   "fw",
		Short: "Inspect and change firewall rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirewallStatus(cmd)
		},
	}
	fw.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show firewall status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirewallStatus(cmd)
		},
	})
	fw.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List firewall rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirewallStatus(cmd)
		},
	})
	fw.AddCommand(newFirewallChangeCommand("open"))
	fw.AddCommand(newFirewallChangeCommand("close"))
	root.AddCommand(fw)
}

func runFirewallStatus(cmd *cobra.Command) error {
	command, err := firewallStatusCommand()
	if err != nil {
		return writer(cmd).Write(output.Result{Title: "Firewall Status", Summary: err.Error()})
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Firewall Status", []runner.Command{command}, []string{"kit fw open", "kit fw close"})
	}
	result := runner.Run(context.Background(), command)
	return writeRunnerResults(cmd, "Firewall Status", "Firewall rules and status.", []runner.Result{result}, []string{"kit fw open", "kit fw close"})
}

func newFirewallChangeCommand(action string) *cobra.Command {
	var protocol string
	command := &cobra.Command{
		Use:   action + " <port>",
		Short: action + " a firewall port",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command, err := firewallChangeCommand(action, args[0], protocol)
			if err != nil {
				return writer(cmd).Write(output.Result{Title: "Firewall " + titleAction(action), Summary: err.Error()})
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Firewall "+titleAction(action), []runner.Command{command}, []string{"kit fw status"})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "Firewall "+titleAction(action), command, "방화벽 규칙을 변경합니다.", false)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "Firewall "+titleAction(action), command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Firewall "+titleAction(action), "Firewall rule change requested.", []runner.Result{result}, []string{"kit fw status"})
		},
	}
	command.Flags().StringVar(&protocol, "protocol", "tcp", "tcp or udp")
	return command
}

func firewallStatusCommand() (runner.Command, error) {
	if detect.CommandExists("ufw") {
		return runner.External("ufw", "status", "verbose"), nil
	}
	if detect.CommandExists("firewall-cmd") {
		return runner.External("firewall-cmd", "--list-all"), nil
	}
	if detect.IsDarwin() && detect.CommandExists("pfctl") {
		return runner.External("sudo", "pfctl", "-sr"), nil
	}
	return runner.Command{}, fmt.Errorf("no supported firewall command found")
}

func firewallChangeCommand(action string, port string, protocol string) (runner.Command, error) {
	if protocol == "" {
		protocol = "tcp"
	}
	if protocol != "tcp" && protocol != "udp" {
		return runner.Command{}, fmt.Errorf("protocol must be tcp or udp")
	}
	if detect.CommandExists("ufw") {
		if action == "open" {
			return runner.External("sudo", "ufw", "allow", port+"/"+protocol), nil
		}
		return runner.External("sudo", "ufw", "delete", "allow", port+"/"+protocol), nil
	}
	if detect.CommandExists("firewall-cmd") {
		firewalldAction := "--add-port="
		if action == "close" {
			firewalldAction = "--remove-port="
		}
		return runner.Shell("sudo firewall-cmd " + firewalldAction + runner.Quote(port+"/"+protocol) + " --permanent && sudo firewall-cmd --reload"), nil
	}
	if detect.IsDarwin() {
		return runner.Command{}, fmt.Errorf("macOS pfctl changes are not automated in the current MVP")
	}
	return runner.Command{}, fmt.Errorf("no supported firewall command found")
}
