package main

import (
	"regexp"
	"sort"
)

// Match represents a found text match in the tmux pane
type Match struct {
	Text string
	Line int    // line number in captured pane (1-based)
	Col  int    // column position (0-based)
	Hint string // assigned hint character(s)
}

// patternDef defines a named regex pattern
type patternDef struct {
	name    string
	pattern string
}

// Built-in patterns ordered by priority (more specific first)
var patterns = []patternDef{
	{"url", `https?://[^\s\])'">]+`},
	// homepath: ~ prefixed paths (e.g. ~/repos/tmux-hint, ~user/dir)
	{"homepath", `~[a-zA-Z0-9_.-]*/[^\s,;'")\]]+`},
	{"path", `(?:^|[\s,])((?:/[^\s/,;'")\]]+)+)`},
	{"ip", `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`},
	{"uuid", `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`},
	{"git-sha", `\b[0-9a-f]{7,40}\b`},
	// filename: common file extensions (e.g. CLAUDE.md, test.sh, package.json)
	{"filename", `\b[\w][\w.-]*\.(?:go|py|js|ts|rb|sh|md|json|yaml|yml|toml|txt|log|conf|cfg|ini|lock|mod|sum|rs|c|cpp|h|java|swift|kt|ex|exs|lua|vim|zsh|bash|fish|env|sql|html|css|xml)\b`},
	// dirname: directory names ending with / (e.g. openrouter/, repos/, go/)
	{"dirname", `\b[a-zA-Z][\w.-]*/`},
	// relpath: relative paths with at least one / (e.g. src/main.go)
	{"relpath", `\b[\w.-]+(?:/[\w.-]+)+/?\b`},
	{"number", `\b\d{4,}\b`},
}

// matchKey is used for deduplication
type matchKey struct {
	line int
	col  int
}

// FindMatches scans all lines for all patterns and returns deduplicated, sorted matches
func FindMatches(lines []Line) []Match {
	seen := make(map[matchKey]bool)
	var results []Match

	compiledPatterns := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.pattern)
		if err != nil {
			continue
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	for _, line := range lines {
		content := line.Content
		if content == "" {
			continue
		}

		for _, re := range compiledPatterns {
			idxs := re.FindAllStringSubmatchIndex(content, -1)
			for _, idx := range idxs {
				// Use the first capture group if present (e.g. path pattern), else full match
				start, end := idx[0], idx[1]
				if len(idx) >= 4 && idx[2] >= 0 {
					start, end = idx[2], idx[3]
				}

				text := content[start:end]
				if text == "" {
					continue
				}

				key := matchKey{line: line.Number, col: start}
				if seen[key] {
					continue
				}
				seen[key] = true

				results = append(results, Match{
					Text: text,
					Line: line.Number,
					Col:  start,
				})
			}
		}
	}

	// Sort by position: top-to-bottom, left-to-right
	sort.Slice(results, func(i, j int) bool {
		if results[i].Line != results[j].Line {
			return results[i].Line < results[j].Line
		}
		return results[i].Col < results[j].Col
	})

	results = removeContainedMatches(results)

	return results
}

// removeContainedMatches removes matches that are fully contained within a larger match on the same line.
// For example, if "~/repos" and "~/repos/tmux-hint/main.go" both match on the same line,
// the shorter "~/repos" is removed because it is a substring of the longer match.
func removeContainedMatches(matches []Match) []Match {
	out := make([]Match, 0, len(matches))
	for i, m := range matches {
		mEnd := m.Col + len(m.Text)
		dominated := false
		for j, other := range matches {
			if i == j || other.Line != m.Line {
				continue
			}
			otherEnd := other.Col + len(other.Text)
			if other.Col <= m.Col && mEnd <= otherEnd && (other.Col < m.Col || mEnd < otherEnd) {
				dominated = true
				break
			}
		}
		if !dominated {
			out = append(out, m)
		}
	}
	return out
}
