package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Berdan/guard-sh/internal/cache"
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
// Cache is keyed per-provider: "providerName:command".
type Multi struct {
	providers        []guard.Provider
	names            []string
	prompts          map[string]string                  // per-provider prompt overrides
	patternRedactors map[string]*redact.Redactor        // per-provider pattern redactors
	entropyRedactors map[string]*redact.EntropyRedactor // per-provider entropy redactors
	cache            *cache.Cache
	debug            io.Writer
}

func NewMulti(names []string, providers []guard.Provider, prompts map[string]string, patternRedactors map[string]*redact.Redactor, entropyRedactors map[string]*redact.EntropyRedactor, c *cache.Cache, debug io.Writer) *Multi {
	return &Multi{
		names:            names,
		providers:        providers,
		prompts:          prompts,
		patternRedactors: patternRedactors,
		entropyRedactors: entropyRedactors,
		cache:            c,
		debug:            debug,
	}
}

func (m *Multi) Query(ctx context.Context, systemPrompt, command string) (string, error) {
	for i, p := range m.providers {
		name := m.names[i]
		cacheKey := name + ":" + command

		// Per-provider cache check
		if m.cache != nil {
			if cached, ok := m.cache.Get(cacheKey); ok {
				if m.debug != nil {
					fmt.Fprintf(m.debug, "  %s%-10s%s  %scache ● hit%s  %s→ %q%s\n", dbgCyan, name, dbgReset, dbgGreen, dbgReset, dbgDim, cached, dbgReset)
				}
				return cached, nil
			}
		}

		prompt := systemPrompt
		if override, ok := m.prompts[name]; ok {
			prompt = override
		}
		cmd := command

		patStatus := fmt.Sprintf("%s○ off%s", dbgDim, dbgReset)
		if pr, ok := m.patternRedactors[name]; ok && pr != nil {
			patStatus = fmt.Sprintf("%s● on%s", dbgGreen, dbgReset)
			cmd = pr.Redact(cmd)
		}
		if m.debug != nil {
			if cmd != command {
				fmt.Fprintf(m.debug, "  %s%-10s%s  pattern %s  %s→ %q%s\n", dbgCyan, name, dbgReset, patStatus, dbgDim, cmd, dbgReset)
			} else {
				fmt.Fprintf(m.debug, "  %s%-10s%s  pattern %s  %sunchanged%s\n", dbgCyan, name, dbgReset, patStatus, dbgDim, dbgReset)
			}
		}

		afterPattern := cmd
		entStatus := fmt.Sprintf("%s○ off%s", dbgDim, dbgReset)
		if er, ok := m.entropyRedactors[name]; ok && er != nil {
			entStatus = fmt.Sprintf("%s● on%s", dbgGreen, dbgReset)
			cmd = er.Redact(cmd)
		}
		if m.debug != nil {
			if cmd != afterPattern {
				fmt.Fprintf(m.debug, "  %s          %s  entropy %s  %s→ %q%s\n", dbgCyan, dbgReset, entStatus, dbgDim, cmd, dbgReset)
			} else {
				fmt.Fprintf(m.debug, "  %s          %s  entropy %s  %sunchanged%s\n", dbgCyan, dbgReset, entStatus, dbgDim, dbgReset)
			}
		}

		start := time.Now()
		result, err := p.Query(ctx, prompt, cmd)
		elapsed := time.Since(start).Milliseconds()
		if err == nil {
			result = strings.TrimSpace(result)
			if m.cache != nil {
				m.cache.Set(cacheKey, result)
			}
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
