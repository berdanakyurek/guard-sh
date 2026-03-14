package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

type Config struct {
	ProviderOrder    []string                   `yaml:"provider_order"`
	Providers        map[string]*ProviderConfig `yaml:"providers"`
	TimeoutSeconds   int                        `yaml:"timeout_seconds"`
	CacheEnabled     *bool                      `yaml:"cache_enabled"`
	CacheMaxSize     int                        `yaml:"cache_max_size"`
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
		p.Model = DefaultModel(name)
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

// UpdateCacheEnabled rewrites only the cache_enabled line in the config file.
func UpdateCacheEnabled(enabled bool) error {
	path := filepath.Join(Dir(), "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}

	val := "true"
	if !enabled {
		val = "false"
	}
	newLine := "cache_enabled: " + val

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cache_enabled:") {
			lines[i] = newLine
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
		}
	}

	// Not found — append before command_whitelist or at end
	for i, line := range lines {
		if strings.TrimSpace(line) == "command_whitelist:" {
			lines = append(lines[:i], append([]string{newLine, ""}, lines[i:]...)...)
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
		}
	}
	lines = append(lines, newLine)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

// UpdateCacheMaxSize rewrites only the cache_max_size line in the config file.
func UpdateCacheMaxSize(size int) error {
	path := filepath.Join(Dir(), "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}

	newLine := "cache_max_size: " + strconv.Itoa(size)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "cache_max_size:") {
			lines[i] = newLine
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
		}
	}

	// Not found — append before command_whitelist or at end
	for i, line := range lines {
		if strings.TrimSpace(line) == "command_whitelist:" {
			lines = append(lines[:i], append([]string{newLine, ""}, lines[i:]...)...)
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
		}
	}
	lines = append(lines, newLine)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

// UpdateWhitelist rewrites only the command_whitelist section of the config
// file, preserving all other content (comments, provider config, etc.).
func UpdateWhitelist(whitelist []string) error {
	path := filepath.Join(Dir(), "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}

	lines := strings.Split(string(data), "\n")

	// Find the command_whitelist: line
	wlIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "command_whitelist:" {
			wlIdx = i
			break
		}
	}

	// Build replacement list lines
	var newItems []string
	for _, cmd := range whitelist {
		newItems = append(newItems, "  - "+cmd)
	}

	if wlIdx == -1 {
		// Section missing — append it
		lines = append(lines, "command_whitelist:")
		lines = append(lines, newItems...)
	} else {
		// Remove existing list items right after the section header
		end := wlIdx + 1
		for end < len(lines) && strings.HasPrefix(lines[end], "  - ") {
			end++
		}
		replaced := make([]string, 0, len(lines)-(end-wlIdx-1)+len(newItems))
		replaced = append(replaced, lines[:wlIdx+1]...)
		replaced = append(replaced, newItems...)
		replaced = append(replaced, lines[end:]...)
		lines = replaced
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

func UpdateProviderOrder(order []string) error {
	path := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}
	lines := strings.Split(string(data), "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "provider_order:" {
			start = i
			break
		}
	}
	if start == -1 {
		return fmt.Errorf("provider_order not found in config")
	}

	end := start + 1
	for end < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[end]), "- ") {
		end++
	}

	newItems := make([]string, len(order))
	for i, name := range order {
		newItems[i] = "  - " + name
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:start+1]...)
	result = append(result, newItems...)
	result = append(result, lines[end:]...)
	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0600)
}

func AddProvider(name, apiKey, model string) error {
	path := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}
	lines := strings.Split(string(data), "\n")
	lines = addToProviderOrder(lines, name)
	lines = upsertProviderBlock(lines, name, apiKey, model)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

func RemoveProvider(name string) error {
	path := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config not found at %s", path)
	}
	lines := strings.Split(string(data), "\n")
	lines = removeFromProviderOrder(lines, name)
	lines = deleteProviderBlock(lines, name)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)
}

func addToProviderOrder(lines []string, name string) []string {
	for _, line := range lines {
		if strings.TrimSpace(line) == "- "+name {
			return lines // already present
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "provider_order:" {
			j := i + 1
			for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
				j++
			}
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:j]...)
			result = append(result, "  - "+name)
			result = append(result, lines[j:]...)
			return result
		}
	}
	return lines
}

func removeFromProviderOrder(lines []string, name string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "- "+name {
			continue
		}
		result = append(result, line)
	}
	return result
}

// findProviderBlock returns (start, end) where start is the "  name:" line index
// and end is the first line after the block. Returns -1,-1 if not found.
func findProviderBlock(lines []string, name string) (int, int) {
	header := "  " + name + ":"
	for i, line := range lines {
		if line == header {
			j := i + 1
			for j < len(lines) && strings.HasPrefix(lines[j], "    ") {
				j++
			}
			return i, j
		}
	}
	return -1, -1
}

func upsertProviderBlock(lines []string, name, apiKey, model string) []string {
	start, end := findProviderBlock(lines, name)
	newBlock := []string{
		"  " + name + ":",
		"    api_key: " + apiKey,
		"    model: " + model,
	}
	if start >= 0 {
		result := make([]string, 0, len(lines))
		result = append(result, lines[:start]...)
		result = append(result, newBlock...)
		result = append(result, lines[end:]...)
		return result
	}
	// Insert at end of providers section
	for i, line := range lines {
		if strings.TrimSpace(line) == "providers:" {
			j := i + 1
			for j < len(lines) {
				l := lines[j]
				if l != "" && !strings.HasPrefix(l, " ") {
					break
				}
				j++
			}
			for j > i+1 && strings.TrimSpace(lines[j-1]) == "" {
				j--
			}
			insert := append([]string{""}, newBlock...)
			result := make([]string, 0, len(lines)+len(insert))
			result = append(result, lines[:j]...)
			result = append(result, insert...)
			result = append(result, lines[j:]...)
			return result
		}
	}
	return lines
}

func deleteProviderBlock(lines []string, name string) []string {
	start, end := findProviderBlock(lines, name)
	if start < 0 {
		return lines
	}
	// Also consume trailing blank line
	if end < len(lines) && strings.TrimSpace(lines[end]) == "" {
		end++
	}
	result := make([]string, 0, len(lines))
	result = append(result, lines[:start]...)
	result = append(result, lines[end:]...)
	return result
}

func DefaultModel(provider string) string {
	switch provider {
	case "gemini":
		return "gemini-broken-model"
	case "claude":
		return "claude-haiku-4-5-20251001"
	case "openai":
		return "gpt-4o-mini"
	case "deepseek":
		return "deepseek-chat"
	default:
		return ""
	}
}
