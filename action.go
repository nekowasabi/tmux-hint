package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
// macOS: pbcopy, Wayland: wl-copy, X11: xclip
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Detect Wayland vs X11
		if isWayland() {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clipboard copy failed: %w", err)
	}
	return nil
}

// isWayland returns true if running under a Wayland session
func isWayland() bool {
	out, err := exec.Command("sh", "-c", "echo $WAYLAND_DISPLAY").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// CopyToTmuxBuffer copies text to the tmux paste buffer (buffer0)
func CopyToTmuxBuffer(text string) error {
	cmd := exec.Command("tmux", "set-buffer", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux set-buffer failed: %w", err)
	}
	return nil
}

// PasteInPane pastes the tmux buffer content into the specified pane
func PasteInPane(paneID string) error {
	args := []string{"paste-buffer"}
	if paneID != "" {
		args = append(args, "-t", paneID)
	}
	cmd := exec.Command("tmux", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux paste-buffer failed: %w", err)
	}
	return nil
}

// JumpToCopyMode enters tmux copy-mode on the given pane and jumps to (lineNum, col).
func JumpToCopyMode(paneID string, lineNum int, col int) error {
	// Enter copy-mode
	enterArgs := []string{"copy-mode"}
	if paneID != "" {
		enterArgs = append(enterArgs, "-t", paneID)
	}
	if err := exec.Command("tmux", enterArgs...).Run(); err != nil {
		return fmt.Errorf("enter copy-mode: %w", err)
	}

	// Anchor to top of visible area.
	topArgs := []string{"send-keys", "-X", "top-line"}
	if paneID != "" {
		topArgs = []string{"send-keys", "-t", paneID, "-X", "top-line"}
	}
	if err := exec.Command("tmux", topArgs...).Run(); err != nil {
		return fmt.Errorf("top-line: %w", err)
	}

	// Move down lineNum-1 lines in a single command using -N repeat-count.
	if lineNum > 1 {
		downArgs := []string{"send-keys", "-X", "-N", fmt.Sprintf("%d", lineNum-1), "cursor-down"}
		if paneID != "" {
			downArgs = []string{"send-keys", "-t", paneID, "-X", "-N", fmt.Sprintf("%d", lineNum-1), "cursor-down"}
		}
		if err := exec.Command("tmux", downArgs...).Run(); err != nil {
			return fmt.Errorf("cursor-down: %w", err)
		}
	}

	// Move to start of line, then right to target column in a single command.
	solArgs := []string{"send-keys", "-X", "start-of-line"}
	if paneID != "" {
		solArgs = []string{"send-keys", "-t", paneID, "-X", "start-of-line"}
	}
	if err := exec.Command("tmux", solArgs...).Run(); err != nil {
		return fmt.Errorf("start-of-line: %w", err)
	}
	if col > 0 {
		rightArgs := []string{"send-keys", "-X", "-N", fmt.Sprintf("%d", col), "cursor-right"}
		if paneID != "" {
			rightArgs = []string{"send-keys", "-t", paneID, "-X", "-N", fmt.Sprintf("%d", col), "cursor-right"}
		}
		if err := exec.Command("tmux", rightArgs...).Run(); err != nil {
			return fmt.Errorf("cursor-right: %w", err)
		}
	}

	return nil
}

// getPaneHistorySize returns the number of lines currently in the tmux scrollback history for pane.
func getPaneHistorySize(paneID string) int {
	args := []string{"display-message", "-p", "#{history_size}"}
	if paneID != "" {
		args = []string{"display-message", "-t", paneID, "-p", "#{history_size}"}
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

// getPaneHeight returns the number of visible rows in the pane.
func getPaneHeight(paneID string) int {
	args := []string{"display-message", "-p", "#{pane_height}"}
	if paneID != "" {
		args = []string{"display-message", "-t", paneID, "-p", "#{pane_height}"}
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

// tmuxDisplayMessage shows a brief message in the tmux status bar
func tmuxDisplayMessage(msg string) {
	_ = exec.Command("tmux", "display-message", msg).Run()
}
