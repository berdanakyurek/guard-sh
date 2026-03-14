package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Berdan/guard-sh/internal/config"
	"github.com/Berdan/guard-sh/internal/guard"
	"github.com/Berdan/guard-sh/internal/llm"
	"github.com/Berdan/guard-sh/internal/llm/claude"
	"github.com/Berdan/guard-sh/internal/llm/deepseek"
	"github.com/Berdan/guard-sh/internal/llm/gemini"
	"github.com/Berdan/guard-sh/internal/llm/openai"
)

//go:embed prompt.txt
var defaultPrompt string

//go:embed config.default.yaml
var defaultConfig string

//go:embed shell/guard.bash
var shellBash string

//go:embed shell/guard.zsh
var shellZsh string

var version = "dev"

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)

func statusBadge(val string) string {
	if val == "on" {
		return green + "● on" + reset
	}
	return red + "○ off" + reset
}

func label(s string) string {
	return dim + s + reset
}

func runStatus(args []string) {
	session := "off"
	global := "off"
	for _, arg := range args {
		if v, ok := strings.CutPrefix(arg, "--session="); ok {
			session = v
		} else if v, ok := strings.CutPrefix(arg, "--global="); ok {
			global = v
		}
	}

	cfg, err := config.Load()
	configPath := config.Dir() + "/config.yaml"

	fmt.Printf("  %sguard-sh%s\n\n", bold+cyan, reset)
	fmt.Printf("  %s  %s\n", label("session "), statusBadge(session))
	fmt.Printf("  %s  %s\n", label("global  "), statusBadge(global))
	fmt.Printf("  %s  %s%s%s\n", label("config  "), dim, configPath, reset)

	if err != nil {
		fmt.Printf("\n  %s%s%s\n\n", red, err.Error(), reset)
		return
	}

	promptPath := config.Dir() + "/prompt.txt"
	fmt.Printf("  %s  %s%s%s\n", label("prompt  "), dim, promptPath, reset)

	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	fmt.Printf("  %s  %s%ds%s\n", label("timeout "), dim, timeout, reset)

	cacheEnabled := cfg.CacheEnabled == nil || *cfg.CacheEnabled
	cacheMaxSize := cfg.CacheMaxSize
	if cacheMaxSize <= 0 {
		cacheMaxSize = 1000
	}
	fmt.Printf("  %s  %s\n", label("cache   "), statusBadge(map[bool]string{true: "on", false: "off"}[cacheEnabled]))
	if cacheEnabled {
		fmt.Printf("  %s  %s%d max entries%s\n", label("        "), dim, cacheMaxSize, reset)
		cachePath := config.Dir() + "/cache.json"
		if info, statErr := os.Stat(cachePath); statErr == nil {
			var entries map[string]json.RawMessage
			if data, readErr := os.ReadFile(cachePath); readErr == nil {
				_ = json.Unmarshal(data, &entries)
			}
			fmt.Printf("  %s  %s%d entries, %s%s\n", label("        "), dim, len(entries), formatBytes(info.Size()), reset)
		}
	}

	fmt.Printf("\n  %sproviders%s\n", bold, reset)
	for i, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		model := ""
		if p != nil {
			model = p.Model
		}
		if model == "" {
			model = config.DefaultModel(name)
		}
		fmt.Printf("  %s%d%s  %s%-10s%s%s%s\n",
			dim, i+1, reset,
			cyan, name, reset,
			dim, model+reset,
		)
	}

	if len(cfg.CommandWhitelist) > 0 {
		fmt.Printf("\n  %swhitelist%s\n", bold, reset)
		const max = 10
		shown := cfg.CommandWhitelist
		if len(shown) > max {
			shown = shown[:max]
		}
		for i, cmd := range shown {
			fmt.Printf("  %s%d%s  %s%s%s\n", dim, i+1, reset, dim, cmd, reset)
		}
		if remaining := len(cfg.CommandWhitelist) - max; remaining > 0 {
			fmt.Printf("  %s+%d more (to see all, run \"guard-sh whitelist\")%s\n", dim, remaining, reset)
		}
	}

	fmt.Println()
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func runCache(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh cache [on|off|size <n>]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "on":
		if err := config.UpdateCacheEnabled(true); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("guard-sh: cache enabled")
	case "off":
		if err := config.UpdateCacheEnabled(false); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("guard-sh: cache disabled")
	case "size":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: guard-sh cache size <n>\n")
			os.Exit(2)
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "guard-sh: size must be a positive integer\n")
			os.Exit(1)
		}
		if err := config.UpdateCacheMaxSize(n); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("guard-sh: cache max size set to %d\n", n)
	case "clear":
		path := config.Dir() + "/cache.json"
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("guard-sh: cache cleared")
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh cache [on|off|clear|size <n>]\n")
		os.Exit(2)
	}
}

