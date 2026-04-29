package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Writer struct {
	out  io.Writer
	json bool
}

func NewWriter(out io.Writer, jsonOutput bool) Writer {
	return Writer{out: out, json: jsonOutput}
}

func (w Writer) Write(result Result) error {
	if w.json {
		encoder := json.NewEncoder(w.out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	writeBlock := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		fmt.Fprintln(w.out, title)
		for _, line := range lines {
			for _, part := range strings.Split(strings.TrimRight(line, "\n"), "\n") {
				fmt.Fprintf(w.out, "  %s\n", part)
			}
		}
		fmt.Fprintln(w.out)
	}

	if result.Title != "" {
		writeBlock("Title", []string{result.Title})
	}
	writeBlock("Command", result.Command)
	if result.Summary != "" {
		writeBlock("Summary", []string{result.Summary})
	}
	if result.Result != "" {
		writeBlock("Result", []string{result.Result})
	}
	for _, section := range result.Sections {
		lines := section.Rows
		if section.Text != "" {
			lines = append([]string{section.Text}, lines...)
		}
		writeBlock(section.Name, lines)
	}
	writeBlock("Hint", result.Hint)
	return nil
}
