package main

import "fmt"

func runHelp() {
	c := func(s string) string { return cyan + s + reset }
	b := func(s string) string { return bold + s + reset }
	d := func(s string) string { return dim + s + reset }

	fmt.Printf("  %s\n\n", b(c("guard-sh")))
	fmt.Printf("  %s\n\n", d("A shell safety layer powered by LLMs. Intercepts commands before execution and prompts for confirmation on risky ones."))

	// Session control
	fmt.Printf("  %s\n", b("session"))
	fmt.Printf("  %s %s\n", c("guard-sh on"), d("enable for the current terminal session"))
	fmt.Printf("  %s %s\n\n", c("guard-sh off"), d("disable for the current terminal session"))

	// Global control
	fmt.Printf("  %s\n", b("global"))
	fmt.Printf("  %s %s\n", c("guard-sh on --global"), d("auto-enable in every new terminal"))
	fmt.Printf("  %s %s\n\n", c("guard-sh off --global"), d("stop auto-enabling in new terminals"))

	// Status
	fmt.Printf("  %s\n", b("status"))
	fmt.Printf("  %s %s\n\n", c("guard-sh status"), d("show session/global state, prompt, timeout, cache stats, providers, whitelist"))

	// Debug
	fmt.Printf("  %s\n", b("debug"))
	fmt.Printf("  %s %s\n\n", c("guard-sh check \"<command>\" --debug"), d("trace whitelist, cache, provider attempts, and LLM response"))

	// Healthcheck
	fmt.Printf("  %s\n", b("healthcheck"))
	fmt.Printf("  %s %s\n\n", c("guard-sh healthcheck"), d("validate providers: API keys, models, latency, shell integration"))

	// Whitelist
	fmt.Printf("  %s\n", b("whitelist"))
	fmt.Printf("  %s %s\n", c("guard-sh whitelist"), d("list all whitelisted commands"))
	fmt.Printf("  %s %s\n", c("guard-sh whitelist add <cmd>"), d("add a command — LLM is never called for it"))
	fmt.Printf("  %s %s\n\n", c("guard-sh whitelist remove <cmd>"), d("remove a command from the whitelist"))

	// Provider
	fmt.Printf("  %s\n", b("provider"))
	fmt.Printf("  %s %s\n", c("guard-sh provider add"), d("interactively add a provider (select model, enter API key)"))
	fmt.Printf("  %s %s\n", c("guard-sh provider remove"), d("interactively remove a configured provider"))
	fmt.Printf("  %s %s\n\n", c("guard-sh provider order"), d("interactively reorder providers with arrow keys"))

	// Cache
	fmt.Printf("  %s\n", b("cache"))
	fmt.Printf("  %s %s\n", c("guard-sh cache on"), d("enable response caching"))
	fmt.Printf("  %s %s\n", c("guard-sh cache off"), d("disable response caching"))
	fmt.Printf("  %s %s\n", c("guard-sh cache size <n>"), d("set max number of cached responses"))
	fmt.Printf("  %s %s\n\n", c("guard-sh cache clear"), d("delete all cached responses"))

	// Config
	fmt.Printf("  %s\n", b("config"))
	fmt.Printf("  %s\n\n", d("~/.config/guard-sh/config.yaml — providers, API keys, whitelist, cache, timeout"))

	// Setup
	fmt.Printf("  %s\n", b("setup"))
	fmt.Printf("  %s %s\n\n", c("guard-sh setup"), d("create config dir, write shell scripts, add shell integration"))

	// Version
	fmt.Printf("  %s\n", b("version"))
	fmt.Printf("  %s %s\n\n", c("guard-sh version"), d("print version"))
}
