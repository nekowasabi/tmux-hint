package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: tmux-hint start <pane-id>")
			os.Exit(1)
		}
		cmdStart(os.Args[2])

	case "input":
		// Called from within display-popup to read key input
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: tmux-hint input <pane-id>")
			os.Exit(1)
		}
		cmdInput(os.Args[2])

	case "load-config":
		cmdLoadConfig()

	default:
		usage()
		os.Exit(1)
	}
}

// cmdStart is the main entry point for hint mode.
func cmdStart(paneID string) {
	// 1. Capture pane content
	lines, err := capturePaneLines(paneID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tmux-hint: capture pane failed: %v\n", err)
		os.Exit(1)
	}

	// 2. Find matches
	matches := FindMatches(lines)
	if len(matches) == 0 {
		tmuxDisplayMessage("tmux-hint: no matches found")
		return
	}

	// 3. Assign hints
	hints := GenerateHints(len(matches))
	for i := range matches {
		matches[i].Hint = hints[i]
	}

	// 4. Render overlay and show popup, get user input
	content := RenderOverlay(lines, matches)
	selectedHint, err := ShowPopup(paneID, content, matches)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tmux-hint: popup error: %v\n", err)
		return
	}
	if selectedHint == "" {
		return // cancelled
	}

	// 5. Execute action based on input
	executeAction(paneID, selectedHint, matches)
}

// cmdInput is called inside the display-popup process.
// Reads a hint from stdin and writes the result to stdout.
func cmdInput(_ string) {
	hint, err := ReadHint(nil)
	if err != nil {
		os.Exit(1)
	}
	if hint == "" {
		os.Exit(0)
	}
	fmt.Print(hint)
}

// executeAction finds the match for the selected hint and performs the appropriate action.
func executeAction(paneID, input string, matches []Match) {
	// vim-mode jump when input starts with 'v' followed by a hint
	vimMode := false
	hint := input
	if strings.HasPrefix(input, "v") && len(input) > 1 {
		vimMode = true
		hint = input[1:]
	}

	// Find match for hint
	var target *Match
	for i := range matches {
		if matches[i].Hint == hint {
			target = &matches[i]
			break
		}
	}
	if target == nil {
		tmuxDisplayMessage("tmux-hint: hint not found: " + hint)
		return
	}

	if vimMode {
		if err := JumpToCopyMode(paneID, target.Line, target.Col); err != nil {
			fmt.Fprintf(os.Stderr, "tmux-hint: vim jump failed: %v\n", err)
		}
	} else {
		// Copy to both system clipboard and tmux buffer
		if err := CopyToClipboard(target.Text); err != nil {
			fmt.Fprintf(os.Stderr, "tmux-hint: clipboard warning: %v\n", err)
		}
		if err := CopyToTmuxBuffer(target.Text); err != nil {
			fmt.Fprintf(os.Stderr, "tmux-hint: tmux buffer warning: %v\n", err)
		}
		tmuxDisplayMessage("tmux-hint: copied: " + truncate(target.Text, 40))
	}
}

// cmdLoadConfig sets up tmux key bindings for tmux-hint.
func cmdLoadConfig() {
	key := getTmuxOption("@tmux-hint-key", "F")

	selfPath, err := os.Executable()
	if err != nil {
		selfPath = "tmux-hint"
	}

	bindCmd := fmt.Sprintf("%s start #{pane_id}", selfPath)
	fmt.Printf("Setting tmux keybinding: prefix+%s -> tmux-hint start\n", key)

	if err := exec.Command("tmux", "bind-key", key, "run-shell", bindCmd).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tmux-hint: bind-key failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("tmux-hint: keybinding configured successfully")
}

// getTmuxOption retrieves a tmux option value, returning defaultVal if not set
func getTmuxOption(option, defaultVal string) string {
	out, err := exec.Command("tmux", "show-option", "-gqv", option).Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return defaultVal
	}
	return strings.TrimSpace(string(out))
}

// truncate shortens a string to maxLen chars, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func usage() {
	fmt.Println("tmux-hint - fast text hint selector for tmux")
	fmt.Println("Usage:")
	fmt.Println("  tmux-hint start <pane-id>   - start hint mode")
	fmt.Println("  tmux-hint input <pane-id>   - read hint input (used internally by popup)")
	fmt.Println("  tmux-hint load-config        - configure tmux keybindings")
}
