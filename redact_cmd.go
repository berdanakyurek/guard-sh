package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Berdan/guard-sh/internal/config"
)

func runRedact(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact <type> [on|off]\n")
		fmt.Fprintf(os.Stderr, "Types: pattern, entropy\n")
		os.Exit(2)
	}
	switch args[0] {
	case "pattern":
		runRedactType("pattern", args[1:])
	case "entropy":
		runRedactEntropyType(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "guard-sh: unknown redaction type %q\n", args[0])
		fmt.Fprintf(os.Stderr, "Types: pattern, entropy\n")
		os.Exit(2)
	}
}

func runRedactType(redactType string, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact %s [on|off]\n", redactType)
		os.Exit(2)
	}
	switch args[0] {
	case "on":
		runRedactPatternSet(true)
	case "off":
		runRedactPatternSet(false)
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact %s [on|off]\n", redactType)
		os.Exit(2)
	}
}

func runRedactEntropyType(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact entropy [on|off]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "on":
		runRedactEntropySet(true)
	case "off":
		runRedactEntropySet(false)
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact entropy [on|off]\n")
		os.Exit(2)
	}
}

func runRedactEntropySet(enable bool) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.ProviderOrder) == 0 {
		fmt.Fprintln(os.Stderr, "guard-sh: no providers configured")
		os.Exit(1)
	}

	action := "enable"
	if !enable {
		action = "disable"
	}
	fmt.Printf("\n  %sentropy redaction: %s%s\n\n", bold, action, reset)

	for i, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		effective := p.EffectiveEntropyRedaction(cfg.Redaction.ShannonEntropyBased)
		badge := statusBadge(map[bool]string{true: "on", false: "off"}[effective.IsEnabled()])
		fmt.Printf("  %s%d%s  %s%-10s%s  %s\n", dim, i+1, reset, cyan, name, reset, badge)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n  %s>%s ", dim, reset)
	idx, err := pickNumber(reader, len(cfg.ProviderOrder))
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	name := cfg.ProviderOrder[idx-1]
	if err := config.UpdateProviderEntropyRedaction(name, enable); err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	if enable {
		fmt.Printf("\n  %s● entropy redaction enabled for %s%s\n\n", green, name, reset)
	} else {
		fmt.Printf("\n  %s○ entropy redaction disabled for %s%s\n\n", red, name, reset)
	}
}

func runRedactPatternSet(enable bool) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.ProviderOrder) == 0 {
		fmt.Fprintln(os.Stderr, "guard-sh: no providers configured")
		os.Exit(1)
	}

	action := "enable"
	if !enable {
		action = "disable"
	}
	fmt.Printf("\n  %spattern redaction: %s%s\n\n", bold, action, reset)

	for i, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		effective := p.EffectivePatternRedaction(cfg.Redaction.PatternBased)
		badge := statusBadge(map[bool]string{true: "on", false: "off"}[effective.IsEnabled()])
		fmt.Printf("  %s%d%s  %s%-10s%s  %s\n", dim, i+1, reset, cyan, name, reset, badge)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n  %s>%s ", dim, reset)
	idx, err := pickNumber(reader, len(cfg.ProviderOrder))
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	name := cfg.ProviderOrder[idx-1]
	if err := config.UpdateProviderPatternRedaction(name, enable); err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	if enable {
		fmt.Printf("\n  %s● pattern redaction enabled for %s%s\n\n", green, name, reset)
	} else {
		fmt.Printf("\n  %s○ pattern redaction disabled for %s%s\n\n", red, name, reset)
	}
}
