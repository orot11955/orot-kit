package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const DefaultTimeout = 30 * time.Second

type Command struct {
	Name    string
	Args    []string
	Display string
	Shell   bool
	Timeout time.Duration
}

type Result struct {
	Command  string
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Err      error
}

func External(name string, args ...string) Command {
	return Command{Name: name, Args: args}
}

func Shell(command string) Command {
	return Command{Display: command, Shell: true}
}

func (c Command) String() string {
	if c.Display != "" {
		return c.Display
	}
	parts := []string{Quote(c.Name)}
	for _, arg := range c.Args {
		parts = append(parts, Quote(arg))
	}
	return strings.Join(parts, " ")
}

func Run(ctx context.Context, c Command) Result {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var cmd *exec.Cmd
	if c.Shell {
		cmd = exec.CommandContext(runCtx, "sh", "-c", c.String())
	} else {
		cmd = exec.CommandContext(runCtx, c.Name, c.Args...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Command:  c.String(),
		Stdout:   strings.TrimRight(stdout.String(), "\n"),
		Stderr:   strings.TrimRight(stderr.String(), "\n"),
		ExitCode: 0,
		Duration: time.Since(start),
		Err:      err,
	}

	if err != nil {
		result.ExitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			}
		}
		if runCtx.Err() == context.DeadlineExceeded {
			result.Err = fmt.Errorf("command timed out after %s: %s", timeout, c.String())
		}
	}

	return result
}

func RunMany(ctx context.Context, commands []Command) []Result {
	results := make([]Result, 0, len(commands))
	for _, command := range commands {
		results = append(results, Run(ctx, command))
	}
	return results
}

func Quote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.IndexFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			strings.ContainsRune("-_./:@%+=,", r))
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
