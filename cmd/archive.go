package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

func registerArchiveCommands(root *cobra.Command) {
	root.AddCommand(newArchiveCommand("archive"))
	root.AddCommand(newArchiveCommand("compress"))
	root.AddCommand(newExtractCommand())
}

func newArchiveCommand(use string) *cobra.Command {
	var format string
	var outputFile string
	command := &cobra.Command{
		Use:   use + " [target] [output]",
		Short: "Create an archive with a guided builder",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}
			if len(args) > 1 {
				outputFile = args[1]
			}
			if target == "" || format == "" || outputFile == "" {
				prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
				var err error
				if target == "" {
					target, err = prompt.Ask("압축할 대상", ".")
					if err != nil {
						return err
					}
				}
				if format == "" {
					format, err = prompt.Select("압축 형식", []builder.Choice{
						{Label: "tar.gz", Value: "tar.gz"},
						{Label: "zip", Value: "zip"},
						{Label: "gzip", Value: "gzip"},
					}, 0)
					if err != nil {
						return err
					}
				}
				if outputFile == "" {
					outputFile, err = prompt.Ask("출력 파일명", defaultArchiveName(target, format))
					if err != nil {
						return err
					}
				}
			}
			command, err := archiveCommand(target, format, outputFile)
			if err != nil {
				return err
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Archive", []runner.Command{command}, []string{"kit extract " + outputFile})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "Archive", command, "Archive will be created without deleting the source.", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "Archive", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Archive", "Archive command finished.", []runner.Result{result}, []string{"kit extract " + outputFile})
		},
	}
	command.Flags().StringVar(&format, "format", "", "tar.gz, zip, or gzip")
	command.Flags().StringVarP(&outputFile, "output", "o", "", "output archive path")
	return command
}

func newExtractCommand() *cobra.Command {
	var dest string
	command := &cobra.Command{
		Use:   "extract [archive] [dest]",
		Short: "Extract an archive with a guided builder",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			archivePath := ""
			if len(args) > 0 {
				archivePath = args[0]
			}
			if len(args) > 1 {
				dest = args[1]
			}
			if archivePath == "" || dest == "" {
				prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
				var err error
				if archivePath == "" {
					archivePath, err = prompt.Ask("압축 파일 경로", "")
					if err != nil {
						return err
					}
				}
				if dest == "" {
					dest, err = prompt.Ask("압축 해제 위치", ".")
					if err != nil {
						return err
					}
				}
			}
			command, err := extractCommand(archivePath, dest)
			if err != nil {
				return err
			}
			if opts.dryRun {
				return writeDryRun(cmd, "Extract", []runner.Command{command}, []string{"kit ls " + dest})
			}
			if !opts.yes {
				ok, err := confirmExecution(cmd, "Extract", command, "Archive contents will be written to "+dest+".", true)
				if err != nil {
					return err
				}
				if !ok {
					return canceled(cmd, "Extract", command)
				}
			}
			result := runner.Run(context.Background(), command)
			return writeRunnerResults(cmd, "Extract", "Extract command finished.", []runner.Result{result}, []string{"kit ls " + dest})
		},
	}
	command.Flags().StringVarP(&dest, "dest", "C", "", "destination directory")
	return command
}

func archiveCommand(target string, format string, outputFile string) (runner.Command, error) {
	if target == "" {
		return runner.Command{}, fmt.Errorf("archive target is required")
	}
	if outputFile == "" {
		outputFile = defaultArchiveName(target, format)
	}
	switch format {
	case "tar.gz", "tgz":
		return runner.External("tar", "-czf", outputFile, target), nil
	case "zip":
		return runner.External("zip", "-r", outputFile, target), nil
	case "gzip", "gz":
		return runner.Shell("gzip -c " + runner.Quote(target) + " > " + runner.Quote(outputFile)), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported archive format: %s", format)
	}
}

func extractCommand(archivePath string, dest string) (runner.Command, error) {
	if archivePath == "" {
		return runner.Command{}, fmt.Errorf("archive path is required")
	}
	if dest == "" {
		dest = "."
	}
	lower := strings.ToLower(archivePath)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return runner.Shell("mkdir -p " + runner.Quote(dest) + " && tar -xzf " + runner.Quote(archivePath) + " -C " + runner.Quote(dest)), nil
	case strings.HasSuffix(lower, ".zip"):
		return runner.External("unzip", archivePath, "-d", dest), nil
	case strings.HasSuffix(lower, ".gz"):
		outputPath := filepath.Join(dest, strings.TrimSuffix(filepath.Base(archivePath), ".gz"))
		return runner.Shell("mkdir -p " + runner.Quote(dest) + " && gzip -dc " + runner.Quote(archivePath) + " > " + runner.Quote(outputPath)), nil
	default:
		return runner.Command{}, fmt.Errorf("unsupported archive type: %s", archivePath)
	}
}

func defaultArchiveName(target string, format string) string {
	base := strings.TrimSuffix(filepath.Base(target), string(os.PathSeparator))
	if base == "." || base == string(os.PathSeparator) || base == "" {
		base = "archive"
	}
	switch format {
	case "zip":
		return base + ".zip"
	case "gzip", "gz":
		return base + ".gz"
	default:
		return base + ".tar.gz"
	}
}

func canceled(cmd *cobra.Command, title string, command runner.Command) error {
	return writer(cmd).Write(output.Result{
		Title:   title,
		Command: []string{command.String()},
		Summary: "Canceled.",
	})
}
