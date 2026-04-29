package runtime

import (
	"context"
	"strings"

	"github.com/orot-dev/orot-kit/internal/detect"
	"github.com/orot-dev/orot-kit/internal/runner"
)

type Current struct {
	Name    string
	Command string
	Version string
	Path    string
}

func DetectCurrent(ctx context.Context, name string) Current {
	specs := detect.RuntimeSpecs()
	spec, ok := specs[name]
	if !ok {
		return Current{Name: name}
	}

	current := Current{Name: name, Command: spec.Command}
	if detect.CommandExists(spec.Command) {
		version := runner.Run(ctx, runner.External(spec.Command, spec.VersionArgs...))
		current.Version = strings.TrimSpace(firstNonEmpty(version.Stdout, version.Stderr))
		path := runner.Run(ctx, runner.External(spec.PathCommand, spec.Command))
		current.Path = strings.TrimSpace(path.Stdout)
	}
	return current
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
