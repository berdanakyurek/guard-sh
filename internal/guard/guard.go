package guard

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Berdan/guard-sh/internal/cache"
)

const (
	dbgReset = "\033[0m"
	dbgBold  = "\033[1m"
	dbgDim   = "\033[2m"
	dbgGreen = "\033[32m"
	dbgCyan  = "\033[36m"
	dbgRed   = "\033[31m"
)

// Provider is the interface that LLM backends must implement.
type Provider interface {
	Query(ctx context.Context, systemPrompt, command string) (string, error)
}

// Guard checks shell commands against an LLM provider.
type Guard struct {
	provider  Provider
	prompt    string
	whitelist map[string]bool
	cache     *cache.Cache
	debug     io.Writer
}

// New creates a Guard. If a custom prompt.txt exists in configDir it takes
// precedence over the compiled-in defaultPrompt.
func New(provider Provider, defaultPrompt, configDir string, whitelist []string, cacheMaxSize int, debug io.Writer) *Guard {
	prompt := defaultPrompt
	if data, err := os.ReadFile(filepath.Join(configDir, "prompt.txt")); err == nil {
		prompt = string(data)
	}
	wl := make(map[string]bool, len(whitelist))
	for _, cmd := range whitelist {
		wl[strings.TrimSpace(cmd)] = true
	}
	return &Guard{provider: provider, prompt: prompt, whitelist: wl, cache: cache.Load(configDir, cacheMaxSize), debug: debug}
}

func (g *Guard) dbg(format string, args ...any) {
	if g.debug != nil {
		fmt.Fprintf(g.debug, format, args...)
	}
}

// Check queries the LLM. Returns (safe=true, warning="") if OK,
// or (safe=false, warning="...") if the command needs confirmation.
// On any error it fails open.
func (g *Guard) Check(ctx context.Context, cmd string) (safe bool, warning string) {
	g.dbg("\n  %scommand%s   %s%q%s\n\n", dbgDim, dbgReset, dbgCyan, cmd, dbgReset)

	if len(g.whitelist) > 0 {
		bases := extractBaseCommands(cmd)
		if len(bases) > 0 && g.allWhitelisted(bases) {
			g.dbg("  %swhitelist%s %s● hit%s %s(%s)%s\n\n", dbgDim, dbgReset, dbgGreen, dbgReset, dbgDim, strings.Join(bases, ", "), dbgReset)
			return true, ""
		}
		g.dbg("  %swhitelist%s %s○ miss%s\n", dbgDim, dbgReset, dbgDim, dbgReset)
	} else {
		g.dbg("  %swhitelist%s %sempty%s\n", dbgDim, dbgReset, dbgDim, dbgReset)
	}

	if g.cache != nil {
		if cached, ok := g.cache.Get(cmd); ok {
			g.dbg("  %scache%s     %s● hit%s %s→ %q%s\n\n", dbgDim, dbgReset, dbgGreen, dbgReset, dbgDim, cached, dbgReset)
			if cached == "OK" || cached == "" {
				return true, ""
			}
			return false, cached
		}
		g.dbg("  %scache%s     %s○ miss%s\n", dbgDim, dbgReset, dbgDim, dbgReset)
	} else {
		g.dbg("  %scache%s     %sdisabled%s\n", dbgDim, dbgReset, dbgDim, dbgReset)
	}

	g.dbg("\n  %sproviders%s\n", dbgBold, dbgReset)

	response, err := g.provider.Query(ctx, g.prompt, cmd)
	if err != nil {
		g.dbg("\n  %s✗ all providers failed%s\n\n", dbgRed, dbgReset)
		return false, "Could not reach any provider. Proceed anyway?"
	}

	response = strings.TrimSpace(response)
	if g.cache != nil {
		g.cache.Set(cmd, response)
	}

	if response == "OK" || response == "" {
		g.dbg("\n  %sresult%s    %s● safe%s\n\n", dbgDim, dbgReset, dbgGreen, dbgReset)
		return true, ""
	}

	g.dbg("\n  %sresult%s    %s%q%s\n\n", dbgDim, dbgReset, dbgRed, response, dbgReset)
	return false, response
}

func (g *Guard) allWhitelisted(bases []string) bool {
	for _, b := range bases {
		if b == "" {
			continue
		}
		if !g.whitelist[b] && !g.whitelist[filepath.Base(b)] {
			return false
		}
	}
	return true
}

// extractBaseCommands splits a shell command string on unquoted operators
// (&&, ||, ;, |, newline) and returns the base command name of each part.
func extractBaseCommands(cmd string) []string {
	var commands []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	i := 0
	for i < len(cmd) {
		c := cmd[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteByte(c)
		case !inSingle && !inDouble:
			if i+1 < len(cmd) && (cmd[i:i+2] == "&&" || cmd[i:i+2] == "||") {
				commands = append(commands, baseCommand(current.String()))
				current.Reset()
				i += 2
				continue
			}
			if c == ';' || c == '|' || c == '\n' {
				commands = append(commands, baseCommand(current.String()))
				current.Reset()
			} else {
				current.WriteByte(c)
			}
		default:
			current.WriteByte(c)
		}
		i++
	}
	if s := strings.TrimSpace(current.String()); s != "" {
		commands = append(commands, baseCommand(s))
	}
	return commands
}

// baseCommand extracts the command name from a fragment, skipping
// leading variable assignments (FOO=bar) and subshell characters.
func baseCommand(fragment string) string {
	fields := strings.Fields(strings.TrimSpace(fragment))
	for _, f := range fields {
		if strings.Contains(f, "=") {
			continue // skip VAR=value assignments
		}
		f = strings.Trim(f, "()") // strip surrounding ( ) from subshells
		if f == "" {
			continue
		}
		return f
	}
	return ""
}
