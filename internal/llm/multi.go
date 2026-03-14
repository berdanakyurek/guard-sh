package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Berdan/guard-sh/internal/guard"
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
	providers []guard.Provider
	names     []string
	debug     io.Writer
}

func NewMulti(names []string, providers []guard.Provider, debug io.Writer) *Multi {
	return &Multi{names: names, providers: providers, debug: debug}
}

func (m *Multi) Query(ctx context.Context, systemPrompt, command string) (string, error) {
	for i, p := range m.providers {
		name := m.names[i]
		if m.debug != nil {
			fmt.Fprintf(m.debug, "  %s%-10s%s", dbgCyan, name, dbgReset)
		}
		start := time.Now()
		result, err := p.Query(ctx, systemPrompt, command)
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