func runWhitelist(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 0 {
		for _, cmd := range cfg.CommandWhitelist {
			fmt.Println(cmd)
		}
		return
	}

	subcmd := args[0]
	if subcmd != "add" && subcmd != "remove" || len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh whitelist [add|remove] <command>\n")
		os.Exit(2)
	}
	target := args[1]

	switch subcmd {
	case "add":
		for _, cmd := range cfg.CommandWhitelist {
			if cmd == target {
				fmt.Fprintf(os.Stderr, "guard-sh: %q is already in the whitelist\n", target)
				os.Exit(1)
			}
		}
		cfg.CommandWhitelist = append(cfg.CommandWhitelist, target)
		if err := config.UpdateWhitelist(cfg.CommandWhitelist); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("guard-sh: %q added to whitelist\n", target)

	case "remove":
		found := false
		updated := cfg.CommandWhitelist[:0]
		for _, cmd := range cfg.CommandWhitelist {
			if cmd == target {
				found = true
			} else {
				updated = append(updated, cmd)
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "guard-sh: %q is not in the whitelist\n", target)
			os.Exit(1)
		}
		if err := config.UpdateWhitelist(updated); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("guard-sh: %q removed from whitelist\n", target)
	}
}

func main() {
	if len(os.Args) >= 2 && (os.Args[1] == "on" || os.Args[1] == "off") {
		fmt.Fprintf(os.Stderr, "guard-sh: shell integration not loaded. Run: source /path/to/shell/guard.bash\n")
		os.Exit(2)
	}

	if len(os.Args) >= 2 && os.Args[1] == "status" {
		runStatus(os.Args[2:])
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "whitelist" {
		runWhitelist(os.Args[2:])
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "cache" {
		runCache(os.Args[2:])
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		runHealthcheck()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "provider" {
		runProvider(os.Args[2:])
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "setup" {
		runSetup()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "help" {
		runHelp()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "version" {
		fmt.Println(version)
		return
	}

	if len(os.Args) < 3 || os.Args[1] != "check" {
		fmt.Fprintln(os.Stderr, "Unknown Command: Run \"guard-sh help\" to see available commands.")
		os.Exit(2)
	}

	cmd := os.Args[2]
	if cmd == "" {
		os.Exit(0)
	}

	var debugOut io.Writer
	for _, arg := range os.Args[3:] {
		if arg == "--debug" {
			debugOut = os.Stderr
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(0) // fail open
	}

	if len(cfg.ProviderOrder) == 0 {
		fmt.Fprintln(os.Stderr, "guard-sh: no providers configured — run \"guard-sh provider add\" to add one")
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
		case "claude":
			provider = claude.New(p.APIKey, p.Model)
		case "deepseek":
			provider = deepseek.New(p.APIKey, p.Model)
		case "openai":
			provider = openai.New(p.APIKey, p.Model)
		default:
			fmt.Fprintf(os.Stderr, "guard-sh: unknown provider %q\n", name)
			os.Exit(0) // fail open
		}

		names = append(names, name)
		providers = append(providers, provider)
	}

	cacheMaxSize := 0 // disabled
	if cfg.CacheEnabled == nil || *cfg.CacheEnabled {
		cacheMaxSize = cfg.CacheMaxSize
	}
	g := guard.New(llm.NewMulti(names, providers, debugOut), defaultPrompt, config.Dir(), cfg.CommandWhitelist, cacheMaxSize, debugOut)

	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	safe, warning := g.Check(ctx, cmd)
	if safe {
		os.Exit(0)
	}
	fmt.Println(warning + " [Y/n]")
	os.Exit(1)
}
