package main

// GenerateHints assigns hint strings to matches.
// Up to 26 matches: a-z
// Beyond 26: aa-az, ba-bz, ... (base-26 style)
func GenerateHints(count int) []string {
	hints := make([]string, 0, count)
	for i := 0; i < count; i++ {
		hints = append(hints, encodeHint(i))
	}
	return hints
}

// encodeHint converts an index to a hint string.
// 0->a, 1->b, ..., 25->z, 26->aa, 27->ab, ...
func encodeHint(n int) string {
	const alpha = "abcdefghijklmnopqrstuvwxyz"
	if n < 26 {
		return string(alpha[n])
	}

	// For n >= 26, generate multi-character hints
	// Treat as base-26 with the first character as prefix
	result := []byte{}
	n++ // shift so that 26 -> "aa" works correctly via modular arithmetic

	for n > 0 {
		n-- // adjust for 1-based indexing in base-26
		result = append([]byte{alpha[n%26]}, result...)
		n /= 26
	}

	return string(result)
}
