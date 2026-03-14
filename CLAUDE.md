# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

**guard-sh** is a shell safety layer. It hooks into bash/zsh to intercept commands before execution, queries an LLM to assess risk, and prompts the user for confirmation if the command is deemed risky. Safe commands are whitelisted or cached to skip the LLM call.

## Commands

```bash
# Build
go build -o guard-sh .

# Install (builds, deploys to ~/.local/bin, sets up shell integration)
bash install.sh
bash install.sh --without-shell   # skip shell integration

# Test
go test ./...
go test ./internal/guard          # run only guard package tests

# Format / vet
go fmt ./...
go vet ./...
```

There is no Makefile.

## Architecture

### Request flow

```
shell command typed
  → shell hook (shell/guard.bash or shell/guard.zsh)
  → guard-sh check <command>
  → internal/guard: whitelist check → cache check → LLM query
  → internal/llm/multi.go: try each provider in order, first success wins
  → response printed; shell prompts [Y/n] if not "OK"
```

### Key packages

- **`internal/guard/`** — core logic: whitelist matching, cache lookup, LLM dispatch, command parsing (handles `&&`, `||`, `;`, `|`, subshells, variable assignments)
- **`internal/llm/multi.go`** — tries providers in `provider_order` config; fails open (allows command) if all fail
- **`internal/llm/{claude,gemini,openai,deepseek}/`** — one file per provider, each makes HTTP POST to its API; all implement the same `Provider` interface
- **`internal/cache/`** — LRU cache persisted to `~/.config/guard-sh/cache.json`
- **`internal/config/`** — YAML config loader from `~/.config/guard-sh/config.yaml`

### Shell integration

- **Bash**: `DEBUG` trap with `extdebug` — intercepts before execution
- **Zsh**: custom widget bound to `^M`/`^J` (Enter key)
- Both call `guard-sh check` and check exit code: 0 = allow, 1 = block

### System prompt

`prompt.txt` is embedded in the binary and also copied to `~/.config/guard-sh/prompt.txt` at install time. Editing the file on disk takes effect immediately without rebuilding. The LLM is told to reply with `"OK"` for safe commands or a short plain-text warning (no markdown) for risky ones.

### Runtime config location

`~/.config/guard-sh/config.yaml` — providers, API keys, whitelist, cache settings, timeout. See `config.example.yaml` for all options.
