package redact

import (
	"math"
	"strings"
)

// Entropy computes the Shannon entropy of s in bits per character.
func Entropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	freq := make(map[rune]float64)
	for _, c := range s {
		freq[c]++
	}
	n := float64(len([]rune(s)))
	var h float64
	for _, count := range freq {
		p := count / n
		h -= p * math.Log2(p)
	}
	return h
}

// EntropyRedactor redacts whitespace-delimited tokens whose Shannon entropy
// exceeds a threshold, replacing them with [REDACTED].
type EntropyRedactor struct {
	threshold float64
	minLength int
}

// NewEntropyRedactor creates an EntropyRedactor with the given threshold (bits/char)
// and minimum token length. Tokens shorter than minLength are never redacted.
func NewEntropyRedactor(threshold float64, minLength int) *EntropyRedactor {
	return &EntropyRedactor{threshold: threshold, minLength: minLength}
}

// Redact replaces whitespace-delimited tokens with entropy above the threshold
// with [REDACTED], preserving all surrounding whitespace.
func (e *EntropyRedactor) Redact(s string) string {
	var result strings.Builder
	i := 0
	for i < len(s) {
		if isSpaceByte(s[i]) {
			result.WriteByte(s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && !isSpaceByte(s[j]) {
			j++
		}
		token := s[i:j]
		if len(token) >= e.minLength && Entropy(token) >= e.threshold {
			result.WriteString("[REDACTED]")
		} else {
			result.WriteString(token)
		}
		i = j
	}
	return result.String()
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
