package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	urlpath "path"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	kitruntime "github.com/orot-dev/orot-kit/internal/runtime"
	"github.com/spf13/cobra"
)

func registerNetworkCommands(root *cobra.Command) {
	network := newNetworkCommand()
	addNetworkSubcommands(network)
	root.AddCommand(network)
	root.AddCommand(hiddenCommand(newIPCommand()))
	root.AddCommand(hiddenCommand(newPingCommand()))
	root.AddCommand(hiddenCommand(newDigCommand()))
	root.AddCommand(hiddenCommand(newCurlCommand()))
	root.AddCommand(hiddenCommand(newDownloadCommand()))
	root.AddCommand(hiddenCommand(newPortCommand()))
	root.AddCommand(hiddenCommand(newTCPDumpCommand()))
}

func addNetworkSubcommands(command *cobra.Command) {
	command.AddCommand(newIPCommand())
	command.AddCommand(newPingCommand())
	command.AddCommand(newDigCommand())
	command.AddCommand(newCurlCommand())
	command.AddCommand(newDownloadCommand())
	command.AddCommand(newPortCommand())
	command.AddCommand(newTCPDumpCommand())
}

func newNetworkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "network",
		Short: "Show network summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			commands := networkCommands()
			if opts.dryRun {
				return writeDryRun(cmd, "Network Summary", commands, []string{"kit network ip", "kit network port", "kit network ping <host>"})
			}
			results := runner.RunMany(context.Background(), commands)
			system := detect.System()
			summary := "Hostname: " + system.Hostname + "\nOS: " + system.OS
			if primary := detect.PrimaryIP(); primary != "" {
				summary += "\nPrimary IP: " + primary
			}
			if interfaces, err := detect.Interfaces(); err == nil {
				summary += fmt.Sprintf("\nInterfaces: %d detected", len(interfaces))
			}
			return writeRunnerResults(cmd, "Network Summary", summary, results, []string{"kit network ip", "kit network port", "kit network ping <host>"})
		},
	}
}

func networkCommands() []runner.Command {
	commands := []runner.Command{runner.External("hostname")}
	if detect.CommandExists("ip") {
		commands = append(commands, runner.External("ip", "addr"), runner.External("ip", "route"))
	} else if detect.CommandExists("ifconfig") {
		commands = append(commands, runner.External("ifconfig"))
	}
	if detect.CommandExists("resolvectl") {
		commands = append(commands, runner.External("resolvectl", "dns"))
	} else if detect.IsLinux() {
		commands = append(commands, runner.External("cat", "/etc/resolv.conf"))
	} else if detect.IsDarwin() && detect.CommandExists("scutil") {
		commands = append(commands, runner.External("scutil", "--dns"))
	}
	return commands
}

func newIPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "Show network interfaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			commandName := detect.FirstCommand("ip", "ifconfig")
			if commandName == "" {
				return writer(cmd).Write(output.Result{Title: "IP", Summary: "No ip or ifconfig command found."})
			}
			command := runner.External(commandName, "addr")
			if commandName == "ifconfig" {
				command = runner.External("ifconfig")
			}
			if opts.dryRun {
				return writeDryRun(cmd, "IP", []runner.Command{command}, []string{"kit network"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "IP", "Network interface details.", []runner.Result{result}, []string{"kit network"})
		},
	}
}

