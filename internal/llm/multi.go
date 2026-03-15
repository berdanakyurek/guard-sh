package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Berdan/guard-sh/internal/guard"
	"github.com/Berdan/guard-sh/internal/redact"
)

const (
	dbgReset = "\033[0m"
	dbgDim   = "\033[2m"
	dbgGreen = "\033[32m"
	dbgCyan  = "\033[36m"
	dbgRed   = "\033[31m"
)

// Multi tries each provider in order, falling back to the next on error.
type Multi struct {
	providers        []guard.Provider
	names            []string
	prompts          map[string]string // per-provider prompt overrides
	redactor         *redact.Redactor
	redactEnabled    map[string]bool // per-provider pattern redaction toggle
	entropyRedactors map[string]*redact.EntropyRedactor // per-provider entropy redactors
	debug            io.Writer
}

func NewMulti(names []string, providers []guard.Provider, prompts map[string]string, redactor *redact.Redactor, redactEnabled map[string]bool, entropyRedactors map[string]*redact.EntropyRedactor, debug io.Writer) *Multi {
	return &Multi{
		names:            names,
		providers:        providers,
		prompts:          prompts,
		redactor:         redactor,
		redactEnabled:    redactEnabled,
		entropyRedactors: entropyRedactors,
		debug:            debug,
	}
}

func (m *Multi) Query(ctx context.Context, systemPrompt, command string) (string, error) {
	for i, p := range m.providers {
		name := m.names[i]
		prompt := systemPrompt
		if override, ok := m.prompts[name]; ok {
			prompt = override
		}
		cmd := command
		if m.redactor != nil && m.redactEnabled[name] {
			cmd = m.redactor.Redact(cmd)
			if m.debug != nil && cmd != command {
				fmt.Fprintf(m.debug, "  %sredact (pattern)%s  %s→ %q%s\n", dbgDim, dbgReset, dbgDim, cmd, dbgReset)
			}
		}
		if er, ok := m.entropyRedactors[name]; ok && er != nil {
			before := cmd
			cmd = er.Redact(cmd)
			if m.debug != nil && cmd != before {
				fmt.Fprintf(m.debug, "  %sredact (entropy)%s  %s→ %q%s\n", dbgDim, dbgReset, dbgDim, cmd, dbgReset)
			}
		}
		if m.debug != nil {
			fmt.Fprintf(m.debug, "  %s%-10s%s", dbgCyan, name, dbgReset)
		}
		start := time.Now()
		result, err := p.Query(ctx, prompt, cmd)
		elapsed := time.Since(start).Milliseconds()
		if err == nil {
			if m.debug != nil {
				fmt.Fprintf(m.debug, "  %s✓ ok%s %s(%dms)%s\n", dbgGreen, dbgReset, dbgDim, elapsed, dbgReset)
			}
			return result, nil
		}
		if m.debug != nil {
			fmt.Fprintf(m.debug, "  %s✗ %s%s %s(%dms), trying next%s\n", dbgRed, err.Error(), dbgReset, dbgDim, elapsed, dbgReset)
		}
	}
	return "", errors.New("all providers failed")
}
