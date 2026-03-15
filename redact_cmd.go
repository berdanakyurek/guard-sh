package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Berdan/guard-sh/internal/config"
)

func runRedact(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact [on|off]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "on":
		runRedactSet(true)
	case "off":
		runRedactSet(false)
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh redact [on|off]\n")
		os.Exit(2)
	}
}

func runRedactSet(enable bool) {
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
	fmt.Printf("\n  %sredaction: %s%s\n\n", bold, action, reset)

	for i, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		badge := statusBadge(map[bool]string{true: "on", false: "off"}[p.RedactionEnabled()])
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
	if err := config.UpdateProviderRedaction(name, enable); err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	if enable {
		fmt.Printf("\n  %s● redaction enabled for %s%s\n\n", green, name, reset)
	} else {
		fmt.Printf("\n  %s○ redaction disabled for %s%s\n\n", red, name, reset)
	}
}