func newPingCommand() *cobra.Command {
	var count int
	command := &cobra.Command{
		Use:   "ping <host>",
		Short: "Ping a host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			countFlag := "-c"
			if detect.IsDarwin() {
				countFlag = "-c"
			}
			command := runner.External("ping", countFlag, strconv.Itoa(count), args[0])
			if opts.dryRun {
				return writeDryRun(cmd, "Ping", []runner.Command{command}, []string{"kit network", "kit network dig <name>"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Ping", "Connectivity probe.", []runner.Result{result}, []string{"kit network", "kit network dig <name>"})
		},
	}
	command.Flags().IntVar(&count, "count", 4, "packet count")
	return command
}

func newDigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dig <name>",
		Short: "Resolve DNS name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.External("dig", args[0])
			if !detect.CommandExists("dig") {
				command = runner.External("nslookup", args[0])
			}
			if opts.dryRun {
				return writeDryRun(cmd, "DNS Lookup", []runner.Command{command}, []string{"kit network", "kit network ping <host>"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "DNS Lookup", "DNS resolution result.", []runner.Result{result}, []string{"kit network", "kit network ping <host>"})
		},
	}
}

func newCurlCommand() *cobra.Command {
	var method string
	command := &cobra.Command{
		Use:   "curl <url>",
		Short: "Run a focused curl request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.External("curl", "-i", "-L", "--max-time", "10", "-X", method, args[0])
			if opts.dryRun {
				return writeDryRun(cmd, "Curl", []runner.Command{command}, []string{"kit network"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Curl", "HTTP response.", []runner.Result{result}, []string{"kit network"})
		},
	}
	command.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	return command
}

func newDownloadCommand() *cobra.Command {
	var outputPath string
	var sha256 string
	var retry int
	var timeout int
	var executable bool
	command := &cobra.Command{
		Use:   "download <url> [output]",
		Short: "Download a file with curl",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if len(args) > 1 && outputPath == "" {
				outputPath = args[1]
			}
			if outputPath == "" {
				outputPath = defaultDownloadOutput(url)
			}
			downloadCommand := curlDownloadCommand(url, outputPath, retry, timeout)
			commands := []runner.Command{downloadCommand}
			if executable {
				commands = append(commands, runner.External("chmod", "+x", outputPath))
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Download", commands, []string{"kit network download <url> --sha256 <sum>", "kit network curl <url>"})
			}

			results := []runner.Result{runner.Run(context.Background(), downloadCommand)}
			verified := false
			if results[0].Err == nil && sha256 != "" {
				if err := kitruntime.VerifySHA256(outputPath, sha256); err != nil {
					return err
				}
				verified = true
			}
			if results[0].Err == nil && executable {
				results = append(results, runner.Run(context.Background(), commands[1]))
			}
			summary := "Downloaded with curl.\nOutput: " + outputPath
			if verified {
				summary += "\nSHA256: verified"
			}
			return writeRunnerResults(cmd, "Download", summary, results, []string{"kit network curl " + url})
		},
	}
	command.Flags().StringVarP(&outputPath, "output", "o", "", "output file path")
	command.Flags().StringVar(&sha256, "sha256", "", "expected SHA256 checksum")
	command.Flags().IntVar(&retry, "retry", 2, "curl retry count")
	command.Flags().IntVar(&timeout, "timeout", 60, "curl max transfer time in seconds")
	command.Flags().BoolVar(&executable, "executable", false, "mark downloaded file executable")
	return command
}

func curlDownloadCommand(rawURL string, outputPath string, retry int, timeout int) runner.Command {
	if retry < 0 {
		retry = 0
	}
	if timeout <= 0 {
		timeout = 60
	}
	args := []string{"-fL", "--retry", strconv.Itoa(retry), "--connect-timeout", "10", "--max-time", strconv.Itoa(timeout), "--output", outputPath, rawURL}
	return runner.External("curl", args...)
}

func defaultDownloadOutput(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		base := urlpath.Base(parsed.Path)
		if base != "." && base != "/" && base != "" {
			return base
		}
	}
	return "download.out"
}

func newPortCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "port",
		Short: "List listening ports",
		RunE: func(cmd *cobra.Command, args []string) error {
			command := portListCommand()
			if opts.dryRun {
				return writeDryRun(cmd, "Listening Ports", []runner.Command{command}, []string{"kit network port kill <pid>"})
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Listening Ports", "Open listening sockets and owning processes when available.", []runner.Result{result}, []string{"kit network port kill <pid>"})
		},
	}
	command.AddCommand(newPortKillCommand())
	return command
}

func portListCommand() runner.Command {
	if detect.CommandExists("ss") {
		return runner.External("ss", "-ltnp")
	}
	if detect.CommandExists("lsof") {
		return runner.External("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	}
	return runner.External("netstat", "-ltnp")
}

func newPortKillCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "kill <pid>",
		Short: "Kill a process by PID after confirmation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			command := runner.External("kill", "-9", args[0])
			if opts.dryRun {
				return writeDryRun(cmd, "Port Kill", []runner.Command{command}, []string{"kit network port"})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "Port Kill", command, "프로세스를 강제로 종료합니다.", false)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "Port Kill", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Port Kill", "Process kill requested.", []runner.Result{result}, []string{"kit network port"})
		},
	}
}

