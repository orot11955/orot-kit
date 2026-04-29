package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"time"

	"github.com/orot-dev/orot-kit/internal/config"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	baseURL string
	binPath string
	retry   int
	timeout int
}

type updatePlan struct {
	Binary string
	URL    string
	Target string
}

func registerUpdateCommand(root *cobra.Command) {
	options := updateOptions{retry: 2, timeout: 60}
	command := &cobra.Command{
		Use:   "update",
		Short: "Update kit binary from the install server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, options)
		},
	}
	command.Flags().StringVar(&options.baseURL, "base-url", "", "kit install server base URL")
	command.Flags().StringVar(&options.binPath, "bin", "", "kit binary path to replace")
	command.Flags().IntVar(&options.retry, "retry", 2, "curl retry count")
	command.Flags().IntVar(&options.timeout, "timeout", 60, "curl max transfer time in seconds")
	root.AddCommand(command)
}

func runUpdate(cmd *cobra.Command, options updateOptions) error {
	plan, err := buildUpdatePlan(options)
	if err != nil {
		return err
	}
	if opts.dryRun {
		return writeDryRun(cmd, "Update", updateCommands(plan, "<temp-file>", options), []string{"kit update --base-url " + planBaseURL(plan.URL), "kit -v"})
	}

	if err := os.MkdirAll(filepath.Dir(plan.Target), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(plan.Target), ".kit-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	download := curlDownloadCommand(plan.URL, tmpPath, options.retry, options.timeout)
	download.Timeout = updateRunnerTimeout(options.timeout)
	result := runner.Run(context.Background(), download)
	if result.Err != nil {
		return fmt.Errorf("update download failed: %s", runnerResultMessage(result))
	}

	mode := os.FileMode(0o755)
	if info, err := os.Stat(plan.Target); err == nil {
		mode = info.Mode().Perm() | 0o111
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, plan.Target); err != nil {
		return fmt.Errorf("could not replace %s: %w", plan.Target, err)
	}
	cleanup = false

	return writer(cmd).Write(output.Result{
		Title:   "Update",
		Command: commandStrings(updateCommands(plan, tmpPath, options)),
		Summary: "kit binary updated.",
		Result:  fmt.Sprintf("Binary: %s\nSource: %s\nPath: %s", plan.Binary, plan.URL, plan.Target),
		Hint:    []string{"kit -v", "kit info"},
	})
}

func buildUpdatePlan(options updateOptions) (updatePlan, error) {
	baseURL, err := resolveInstallBaseURL(options.baseURL)
	if err != nil {
		return updatePlan{}, err
	}
	binary, err := kitBinaryName(stdruntime.GOOS, stdruntime.GOARCH)
	if err != nil {
		return updatePlan{}, err
	}
	target, err := updateTargetPath(options.binPath)
	if err != nil {
		return updatePlan{}, err
	}
	return updatePlan{
		Binary: binary,
		URL:    updateDownloadURL(baseURL, binary),
		Target: target,
	}, nil
}

func updateCommands(plan updatePlan, tempPath string, options updateOptions) []runner.Command {
	return []runner.Command{
		curlDownloadCommand(plan.URL, tempPath, options.retry, options.timeout),
		runner.External("chmod", "+x", tempPath),
		runner.External("mv", tempPath, plan.Target),
	}
}

func resolveInstallBaseURL(flagValue string) (string, error) {
	if flagValue != "" {
		return strings.TrimRight(flagValue, "/"), nil
	}
	if value := os.Getenv("KIT_BASE_URL"); value != "" {
		return strings.TrimRight(value, "/"), nil
	}
	if value := os.Getenv("KIT_INSTALL_BASE_URL"); value != "" {
		return strings.TrimRight(value, "/"), nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Server.InstallBaseURL != "" {
		return strings.TrimRight(cfg.Server.InstallBaseURL, "/"), nil
	}
	return "http://localhost:8080", nil
}

func kitBinaryName(goos string, goarch string) (string, error) {
	switch goos {
	case "darwin", "linux":
	default:
		return "", fmt.Errorf("unsupported OS for kit update: %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("unsupported architecture for kit update: %s", goarch)
	}
	return "kit-" + goos + "-" + goarch, nil
}

func updateDownloadURL(baseURL string, binary string) string {
	raw := strings.TrimRight(baseURL, "/") + "/bin/" + binary
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw + "?update=1"
	}
	query := parsed.Query()
	query.Set("update", "1")
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func updateTargetPath(binPath string) (string, error) {
	if binPath == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", err
		}
		binPath = executable
	}
	absolute, err := filepath.Abs(expandPath(binPath))
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func updateRunnerTimeout(timeout int) time.Duration {
	if timeout <= 0 {
		timeout = 60
	}
	return time.Duration(timeout+15) * time.Second
}

func runnerResultMessage(result runner.Result) string {
	if strings.TrimSpace(result.Stderr) != "" {
		return strings.TrimSpace(result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return strings.TrimSpace(result.Stdout)
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	return "unknown error"
}

func planBaseURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(strings.TrimSuffix(raw, "/bin/"+filepath.Base(raw)), "/")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/bin/"+filepath.Base(parsed.Path))
	parsed.RawQuery = ""
	return strings.TrimRight(parsed.String(), "/")
}
