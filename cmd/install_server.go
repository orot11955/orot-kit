package cmd

import (
	"net/http"
	"path/filepath"

	"github.com/orot-dev/orot-kit/internal/installer"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/spf13/cobra"
)

func registerInstallServerCommands(root *cobra.Command) {
	var addr string
	var binDir string
	var runtimeCacheDir string
	var baseURL string
	var assetsDir string
	var statsFile string
	command := &cobra.Command{
		Use:   "install-server",
		Short: "Serve kit installer endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			config := installer.Config{
				BinDir:          expandPath(binDir),
				RuntimeCacheDir: expandPath(runtimeCacheDir),
				BaseURL:         baseURL,
				AssetsDir:       expandPath(assetsDir),
				StatsFile:       expandPath(statsFile),
			}
			if opts.dryRun {
				return writer(cmd).Write(output.Result{
					Title:   "Install Server",
					Command: []string{"kit install-server --addr " + addr + " --bin-dir " + config.BinDir + " --runtime-cache-dir " + config.RuntimeCacheDir + " --assets-dir " + config.AssetsDir + " --stats-file " + config.StatsFile + " --base-url " + config.BaseURL},
					Summary: "Dry run: HTTP server was not started.",
					Hint:    []string{"curl " + baseURL + "/healthz", "curl " + baseURL + "/version", "curl -fsSL " + baseURL + "/install.sh | sh", "curl -fsSL " + baseURL + "/uninstall.sh | sh"},
				})
			}
			cmd.Printf("Title\n  Install Server\n\nResult\n  Listening on %s\n  Base URL: %s\n  Binaries: %s\n  Runtime Cache: %s\n  Assets: %s\n  Stats: %s\n\n", addr, baseURL, config.BinDir, config.RuntimeCacheDir, config.AssetsDir, config.StatsFile)
			return http.ListenAndServe(addr, installer.NewServerWithConfig(config))
		},
	}
	command.Flags().StringVar(&addr, "addr", ":8080", "listen address")
	command.Flags().StringVar(&binDir, "bin-dir", "dist", "directory containing kit-<os>-<arch> binaries")
	command.Flags().StringVar(&runtimeCacheDir, "runtime-cache-dir", defaultInstallServerRuntimeCacheDir(), "runtime archive cache directory")
	command.Flags().StringVar(&assetsDir, "assets-dir", "assets", "directory containing installer page assets")
	command.Flags().StringVar(&statsFile, "stats-file", defaultInstallServerStatsFile(), "download counter JSON file")
	command.Flags().StringVar(&baseURL, "base-url", "http://localhost:8080", "public base URL used by install.sh and metadata")
	root.AddCommand(command)
}

func defaultInstallServerRuntimeCacheDir() string {
	return filepath.Join("~", ".kit-server", "cache", "runtimes")
}

func defaultInstallServerStatsFile() string {
	return filepath.Join("~", ".kit-server", "download-stats.json")
}
