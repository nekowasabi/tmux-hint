package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ANSI color codes
const (
	colorReset     = "\033[0m"
	colorHint      = "\033[1;33m" // bold yellow - hint chars
	colorMatchText = "\033[1;31m" // bold red - matched text
)

// RenderOverlay creates the display content with hints highlighted using ANSI escape codes.
// Each match is annotated: the hint char is shown in bold yellow, the remaining text in bold red.
func RenderOverlay(lines []Line, matches []Match) string {
	// Build a map: lineNumber -> list of matches on that line
	type annotation struct {
		col int
		m   Match
	}
	lineMatches := make(map[int][]annotation)
	for _, m := range matches {
		lineMatches[m.Line] = append(lineMatches[m.Line], annotation{col: m.Col, m: m})
	}

	var sb strings.Builder
	for _, line := range lines {
		annots, hasMatch := lineMatches[line.Number]
		if !hasMatch {
			sb.WriteString(line.Content)
			sb.WriteByte('\n')
			continue
		}

		content := line.Content
		offset := 0
		// Process annotations in column order (already sorted from FindMatches)
		for _, ann := range annots {
			m := ann.m
			start := m.Col
			end := start + len(m.Text)

			// Bounds check
			if start > len(content) {
				start = len(content)
			}
			if end > len(content) {
				end = len(content)
			}

			// Write text before this match (unadjusted)
			if start > offset {
				sb.WriteString(content[offset:start])
			}

			// Overwrite the first len(hint) bytes of the match with the hint char(s).
			// This keeps line width identical to the original, preserving column alignment.
			if len(m.Hint) > 0 && start < len(content) {
				hintLen := len(m.Hint)
				sb.WriteString(colorHint)
				sb.WriteString(m.Hint)
				sb.WriteString(colorReset)
				// Remaining match text after the hint chars
				if hintLen < len(m.Text) {
					sb.WriteString(colorMatchText)
					sb.WriteString(m.Text[hintLen:])
					sb.WriteString(colorReset)
				}
			}

			offset = end
		}

		// Write remaining text after last match
		if offset < len(content) {
			sb.WriteString(content[offset:])
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ShowPopup displays the rendered overlay in a tmux display-popup and reads the user's hint input.
// Returns the hint string selected by the user, or empty string if cancelled.
func ShowPopup(paneID string, content string, matches []Match) (string, error) {
	// Write rendered content to a temp file
	tmpContent, err := os.CreateTemp("", "tmux-hint-content-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp content file: %w", err)
	}
	defer os.Remove(tmpContent.Name())

	if _, err := tmpContent.WriteString(content); err != nil {
		tmpContent.Close()
		return "", fmt.Errorf("write content: %w", err)
	}
	tmpContent.Close()

	// Write a temp file that will receive the selected hint
	tmpResult, err := os.CreateTemp("", "tmux-hint-result-*.txt")
	if err != nil {
		return "", fmt.Errorf("create temp result file: %w", err)
	}
	resultPath := tmpResult.Name()
	tmpResult.Close()
	defer os.Remove(resultPath)

	// The popup script: show content, then read hint keys
	// We use 'cat' to display, then our own binary in 'input' mode to capture keys
	selfPath, err := os.Executable()
	if err != nil {
		selfPath = "tmux-hint"
	}

	script := fmt.Sprintf(
		"cat %s; %s input %s > %s 2>/dev/null",
		shellQuote(tmpContent.Name()),
		shellQuote(selfPath),
		shellQuote(paneID),
		shellQuote(resultPath),
	)

	// Use full pane size so captured content fits without scrolling
	cmd := exec.Command("tmux", "display-popup", "-E",
		"-w", "100%",
		"-h", "100%",
		"-x", "0",
		"-y", "0",
		"-b", "none",
		"bash", "-c", script,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run popup (blocks until user quits or selects)
	_ = cmd.Run() // ignore exit code; user might press q/Esc

	// Read result
	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		return "", nil // no selection is not an error
	}

	result := strings.TrimSpace(string(resultBytes))
	return result, nil
}

// ReadHint reads hint characters interactively from stdin using raw terminal mode.
// Returns the complete hint string (1 or 2 chars) or empty on cancel.
// When matches is nil (called from cmdInput subprocess), validation is skipped and
// any non-cancel keystroke is returned; a second char is awaited with a 500ms timeout.
func ReadHint(matches []Match) (string, error) {
	// Build set of valid single-char prefixes and full hints
	validHints := make(map[string]bool)
	prefixes := make(map[string]bool)
	for _, m := range matches {
		validHints[m.Hint] = true
		if len(m.Hint) > 1 {
			prefixes[string(m.Hint[0])] = true
		}
	}
	// noValidation: called from popup subprocess (cmdInput) without match context
	noValidation := len(validHints) == 0

	// Put terminal in raw mode using stty
	sttyRaw := exec.Command("stty", "raw", "-echo")
	sttyRaw.Stdin = os.Stdin
	if err := sttyRaw.Run(); err != nil {
		if noValidation {
			return readRawAny()
		}
		return readHintNormal(validHints, prefixes)
	}

	// Restore terminal on exit
	defer func() {
		restore := exec.Command("stty", "sane")
		restore.Stdin = os.Stdin
		_ = restore.Run()
	}()

	buf := make([]byte, 1)

	// Read first key
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return "", nil
	}

	ch1 := buf[0]

	// Cancel keys: q or ESC
	if ch1 == 'q' || ch1 == 27 {
		return "", nil
	}

	first := string(ch1)

	// noValidation mode: read up to 2 chars with timeout for multi-char hints.
	// Special case: if first char is 'v' (vim-jump prefix), read one more char
	// immediately (blocking), then attempt an optional 3rd char with timeout.
	// This ensures "v"+"as" (2-char hint) is correctly captured as "vas".
	if noValidation {
		if ch1 == 'v' {
			// Read hint's first char (blocking – user must press it)
			b1 := make([]byte, 1)
			n2, err2 := os.Stdin.Read(b1)
			if err2 != nil || n2 == 0 {
				return "", nil
			}
			hc1 := b1[0]
			if hc1 == 'q' || hc1 == 27 {
				return "", nil
			}
			// Try to read optional 2nd hint char with timeout
			third := make(chan byte, 1)
			go func() {
				b := make([]byte, 1)
				if n3, err3 := os.Stdin.Read(b); err3 == nil && n3 > 0 {
					third <- b[0]
				}
			}()
			select {
			case c := <-third:
				if c == 'q' || c == 27 {
					return "v" + string(hc1), nil
				}
				return "v" + string(hc1) + string(c), nil
			case <-time.After(500 * time.Millisecond):
				return "v" + string(hc1), nil
			}
		}

		second := make(chan byte, 1)
		go func() {
			b := make([]byte, 1)
			if n2, err2 := os.Stdin.Read(b); err2 == nil && n2 > 0 {
				second <- b[0]
			}
		}()
		select {
		case c := <-second:
			if c == 'q' || c == 27 {
				return first, nil // treat second cancel as end; return first char
			}
			return first + string(c), nil
		case <-time.After(500 * time.Millisecond):
			return first, nil
		}
	}

	// Validation mode: check against known hints
	if validHints[first] {
		if prefixes[first] {
			n2, err := os.Stdin.Read(buf)
			if err != nil || n2 == 0 {
				return first, nil
			}
			ch2 := buf[0]
			if ch2 == 'q' || ch2 == 27 {
				return "", nil
			}
			two := first + string(ch2)
			if validHints[two] {
				return two, nil
			}
			return first, nil
		}
		return first, nil
	}

	if prefixes[first] {
		n2, err := os.Stdin.Read(buf)
		if err != nil || n2 == 0 {
			return "", nil
		}
		ch2 := buf[0]
		if ch2 == 'q' || ch2 == 27 {
			return "", nil
		}
		two := first + string(ch2)
		if validHints[two] {
			return two, nil
		}
	}

	return "", nil
}

// readRawAny reads any single non-cancel byte without terminal validation (fallback)
func readRawAny() (string, error) {
	buf := make([]byte, 1)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return "", nil
	}
	if buf[0] == 'q' || buf[0] == 27 {
		return "", nil
	}
	return string(buf[0]), nil
}

// readHintNormal reads hint in normal (non-raw) mode as fallback
func readHintNormal(validHints map[string]bool, prefixes map[string]bool) (string, error) {
	buf := make([]byte, 4)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return "", nil
	}
	input := strings.TrimSpace(string(buf[:n]))
	if len(input) == 0 {
		return "", nil
	}
	if input == "q" || input == "\x1b" {
		return "", nil
	}
	if validHints[input] {
		return input, nil
	}
	if len(input) >= 2 && validHints[input[:2]] {
		return input[:2], nil
	}
	if len(input) >= 1 && validHints[input[:1]] {
		return input[:1], nil
	}
	return "", nil
}

// shellQuote wraps a string in single quotes, escaping internal single quotes
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
