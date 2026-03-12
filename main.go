package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/Berdan/guard-sh/internal/config"
	"github.com/Berdan/guard-sh/internal/guard"
	"github.com/Berdan/guard-sh/internal/llm"
	"github.com/Berdan/guard-sh/internal/llm/gemini"
)

//go:embed prompt.txt
var defaultPrompt string

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "on" || os.Args[1] == "off") {
		fmt.Fprintf(os.Stderr, "guard-sh: shell integration not loaded. Run: source /path/to/shell/guard.bash\n")
		os.Exit(2)
	}

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

	var names []string
	var providers []guard.Provider

	for _, name := range cfg.ProviderOrder {
		p, err := cfg.Get(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(0) // fail open
		}

		var provider guard.Provider
		switch name {
		case "gemini":
			provider = gemini.New(p.APIKey, p.Model)
		default:
			fmt.Fprintf(os.Stderr, "guard-sh: unknown provider %q\n", name)
			os.Exit(0) // fail open
		}

		names = append(names, name)
		providers = append(providers, provider)
	}

	g := guard.New(llm.NewMulti(names, providers), defaultPrompt, config.Dir(), cfg.CommandWhitelist)
	ctx := context.Background()

	safe, warning := g.Check(ctx, cmd)
	if safe {
		os.Exit(0)
	}
	fmt.Println(warning + " [Y/n]")
	os.Exit(1)
}
