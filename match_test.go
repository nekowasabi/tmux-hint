package main

import (
	"testing"
)

func TestRemoveContainedMatches(t *testing.T) {
	tests := []struct {
		name     string
		input    []Match
		expected int // number of matches after filtering
	}{
		{
			name:     "no matches",
			input:    []Match{},
			expected: 0,
		},
		{
			name: "no overlap - different lines",
			input: []Match{
				{Text: "foo", Line: 1, Col: 0},
				{Text: "bar", Line: 2, Col: 0},
			},
			expected: 2,
		},
		{
			name: "child fully contained in parent on same line",
			input: []Match{
				{Text: "~/repos", Line: 1, Col: 5},
				{Text: "~/repos/tmux-hint/main.go", Line: 1, Col: 5},
			},
			expected: 1, // only the longer one survives
		},
		{
			name: "child contained at different offset",
			input: []Match{
				{Text: "tmux-hint", Line: 1, Col: 12},
				{Text: "~/repos/tmux-hint/main.go", Line: 1, Col: 5},
			},
			expected: 1,
		},
		{
			name: "identical matches keep one",
			input: []Match{
				{Text: "hello", Line: 1, Col: 0},
				{Text: "hello", Line: 1, Col: 0},
			},
			// identical range: neither strictly contains the other (Col == Col && End == End)
			expected: 2,
		},
		{
			name: "non-overlapping on same line",
			input: []Match{
				{Text: "foo", Line: 1, Col: 0},
				{Text: "bar", Line: 1, Col: 10},
			},
			expected: 2,
		},
		{
			name: "multiple children of one parent",
			input: []Match{
				{Text: "~/works/invase-management/documents/web/INVASE-1799.md", Line: 1, Col: 0},
				{Text: "invase-management", Line: 1, Col: 8},
				{Text: "documents/web/INVASE-1799.md", Line: 1, Col: 26},
				{Text: "INVASE-1799.md", Line: 1, Col: 40},
			},
			expected: 1, // only the full path survives
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeContainedMatches(tt.input)
			if len(result) != tt.expected {
				t.Errorf("removeContainedMatches() returned %d matches, want %d", len(result), tt.expected)
				for i, m := range result {
					t.Logf("  result[%d]: %q Line=%d Col=%d", i, m.Text, m.Line, m.Col)
				}
			}
		})
	}
}

func TestFindMatchesRemovesContained(t *testing.T) {
	// A long path that would produce multiple sub-matches (homepath, dirname, relpath, filename)
	lines := []Line{
		{Number: 1, Content: "editing ~/works/invase-management/documents/web/INVASE-1799-remove-required-input.md"},
	}
	matches := FindMatches(lines)

	// The homepath should dominate; sub-matches like dirname and relpath should be removed
	for i, a := range matches {
		for j, b := range matches {
			if i == j || a.Line != b.Line {
				continue
			}
			aEnd := a.Col + len(a.Text)
			bEnd := b.Col + len(b.Text)
			if b.Col <= a.Col && aEnd <= bEnd && (b.Col < a.Col || aEnd < bEnd) {
				t.Errorf("match %q (col=%d) is contained within %q (col=%d) but was not removed",
					a.Text, a.Col, b.Text, b.Col)
			}
		}
	}
}
