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
	prompts          map[string]string                  // per-provider prompt overrides
	patternRedactors map[string]*redact.Redactor        // per-provider pattern redactors
	entropyRedactors map[string]*redact.EntropyRedactor // per-provider entropy redactors
	debug            io.Writer
}

func NewMulti(names []string, providers []guard.Provider, prompts map[string]string, patternRedactors map[string]*redact.Redactor, entropyRedactors map[string]*redact.EntropyRedactor, debug io.Writer) *Multi {
	return &Multi{
		names:            names,
		providers:        providers,
		prompts:          prompts,
		patternRedactors: patternRedactors,
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
		if pr, ok := m.patternRedactors[name]; ok && pr != nil {
			cmd = pr.Redact(cmd)
		}
		if er, ok := m.entropyRedactors[name]; ok && er != nil {
			cmd = er.Redact(cmd)
		}

		if m.debug != nil {
			patStatus := fmt.Sprintf("%s○ off%s", dbgDim, dbgReset)
			if _, ok := m.patternRedactors[name]; ok {
				patStatus = fmt.Sprintf("%s● on%s", dbgGreen, dbgReset)
			}
			entStatus := fmt.Sprintf("%s○ off%s", dbgDim, dbgReset)
			if _, ok := m.entropyRedactors[name]; ok {
				entStatus = fmt.Sprintf("%s● on%s", dbgGreen, dbgReset)
			}
			fmt.Fprintf(m.debug, "  %s%-10s%s  pattern %s  entropy %s\n", dbgCyan, name, dbgReset, patStatus, entStatus)
			if cmd != command {
				fmt.Fprintf(m.debug, "  %s            → %q%s\n", dbgDim, cmd, dbgReset)
			}
		}

		start := time.Now()
		result, err := p.Query(ctx, prompt, cmd)
		elapsed := time.Since(start).Milliseconds()
		if err == nil {
			if m.debug != nil {
				fmt.Fprintf(m.debug, "  %s            ✓ ok %s(%dms)%s\n", dbgGreen, dbgDim, elapsed, dbgReset)
			}
			return result, nil
		}
		if m.debug != nil {
			fmt.Fprintf(m.debug, "  %s            ✗ %s %s(%dms), trying next%s\n", dbgRed, err.Error(), dbgDim, elapsed, dbgReset)
		}
	}
	return "", errors.New("all providers failed")
}
