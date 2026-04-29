package builder

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Prompt struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompt(in io.Reader, out io.Writer) Prompt {
	return Prompt{in: bufio.NewReader(in), out: out}
}

func (p Prompt) Ask(label string, defaultValue string) (string, error) {
	if defaultValue == "" {
		fmt.Fprintf(p.out, "? %s: ", label)
	} else {
		fmt.Fprintf(p.out, "? %s [%s]: ", label, defaultValue)
	}
	value, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func (p Prompt) Select(label string, choices []Choice, defaultIndex int) (string, error) {
	fmt.Fprintf(p.out, "? %s:\n", label)
	for index, choice := range choices {
		fmt.Fprintf(p.out, "  %d. %s\n", index+1, choice.Label)
	}
	value, err := p.Ask("선택", strconv.Itoa(defaultIndex+1))
	if err != nil {
		return "", err
	}
	selected, err := strconv.Atoi(value)
	if err != nil || selected < 1 || selected > len(choices) {
		return "", fmt.Errorf("invalid selection: %s", value)
	}
	return choices[selected-1].Value, nil
}
