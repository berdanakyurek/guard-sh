# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Developer Identity

At the start of every session, read `.claude-identity` from the repo root if it exists. It contains:
- `DEVELOPER_ID` — your identity (e.g. `claude-developer-1`); use this when referring to yourself
- `BRANCH_PREFIX` — prefix for all branches you create (e.g. `claude-1/feature-name`)
- `GITHUB_APP_ID` — GitHub App ID used for authenticated pushes
- `GITHUB_APP_CLIENT_ID` — GitHub App Client ID
- `GITHUB_APP_CLIENT_SECRET` — GitHub App Client Secret
- `GITHUB_APP_PRIVATE_KEY_PATH` — path to the App's RSA private key (relative to repo root, gitignored)
- `GITHUB_REPO` — target repo in `owner/repo` format

If `.claude-identity` does not exist, you are working in the base repo as the human owner.

## Pushing with GitHub App Authentication

Never use the human owner's git credentials. Always authenticate as the GitHub App defined in `.claude-identity`.

Steps to push:

```bash
# 1. Source identity
source .claude-identity

# 2. Generate JWT (valid 10 min)
PRIVATE_KEY=$(cat "$GITHUB_APP_PRIVATE_KEY_PATH")
NOW=$(date +%s)
EXP=$((NOW + 600))
HEADER=$(echo -n '{"alg":"RS256","typ":"JWT"}' | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')
PAYLOAD=$(echo -n "{\"iat\":$NOW,\"exp\":$EXP,\"iss\":$GITHUB_APP_ID}" | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')
SIG=$(echo -n "${HEADER}.${PAYLOAD}" | openssl dgst -sha256 -sign "$GITHUB_APP_PRIVATE_KEY_PATH" | openssl base64 -e | tr -d '=' | tr '/+' '_-' | tr -d '\n')
JWT="${HEADER}.${PAYLOAD}.${SIG}"

# 3. Get installation ID
INSTALLATION_ID=$(curl -s -H "Authorization: Bearer $JWT" -H "Accept: application/vnd.github+json" \
  https://api.github.com/app/installations | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['id'])")

# 4. Get installation access token
TOKEN=$(curl -s -X POST -H "Authorization: Bearer $JWT" -H "Accept: application/vnd.github+json" \
  https://api.github.com/app/installations/$INSTALLATION_ID/access_tokens | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

# 5. Push using token
git push https://x-access-token:${TOKEN}@github.com/${GITHUB_REPO}.git HEAD
```

## What This Project Does

**guard-sh** is a shell safety layer. It hooks into bash/zsh/fish to intercept commands before execution, queries an LLM to assess risk, and prompts the user for confirmation if the command is deemed risky. Safe commands are whitelisted or cached to skip the LLM call.

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

## CLI subcommands

```
guard-sh on / off                   enable/disable for current session (handled by shell hook)
guard-sh on --global / off --global   auto-enable/disable in every new terminal
guard-sh status                 show session/global state, config, timeout, work dir, cache, providers, redaction patterns, shell integration, whitelist
guard-sh check "<cmd>"          core check — exit 0 if safe, exit 1 with warning if risky
guard-sh check "<cmd>" --debug  trace whitelist hit, cache hit, provider attempts, LLM response
guard-sh healthcheck            validate API keys, models, latency, shell integration
guard-sh provider add           interactive: pick provider, enter API key (or URL for ollama), pick model
guard-sh provider remove        interactive: remove a configured provider
guard-sh provider order         interactive TUI: reorder providers with arrow keys
guard-sh whitelist              list all whitelisted commands
guard-sh whitelist add <cmd>    add a command (LLM never called for it)
guard-sh whitelist remove <cmd> remove a command from the whitelist
guard-sh redact list             list all global redaction patterns
guard-sh redact pattern on       interactive: enable pattern-based redaction for a provider
guard-sh redact pattern off      interactive: disable pattern-based redaction for a provider
guard-sh redact entropy on       interactive: enable Shannon entropy-based redaction for a provider
guard-sh redact entropy off      interactive: disable Shannon entropy-based redaction for a provider
guard-sh cache on/off           enable/disable response caching
guard-sh cache size <n>         set max cached entries
guard-sh cache clear            delete all cached responses
guard-sh setup                  create config dir, write shell scripts, add shell integration to rc
guard-sh uninstall              remove shell integration from rc files and shell scripts (keeps config)
guard-sh uninstall --purge      also remove the config dir
guard-sh help                   print all commands with descriptions
guard-sh version                print version
```

## Architecture

### Request flow

```
shell command typed
  → shell hook (shell/guard.bash or shell/guard.zsh or shell/guard.fish)
  → guard-sh check <command>
  → main.go: prepend "Working directory: <wd>\nCommand: " if send_working_directory is enabled
  → internal/guard: whitelist check (rawCmd) → LLM dispatch (query)
  → internal/llm/multi.go: per-provider cache check → redaction → provider query, first success wins
  → response printed; shell prompts [Y/n] if not "OK"
```

### Key packages

- **`internal/guard/`** — core logic: whitelist matching, LLM dispatch, command parsing (handles `&&`, `||`, `;`, `|`, subshells, variable assignments)
- **`internal/llm/multi.go`** — per-provider cache check (key: `provider:query`), redaction, provider query in `provider_order`; fails open if all fail
- **`internal/llm/{claude,gemini,openai,deepseek,ollama}/`** — one file per provider, each makes HTTP POST to its API; all implement the same `Provider` interface. Ollama uses `host` instead of `api_key` and hits a local endpoint (`/api/chat`).
- **`internal/redact/`** — regex-based redaction; `Redactor.Redact(s)` replaces pattern matches with `[REDACTED]`. Patterns come from `redact_patterns` in config. Applied in `llm.Multi` per-provider before the LLM call.
- **`internal/cache/`** — LRU cache persisted to `~/.config/guard-sh/cache.json`
- **`internal/config/`** — YAML config loader from `~/.config/guard-sh/config.yaml`

### Shell integration

- **Bash**: `DEBUG` trap with `extdebug` — intercepts before execution
- **Zsh**: custom widget bound to `^M`/`^J` (Enter key); handles multiline (`$CONTEXT == "cont"`) and history expansion
- **Fish**: `bind \n`/`bind \r` mapped to `_guard_execute`; uses `commandline` widget
- All call `guard-sh check` and check exit code: 0 = allow, 1 = block

### System prompt

`prompt.txt` is embedded in the binary and also copied to `~/.config/guard-sh/prompt.txt` at install time. Editing the file on disk takes effect immediately without rebuilding. The LLM is told to reply with `"OK"` for safe commands or a short plain-text warning (no markdown) for risky ones.

Provider-specific prompts can be placed at `~/.config/guard-sh/prompt_PROVIDERNAME.txt` (e.g. `prompt_ollama.txt`). If present, the provider-specific file takes precedence over `prompt.txt` for that provider. Loading happens in `main.go` and is passed to `llm.Multi` as a `map[string]string`.

### Runtime config location

`~/.config/guard-sh/config.yaml` — providers, API keys, whitelist, cache settings, timeout, `send_working_directory`. See `config.default.yaml` (embedded in binary) for all options.

### Working directory context

When `send_working_directory: true` (default), `main.go` reads `os.Getwd()` and builds:
```
Working directory: /path/to/cwd
Command: <cmd>
```
This is passed as the `query` to `g.Check(ctx, rawCmd, query)`. The `rawCmd` (bare command) is used for whitelist matching only.
