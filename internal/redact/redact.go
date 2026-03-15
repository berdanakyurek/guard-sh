package redact

import (
	"fmt"
	"regexp"
)

// Redactor replaces sensitive patterns in a string with [REDACTED].
type Redactor struct {
	patterns []*regexp.Regexp
}

// New compiles the given regex patterns. Returns an error if any pattern is invalid.
func New(patterns []string) (*Redactor, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid redact pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &Redactor{patterns: compiled}, nil
}

// Redact replaces all pattern matches in s with [REDACTED].
func (r *Redactor) Redact(s string) string {
	for _, re := range r.patterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}
