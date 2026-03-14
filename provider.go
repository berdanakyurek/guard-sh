package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Berdan/guard-sh/internal/config"
)

func runProvider(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: guard-sh provider [add|remove]\n")
		os.Exit(2)
	}
	switch args[0] {
	case "add":
		runProviderAdd()
	case "remove":
		runProviderRemove()
	default:
		fmt.Fprintf(os.Stderr, "Usage: guard-sh provider [add|remove]\n")
		os.Exit(2)
	}
}

func runProviderAdd() {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: select provider
	fmt.Printf("\n  %sprovider%s\n\n", bold, reset)
	names := []string{"gemini", "claude", "openai", "deepseek"}

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

	// Step 2: API key — validate by fetching models, retry on auth error
	fmt.Printf("\n  %sapi key%s %s(%s)%s\n\n", bold, reset, dim, name, reset)
	var apiKey string
	var models []string
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
	if err := config.AddProvider(name, apiKey, model); err != nil {
		fmt.Fprintf(os.Stderr, "guard-sh: %v\n", err)
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
}

func isAuthError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "HTTP 400") || strings.Contains(s, "HTTP 401") || strings.Contains(s, "HTTP 403")
}

func fetchModels(name, apiKey string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	hc := &http.Client{Timeout: 10 * time.Second}
	switch name {
	case "gemini":
		return fetchGeminiModels(ctx, hc, apiKey)
	case "claude":
		return fetchClaudeModels(ctx, hc, apiKey)
	case "openai":
		return fetchOpenAIModels(ctx, hc, apiKey)
	case "deepseek":
		return fetchDeepSeekModels(ctx, hc, apiKey)
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
