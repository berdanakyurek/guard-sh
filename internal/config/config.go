package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Provider string
	Model    string
	APIKey   string
}

func Load() (Config, error) {
	path := filepath.Join(Dir(), "config")

	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("config not found at %s — run install.sh first", path)
	}
	defer f.Close()

	vals := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	cfg := Config{
		Provider: vals["provider"],
		Model:    vals["model"],
		APIKey:   vals["api_key"],
	}

	if cfg.Provider == "" {
		cfg.Provider = "gemini"
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel(cfg.Provider)
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("api_key is not set in config")
	}

	return cfg, nil
}

func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "guard-sh")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "guard-sh")
}

func defaultModel(provider string) string {
	switch provider {
	case "gemini":
		return "gemini-2.0-flash-lite"
	default:
		return ""
	}
}
