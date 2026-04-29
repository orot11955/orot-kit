package secret

import (
	"context"
	"fmt"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/runner"
)

func CopyToClipboard(ctx context.Context, value string) error {
	command := detect.FirstCommand("pbcopy", "xclip", "wl-copy")
	if command == "" {
		return fmt.Errorf("clipboard command not found")
	}
	var shell string
	switch command {
	case "pbcopy":
		shell = fmt.Sprintf("printf %%s %s | pbcopy", runner.Quote(value))
	case "xclip":
		shell = fmt.Sprintf("printf %%s %s | xclip -selection clipboard", runner.Quote(value))
	case "wl-copy":
		shell = fmt.Sprintf("printf %%s %s | wl-copy", runner.Quote(value))
	}
	result := runner.Run(ctx, runner.Shell(shell))
	if result.Err != nil {
		return result.Err
	}
	return nil
}
