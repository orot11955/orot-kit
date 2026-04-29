package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/output"
	"github.com/orot-dev/orot-kit/internal/runner"
	"github.com/spf13/cobra"
)

func commandStrings(commands []runner.Command) []string {
	values := make([]string, 0, len(commands))
	for _, command := range commands {
		values = append(values, command.String())
	}
	return values
}

func writeDryRun(cmd *cobra.Command, title string, commands []runner.Command, hint []string) error {
	return writer(cmd).Write(output.Result{
		Title:   title,
		Command: commandStrings(commands),
		Summary: "Dry run: command was not executed.",
		Hint:    hint,
	})
}

func confirmExecution(cmd *cobra.Command, title string, command runner.Command, summary string, defaultYes bool) (bool, error) {
	if err := writer(cmd).Write(output.Result{
		Title:   title,
		Command: []string{command.String()},
		Summary: summary,
	}); err != nil {
		return false, err
	}
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	return prompt.Confirm("실행할까요?", defaultYes)
}

func writeRunnerResults(cmd *cobra.Command, title string, summary string, results []runner.Result, hint []string) error {
	commands := make([]string, 0, len(results))
	sections := make([]output.Section, 0, len(results))
	for _, result := range results {
		commands = append(commands, result.Command)
		text := strings.TrimSpace(result.Stdout)
		if result.Stderr != "" {
			if text != "" {
				text += "\n"
			}
			text += "[stderr]\n" + result.Stderr
		}
		if result.Err != nil {
			if text != "" {
				text += "\n"
			}
			text += fmt.Sprintf("[exit %d] %v", result.ExitCode, result.Err)
		}
		if text == "" {
			text = "(no output)"
		}
		name := "Result"
		if len(results) > 1 {
			name = "Result: " + result.Command
		}
		sections = append(sections, output.Section{Name: name, Text: text})
	}
	return writer(cmd).Write(output.Result{
		Title:    title,
		Command:  commands,
		Summary:  summary,
		Sections: sections,
		Hint:     hint,
	})
}

func notImplementedCommand(use string, short string, title string, hints ...string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return writer(cmd).Write(output.Result{
				Title:   title,
				Summary: "This command is planned but not implemented in the current MVP.",
				Hint:    hints,
			})
		},
	}
}

func hiddenCommand(command *cobra.Command) *cobra.Command {
	command.Hidden = true
	return command
}
