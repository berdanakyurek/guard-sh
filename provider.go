package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Berdan/guard-sh/internal/config"
	"golang.org/x/term"
)

func runProvider(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh provider [add|remove|order]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "add":
		runProviderAdd()
	case "remove":
		runProviderRemove()
	case "order":
		runProviderOrder()
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh provider [add|remove|order]\n")
		os.Exit(2)
	}
}

func runProviderOrder() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.ProviderOrder) < 2 {
		fmt.Fprintln(os.Stderr, "guard-sh: need at least 2 providers to reorder")
		os.Exit(1)
	}

	order := make([]string, len(cfg.ProviderOrder))
	copy(order, cfg.ProviderOrder)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	cursor := 0
	grabbed := false
	saved := false
	linesDrawn := 0

	restore := func() {
		fmt.Printf("\033[?25h") // show cursor
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
	defer func() {
		restore()
		if saved {
			fmt.Printf("\n  %s● order saved%s\n\n", green, reset)
		}
	}()

	render := func() {
		if linesDrawn > 0 {
			fmt.Printf("\033[%dA", linesDrawn)
		}
		linesDrawn = 0

		fmt.Printf("\033[?25l") // hide cursor

		line := func(format string, args ...any) {
			fmt.Printf("\r\033[2K"+format+"\r\n", args...)
			linesDrawn++
		}

		line("  %sprovider order%s", bold, reset)
		line("")
		for i, name := range order {
			p := cfg.Providers[name]
			model := ""
			if p != nil {
				model = p.Model
			}
			if model == "" {
				model = config.DefaultModel(name)
			}
			switch {
			case i == cursor && grabbed:
				line("  %s●%s %s%-10s%s %s%s%s", green, reset, cyan, name, reset, dim, model, reset)
			case i == cursor:
				line("  %s›%s %s%-10s%s %s%s%s", cyan, reset, cyan, name, reset, dim, model, reset)
			default:
				line("    %s%-10s%s %s%s%s", cyan, name, reset, dim, model, reset)
			}
		}
		line("")
		if grabbed {
			line("  %s↑↓ move   enter: place%s", dim, reset)
		} else {
			line("  %s↑↓ navigate   enter: grab   s: save & quit   q: quit without saving%s", dim, reset)
		}
	}

	render()

	buf := make([]byte, 3)
	for {
		n, _ := os.Stdin.Read(buf)
		if n == 0 {
			continue
		}

		switch {
		case n == 3 && buf[0] == '\x1b' && buf[1] == '[' && buf[2] == 'A': // up arrow
			if grabbed {
				if cursor > 0 {
					order[cursor], order[cursor-1] = order[cursor-1], order[cursor]
					cursor--
				}
			} else {
				if cursor > 0 {
					cursor--
				}
			}
		case n == 3 && buf[0] == '\x1b' && buf[1] == '[' && buf[2] == 'B': // down arrow
			if grabbed {
				if cursor < len(order)-1 {
					order[cursor], order[cursor+1] = order[cursor+1], order[cursor]
					cursor++
				}
			} else {
				if cursor < len(order)-1 {
					cursor++
				}
			}
		case n == 1 && (buf[0] == '\r' || buf[0] == '\n'): // enter
			grabbed = !grabbed
		case n == 1 && buf[0] == 's' && !grabbed:
			restore()
			if err := config.UpdateProviderOrder(order); err != nil {
				fmt.Fprintf(os.Stderr, "\r\nguard-sh: %v\r\n", err)
				os.Exit(1)
			}
			saved = true
			return
		case n == 1 && buf[0] == 'q' && !grabbed:
			return
		}
		render()
	}
}

