package main

import (
	"strings"
	"testing"
)

func TestRenderOverlaySkipsOverlap(t *testing.T) {
	lines := []Line{
		{Number: 1, Content: "hello world test"},
	}
	// Two overlapping matches: "hello world" (col 0-11) and "world" (col 6-11)
	// The second match starts before the end of the first, so it should be skipped.
	matches := []Match{
		{Text: "hello world", Line: 1, Col: 0, Hint: "a"},
		{Text: "world", Line: 1, Col: 6, Hint: "b"},
	}

	result := RenderOverlay(lines, matches)

	// "world" should not appear twice in the output
	// Count occurrences of "world" in the raw text (strip ANSI codes)
	plain := stripANSI(result)
	if count := strings.Count(plain, "world"); count > 1 {
		t.Errorf("'world' appears %d times in output (expected at most 1): %q", count, plain)
	}

	// Hint "b" should NOT appear because the overlap guard should skip it
	if strings.Contains(result, colorHint+"b"+colorReset) {
		t.Error("hint 'b' should have been skipped due to overlap")
	}
}

func TestRenderOverlayHighlightLimit(t *testing.T) {
	longText := "~/works/invase-management/documents/web/INVASE-1799-remove-required-input-for-purchase-loan-modal.md"
	lines := []Line{
		{Number: 1, Content: longText},
	}
	matches := []Match{
		{Text: longText, Line: 1, Col: 0, Hint: "a"},
	}

	result := RenderOverlay(lines, matches)

	// The hint 'a' should be in yellow
	if !strings.Contains(result, colorHint+"a"+colorReset) {
		t.Error("hint 'a' not found in output")
	}

	// After the hint, only maxHighlightLen bytes should be red
	remaining := longText[1:] // text after hint "a"
	if len(remaining) <= maxHighlightLen {
		t.Skip("test text is not long enough to trigger highlight limit")
	}

	redPart := remaining[:maxHighlightLen]
	normalPart := remaining[maxHighlightLen:]

	// The red-highlighted portion should exist
	if !strings.Contains(result, colorMatchText+redPart+colorReset) {
		t.Error("expected red-highlighted portion not found")
	}

	// The normal portion should NOT be wrapped in red
	if strings.Contains(result, colorMatchText+normalPart) {
		t.Error("excess text should not be red-highlighted")
	}

	// The normal portion should still be in the output (just not colored)
	if !strings.Contains(result, normalPart) {
		t.Error("excess text is missing from output entirely")
	}
}

func TestRenderOverlayShortMatchFullyHighlighted(t *testing.T) {
	lines := []Line{
		{Number: 1, Content: "see /usr/local/bin ok"},
	}
	matches := []Match{
		{Text: "/usr/local/bin", Line: 1, Col: 4, Hint: "a"},
	}

	result := RenderOverlay(lines, matches)

	// Short match: entire remaining text should be red
	remaining := "/usr/local/bin"[1:] // after hint "a"
	if !strings.Contains(result, colorMatchText+remaining+colorReset) {
		t.Errorf("short match should be fully highlighted, got: %q", result)
	}
}

// stripANSI is defined in capture.go — reused here for tests
