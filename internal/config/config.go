package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type Config struct {
	ProviderOrder    []string                   `yaml:"provider_order"`
	Providers        map[string]*ProviderConfig `yaml:"providers"`
	CommandWhitelist []string                   `yaml:"command_whitelist"`
}

func (c *Config) Get(name string) (*ProviderConfig, error) {
	p, ok := c.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found in config", name)
	}
	if p.APIKey == "" {
		return nil, fmt.Errorf("api_key is not set for provider %q", name)
	}
	if p.Model == "" {
		p.Model = defaultModel(name)
	}
	return p, nil
}

func Load() (Config, error) {
	path := filepath.Join(Dir(), "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config not found at %s — run install.sh first", path)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	if len(cfg.ProviderOrder) == 0 {
		return Config{}, fmt.Errorf("provider_order is empty in config")
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
	case "openai":
		return "gpt-4o-mini"
	default:
		return ""
	}
}