func runProviderAdd() {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: select provider
	fmt.Printf("\n  %sprovider%s\n\n", bold, reset)
	names := []string{"gemini", "claude", "openai", "deepseek", "ollama"}

	cfg, _ := config.Load()
	configured := map[string]bool{}
	for _, n := range cfg.ProviderOrder {
		configured[n] = true
	}

	for i, name := range names {
		suffix := ""
		if configured[name] {
			suffix = dim + "  (configured)" + reset
		}
		fmt.Printf("  %s%d%s  %s%s%s%s\n", dim, i+1, reset, cyan, name, reset, suffix)
	}
	fmt.Printf("\n  %s>%s ", dim, reset)
	choice, err := pickNumber(reader, len(names))
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	name := names[choice-1]

	// Step 2: API key (or host for ollama) — validate by fetching models
	var apiKey, host string
	var models []string

	if name == "ollama" {
		fmt.Printf("\n  %surl%s %s(default: %s)%s\n\n", bold, reset, dim, config.DefaultHost("ollama"), reset)
		for {
			fmt.Printf("  %s> (ENTER for default)  %s", dim, reset)
			line, _ := reader.ReadString('\n')
			host = strings.TrimSpace(line)
			if host == "" {
				host = config.DefaultHost("ollama")
			}
			if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
				host = "http://" + host
			}

			fmt.Printf("  %sconnecting...%s", dim, reset)
			fetched, fetchErr := fetchModels(name, host)
			fmt.Printf("\r%s\r", strings.Repeat(" ", 30))

			if fetchErr != nil {
				fmt.Printf("  %s✗ could not reach ollama (%s), try again%s\n\n", red, fetchErr.Error(), reset)
				continue
			}
			if len(fetched) == 0 {
				fmt.Printf("  %s⚠ no models found, using default list%s\n", dim, reset)
				models = providerModelsFallback[name]
			} else {
				models = fetched
			}
			break
		}
	} else {
		fmt.Printf("\n  %sapi key%s %s(%s)%s\n\n", bold, reset, dim, name, reset)
		for {
			fmt.Printf("  %s>%s ", dim, reset)
			line, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(line)
			if apiKey == "" {
				fmt.Printf("  %s✗ api key cannot be empty%s\n\n", red, reset)
				continue
			}

			fmt.Printf("  %svalidating...%s", dim, reset)
			fetched, fetchErr := fetchModels(name, apiKey)
			fmt.Printf("\r%s\r", strings.Repeat(" ", 30))

			if fetchErr != nil {
				if isAuthError(fetchErr) {
					fmt.Printf("  %s✗ invalid API key, try again%s\n\n", red, reset)
					continue
				}
				// Non-auth error (network, etc.) — fall back to hard-coded list
				fmt.Printf("  %s⚠ could not fetch models (%s), using default list%s\n", red, fetchErr.Error(), reset)
				models = providerModelsFallback[name]
			} else if len(fetched) == 0 {
				fmt.Printf("  %s⚠ no models returned, using default list%s\n", dim, reset)
				models = providerModelsFallback[name]
			} else {
				models = fetched
			}
			break
		}
	}

	// Step 4: select model
	defaultModel := config.DefaultModel(name)
	defaultIdx := 1 // fallback: first in list
	for i, m := range models {
		if m == defaultModel {
			defaultIdx = i + 1
			break
		}
	}
	fmt.Printf("\n  %smodel%s %s(%s)%s\n\n", bold, reset, dim, name, reset)
	for i, m := range models {
		tag := ""
		if i+1 == defaultIdx {
			tag = dim + "  (default)" + reset
		}
		fmt.Printf("  %s%d%s  %s%s%s%s\n", dim, i+1, reset, cyan, m, reset, tag)
	}
	fmt.Printf("\n  %s> (ENTER for default)  %s", dim, reset)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	var model string
	if input == "" {
		model = models[defaultIdx-1]
	} else {
		n, convErr := strconv.Atoi(input)
		if convErr != nil || n < 1 || n > len(models) {
			fmt.Fprintf(os.Stderr, "guard-sh: invalid selection %q\n", input)
			os.Exit(1)
		}
		model = models[n-1]
	}

	// Step 5: save
	var saveErr error
	if name == "ollama" {
		saveErr = config.AddOllamaProvider(host, model)
	} else {
		saveErr = config.AddProvider(name, apiKey, model)
	}
	if saveErr != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", saveErr)
		os.Exit(1)
	}

	fmt.Printf("\n  %s● %s added%s\n\n", green, name, reset)
}

