package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/Berdan/guard-sh/internal/config"
	"github.com/Berdan/guard-sh/internal/guard"
	"github.com/Berdan/guard-sh/internal/llm/gemini"
)

//go:embed prompt.txt
var defaultPrompt string

func main() {
	if len(os.Args) < 3 || os.Args[1] != "check" {
		fmt.Fprintln(os.Stderr, "Usage: guard-sh check <command>")
		os.Exit(2)
	}

	cmd := os.Args[2]
	if cmd == "" {
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(0) // fail open
	}

	var provider guard.Provider
	switch cfg.Provider {
	case "gemini":
		provider = gemini.New(cfg.APIKey, cfg.Model)
	default:
		fmt.Fprintf(os.Stderr, "guard-sh: unknown provider %q\n", cfg.Provider)
		os.Exit(0) // fail open
	}

	g := guard.New(provider, defaultPrompt, config.Dir())
	ctx := context.Background()

	safe, warning := g.Check(ctx, cmd)
	if safe {
		os.Exit(0)
	}
	fmt.Println(warning + " [Y/n]")
	os.Exit(1)
}
