package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Berdan/guard-sh/internal/config"
)

func runHealthcheck() {
	fmt.Printf("  %sguard-sh healthcheck%s\n\n", bold+cyan, reset)

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("  %s%s%s\n\n", red, err.Error(), reset)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hc := &http.Client{Timeout: 10 * time.Second}
	allOK := true

	for _, name := range cfg.ProviderOrder {
		p := cfg.Providers[name]
		apiKey := ""
		model := ""
		if p != nil {
			apiKey = p.APIKey
			model = p.Model
		}
		if model == "" {
			model = config.DefaultModel(name)
		}

		nameCol := fmt.Sprintf("%-10s", name)
		modelCol := fmt.Sprintf("%-34s", model)

		if apiKey == "" {
			fmt.Printf("  %s%s%s  %s%s%s  %s✗ api_key not set%s\n",
				cyan, nameCol, reset, dim, modelCol, reset, red, reset)
			allOK = false
			continue
		}

		var checkErr error
		switch name {
		case "gemini":
			checkErr = healthGemini(ctx, hc, apiKey, model)
		case "claude":
			checkErr = healthClaude(ctx, hc, apiKey, model)
		case "openai":
			checkErr = healthOpenAI(ctx, hc, apiKey, model)
		case "deepseek":
			checkErr = healthDeepSeek(ctx, hc, apiKey, model)
		default:
			checkErr = fmt.Errorf("unknown provider")
		}

		if checkErr != nil {
			fmt.Printf("  %s%s%s  %s%s%s  %s✗ %s%s\n",
				cyan, nameCol, reset, dim, modelCol, reset, red, checkErr.Error(), reset)
			allOK = false
		} else {
			fmt.Printf("  %s%s%s  %s%s%s  %s● ok%s\n",
				cyan, nameCol, reset, dim, modelCol, reset, green, reset)
		}
	}

	fmt.Println()
	if allOK {
		fmt.Printf("  %sall providers healthy%s\n\n", green, reset)
	}
}

// healthGemini calls the Gemini model info endpoint (no tokens).
func healthGemini(ctx context.Context, hc *http.Client, apiKey, model string) error {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s?key=%s", model, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return apiError(resp)
}

// healthClaude calls the Anthropic model info endpoint (no tokens).
func healthClaude(ctx context.Context, hc *http.Client, apiKey, model string) error {
	url := fmt.Sprintf("https://api.anthropic.com/v1/models/%s", model)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return apiError(resp)
}

// healthOpenAI calls the OpenAI model info endpoint (no tokens).
func healthOpenAI(ctx context.Context, hc *http.Client, apiKey, model string) error {
	url := fmt.Sprintf("https://api.openai.com/v1/models/%s", model)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return apiError(resp)
}

// healthDeepSeek lists DeepSeek models and checks the configured model is present (no tokens).
func healthDeepSeek(ctx context.Context, hc *http.Client, apiKey, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("unexpected response")
	}
	for _, m := range result.Data {
		if m.ID == model {
			return nil
		}
	}
	return fmt.Errorf("model %q not found", model)
}

// apiError extracts a human-readable error from a non-200 API response.
func apiError(resp *http.Response) error {
	b, _ := io.ReadAll(resp.Body)
	// Try common {"error":{"message":"..."}} shape (OpenAI, DeepSeek, Gemini, Anthropic).
	var wrapper struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(b, &wrapper) == nil && wrapper.Error.Message != "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, wrapper.Error.Message)
	}
	return fmt.Errorf("HTTP %d", resp.StatusCode)
}