func newTCPDumpCommand() *cobra.Command {
	var iface string
	var port string
	var host string
	var protocol string
	var expression string
	var outputFile string
	var count int
	command := &cobra.Command{
		Use:   "tcpdump",
		Short: "Capture packets with tcpdump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if iface == "" {
				selected, err := selectInterface(cmd)
				if err != nil {
					return err
				}
				iface = selected
			}
			if port == "" && host == "" && protocol == "" && expression == "" {
				prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
				filterType, err := prompt.Select("필터 타입을 선택하세요", []builder.Choice{
					{Label: "제한 없음", Value: "none"},
					{Label: "port", Value: "port"},
					{Label: "host", Value: "host"},
					{Label: "protocol", Value: "protocol"},
					{Label: "raw expression", Value: "raw"},
				}, 0)
				if err != nil {
					return err
				}
				switch filterType {
				case "port":
					port, err = prompt.Ask("포트 번호", "443")
				case "host":
					host, err = prompt.Ask("호스트/IP", "")
				case "protocol":
					protocol, err = prompt.Select("프로토콜", []builder.Choice{
						{Label: "tcp", Value: "tcp"},
						{Label: "udp", Value: "udp"},
						{Label: "icmp", Value: "icmp"},
					}, 0)
				case "raw":
					expression, err = prompt.Ask("tcpdump filter expression", "port 443")
				}
				if err != nil {
					return err
				}
			}
			if outputFile == "" && !cmd.Flags().Changed("write") {
				prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
				save, err := prompt.Confirm("파일로 저장할까요?", false)
				if err != nil {
					return err
				}
				if save {
					outputFile, err = prompt.Ask("저장 파일명", "capture.pcap")
					if err != nil {
						return err
					}
				}
			}
			parts := []string{"sudo", "tcpdump", "-i", runner.Quote(iface), "-c", strconv.Itoa(count)}
			filter := tcpdumpFilter(port, host, protocol, expression)
			if filter != "" {
				parts = append(parts, filter)
			}
			if outputFile != "" {
				parts = append(parts, "-w", runner.Quote(outputFile))
			}
			command := runner.Shell(strings.Join(parts, " "))
			if opts.dryRun {
				return writeDryRun(cmd, "TCP Packet Capture", []runner.Command{command}, []string{"kit network", "kit network port"})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "TCP Packet Capture", command, "This command requires sudo.", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "TCP Packet Capture", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "TCP Packet Capture", "Packet capture finished or stopped.", []runner.Result{result}, []string{"kit network", "kit network port"})
		},
	}
	command.Flags().StringVarP(&iface, "interface", "i", "", "network interface")
	command.Flags().StringVar(&port, "port", "", "port filter")
	command.Flags().StringVar(&host, "host", "", "host filter")
	command.Flags().StringVar(&protocol, "protocol", "", "protocol filter")
	command.Flags().StringVar(&expression, "expr", "", "raw tcpdump expression")
	command.Flags().StringVarP(&outputFile, "write", "w", "", "pcap output file")
	command.Flags().IntVarP(&count, "count", "c", 20, "packet count")
	return command
}

func tcpdumpFilter(port string, host string, protocol string, expression string) string {
	if expression != "" {
		return expression
	}
	if port != "" {
		return "port " + runner.Quote(port)
	}
	if host != "" {
		return "host " + runner.Quote(host)
	}
	if protocol != "" {
		return protocol
	}
	return ""
}

func selectInterface(cmd *cobra.Command) (string, error) {
	interfaces, err := detect.Interfaces()
	if err != nil {
		return "", err
	}
	choices := make([]builder.Choice, 0, len(interfaces))
	defaultIndex := 0
	defaultSet := false
	for _, item := range interfaces {
		label := item.Name
		if len(item.Addresses) > 0 {
			label += " " + strings.Join(item.Addresses, ",")
		}
		choices = append(choices, builder.Choice{Label: label, Value: item.Name})
		if item.Up && !item.Loopback && !defaultSet {
			defaultIndex = len(choices) - 1
			defaultSet = true
		}
	}
	if len(choices) == 0 {
		return "", fmt.Errorf("no network interfaces detected")
	}
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	return prompt.Select("사용할 인터페이스를 선택하세요", choices, defaultIndex)
}
