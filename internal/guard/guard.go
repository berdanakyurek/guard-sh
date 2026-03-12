package guard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// Provider is the interface that LLM backends must implement.
type Provider interface {
	Query(ctx context.Context, systemPrompt, command string) (string, error)
}

// Guard checks shell commands against an LLM provider.
type Guard struct {
	provider Provider
	prompt   string
}

// New creates a Guard. If a custom prompt.txt exists in configDir it takes
// precedence over the compiled-in defaultPrompt.
func New(provider Provider, defaultPrompt, configDir string) *Guard {
	prompt := defaultPrompt
	if data, err := os.ReadFile(filepath.Join(configDir, "prompt.txt")); err == nil {
		prompt = string(data)
	}
	return &Guard{provider: provider, prompt: prompt}
}

// Check queries the LLM. Returns (safe=true, warning="") if OK,
// or (safe=false, warning="...") if the command needs confirmation.
// On any error it fails open.
func (g *Guard) Check(ctx context.Context, cmd string) (safe bool, warning string) {
	response, err := g.provider.Query(ctx, g.prompt, cmd)
	if err != nil {
		return false, "Could not reach any provider. Proceed anyway?"
	}

	response = strings.TrimSpace(response)
	if response == "OK" || response == "" {
		return true, ""
	}

	return false, response
}
