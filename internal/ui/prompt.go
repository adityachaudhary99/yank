package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Confirm asks a yes/no question and returns def on empty input. It reads one
// line from in and writes the prompt to out, so it is fully testable.
func Confirm(in io.Reader, out io.Writer, question string, def bool) bool {
	suffix := " [y/N] "
	if def {
		suffix = " [Y/n] "
	}
	fmt.Fprint(out, question+suffix)
	line, _ := bufio.NewReader(in).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

// Choose presents a numbered list and returns the chosen value, accepting either
// the option's number or its literal text. Invalid or empty input returns "".
func Choose(in io.Reader, out io.Writer, prompt string, options []string) string {
	fmt.Fprintln(out, prompt)
	for i, o := range options {
		fmt.Fprintf(out, "  %d) %s\n", i+1, o)
	}
	fmt.Fprint(out, "> ")
	line, _ := bufio.NewReader(in).ReadString('\n')
	choice := strings.TrimSpace(line)
	for i, o := range options {
		if choice == o || choice == fmt.Sprintf("%d", i+1) {
			return o
		}
	}
	return ""
}
