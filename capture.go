package main

import (
	"os/exec"
	"regexp"
	"strings"
)

// Line represents a single line from the tmux pane with its line number
type Line struct {
	Number  int
	Content string
}

// ansiEscape matches ANSI/VT100 escape sequences
var ansiEscape = regexp.MustCompile(`\x1b(\[[0-9;]*[mGKHFABCDSTJsu]|\][^\x07]*\x07|[()][AB012])`)

// capturePaneLines captures the visible content of a tmux pane and returns it as lines
func capturePaneLines(paneID string) ([]Line, error) {
	args := []string{"capture-pane", "-p", "-J"}
	if paneID != "" {
		args = append(args, "-t", paneID)
	}

	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return nil, err
	}

	raw := string(out)
	rawLines := strings.Split(raw, "\n")

	lines := make([]Line, 0, len(rawLines))
	for i, l := range rawLines {
		cleaned := stripANSI(l)
		lines = append(lines, Line{
			Number:  i + 1,
			Content: cleaned,
		})
	}

	return lines, nil
}

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}
