package llm

import (
	"context"
	"errors"

	"github.com/Berdan/guard-sh/internal/guard"
)

// Multi tries each provider in order, falling back to the next on error.
type Multi struct {
	providers []guard.Provider
	names     []string
}

func NewMulti(names []string, providers []guard.Provider) *Multi {
	return &Multi{names: names, providers: providers}
}

func (m *Multi) Query(ctx context.Context, systemPrompt, command string) (string, error) {
	for _, p := range m.providers {
		result, err := p.Query(ctx, systemPrompt, command)
		if err == nil {
			return result, nil
		}
	}
	return "", errors.New("all providers failed")
}