func runProviderRemove() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.ProviderOrder) == 0 {
		fmt.Println("guard-sh: no providers configured")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("\n  %sprovider to remove%s\n\n", bold, reset)
	for i, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		model := ""
		if p != nil {
			model = p.Model
		}
		if model == "" {
			model = config.DefaultModel(name)
		}
		fmt.Printf("  %s%d%s  %s%-10s%s %s%s%s\n", dim, i+1, reset, cyan, name, reset, dim, model, reset)
	}
	fmt.Printf("\n  %s>%s ", dim, reset)
	idx, err := pickNumber(reader, len(cfg.ProviderOrder))
	if err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}
	name := cfg.ProviderOrder[idx-1]

	if err := config.RemoveProvider(name); err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n  %s○ %s removed%s\n\n", red, name, reset)
}

func pickNumber(reader *bufio.Reader, max int) (int, error) {
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > max {
		return 0, fmt.Errorf("invalid selection %q — enter a number between 1 and %d", input, max)
	}
	return n, nil
}

// providerModelsFallback is used when the API fetch fails.
var providerModelsFallback = map[string][]string{
	"gemini":   {"gemini-3.1-flash-lite-preview", "gemini-2.0-flash-lite", "gemini-2.0-flash", "gemini-1.5-flash", "gemini-1.5-pro"},
	"claude":   {"claude-haiku-4-5-20251001", "claude-sonnet-4-6", "claude-opus-4-6"},
	"openai":   {"gpt-4o-mini", "gpt-4o", "gpt-4-turbo", "o1-mini"},
	"deepseek": {"deepseek-chat", "deepseek-reasoner"},
	"ollama":   {"llama3.2", "llama3.1", "mistral", "gemma3", "phi4"},
}

func isAuthError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "HTTP 400") || strings.Contains(s, "HTTP 401") || strings.Contains(s, "HTTP 403")
}

// fetchModels fetches available models for a provider.
// For ollama, the second argument is the host URL instead of an API key.
func fetchModels(name, apiKeyOrHost string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hc := &http.Client{Timeout: 10 * time.Second}
	switch name {
	case "gemini":
		return fetchGeminiModels(ctx, hc, apiKeyOrHost)
	case "claude":
		return fetchClaudeModels(ctx, hc, apiKeyOrHost)
	case "openai":
		return fetchOpenAIModels(ctx, hc, apiKeyOrHost)
	case "deepseek":
		return fetchDeepSeekModels(ctx, hc, apiKeyOrHost)
	case "ollama":
		return fetchOllamaModels(ctx, hc, apiKeyOrHost)
	}
	return nil, fmt.Errorf("unknown provider")
}

func fetchGeminiModels(ctx context.Context, hc *http.Client, apiKey string) ([]string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models?key=" + apiKey
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var result struct {
		Models []struct {
			Name                       string   `json:"name"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var models []string
	for _, m := range result.Models {
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				id := strings.TrimPrefix(m.Name, "models/")
				if !strings.Contains(id, "embedding") && !strings.Contains(id, "aqa") {
					models = append(models, id)
				}
				break
			}
		}
	}
	return models, nil
}

func fetchClaudeModels(ctx context.Context, hc *http.Client, apiKey string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}

func fetchOpenAIModels(ctx context.Context, hc *http.Client, apiKey string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var models []string
	for _, m := range result.Data {
		id := m.ID
		if strings.HasPrefix(id, "gpt-") || strings.HasPrefix(id, "o1") || strings.HasPrefix(id, "o3") || strings.HasPrefix(id, "o4") || strings.HasPrefix(id, "chatgpt-") {
			models = append(models, id)
		}
	}
	sort.Strings(models)
	return models, nil
}

func ollamaBase(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Scheme + "://" + u.Host
}

func fetchOllamaModels(ctx context.Context, hc *http.Client, rawURL string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBase(rawURL)+"/api/tags", nil)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Models))
	for _, m := range result.Models {
		// Strip ":latest" suffix for cleaner display
		name := strings.TrimSuffix(m.Name, ":latest")
		models = append(models, name)
	}
	return models, nil
}

func fetchDeepSeekModels(ctx context.Context, hc *http.Client, apiKey string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, m.ID)
	}
	return models, nil
}
