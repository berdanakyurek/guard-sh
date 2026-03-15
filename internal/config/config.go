package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PatternRedactionConfig holds pattern-based redaction settings.
type PatternRedactionConfig struct {
	Enabled  *bool    `yaml:"enabled"`
	Patterns []string `yaml:"patterns"`
}

// IsEnabled returns true if enabled is unset (default on) or explicitly true.
func (c *PatternRedactionConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// GetPatterns returns the configured patterns, or nil if none.
func (c *PatternRedactionConfig) GetPatterns() []string {
	if c == nil {
		return nil
	}
	return c.Patterns
}

// EntropyRedactionConfig holds Shannon entropy-based redaction settings.
type EntropyRedactionConfig struct {
	Enabled   *bool   `yaml:"enabled"`
	Threshold float64 `yaml:"threshold"`
	MinLength int     `yaml:"min_length"`
}

// IsEnabled returns true only if explicitly set to true (default off).
func (c *EntropyRedactionConfig) IsEnabled() bool {
	if c == nil || c.Enabled == nil {
		return false
	}
	return *c.Enabled
}

// GetThreshold returns the entropy threshold, defaulting to 4.5 bits/char.
func (c *EntropyRedactionConfig) GetThreshold() float64 {
	if c == nil || c.Threshold <= 0 {
		return 4.5
	}
	return c.Threshold
}

// GetMinLength returns the minimum token length, defaulting to 20.
func (c *EntropyRedactionConfig) GetMinLength() int {
	if c == nil || c.MinLength <= 0 {
		return 20
	}
	return c.MinLength
}

// RedactionConfig holds all redaction settings. Used both at global level and per-provider.
type RedactionConfig struct {
	PatternBased        *PatternRedactionConfig `yaml:"pattern_based"`
	ShannonEntropyBased *EntropyRedactionConfig `yaml:"shannon_entropy_based"`
}

type ProviderConfig struct {
	APIKey    string           `yaml:"api_key"`
	Model     string           `yaml:"model"`
	Host      string           `yaml:"host"`
	Redaction *RedactionConfig `yaml:"redaction"`
}

// EffectivePatternRedaction returns the effective pattern config for this provider,
// merging per-provider overrides on top of the global config.
func (p *ProviderConfig) EffectivePatternRedaction(global *PatternRedactionConfig) *PatternRedactionConfig {
	if p == nil || p.Redaction == nil || p.Redaction.PatternBased == nil {
		return global
	}
	pb := p.Redaction.PatternBased
	result := &PatternRedactionConfig{}
	if pb.Enabled != nil {
		result.Enabled = pb.Enabled
	} else if global != nil {
		result.Enabled = global.Enabled
	}
	if len(pb.Patterns) > 0 {
		result.Patterns = pb.Patterns
	} else if global != nil {
		result.Patterns = global.Patterns
	}
	return result
}

// EffectiveEntropyRedaction returns the effective entropy config for this provider,
// merging per-provider overrides on top of the global config.
func (p *ProviderConfig) EffectiveEntropyRedaction(global *EntropyRedactionConfig) *EntropyRedactionConfig {
	if p == nil || p.Redaction == nil || p.Redaction.ShannonEntropyBased == nil {
		return global
	}
	eb := p.Redaction.ShannonEntropyBased
	result := &EntropyRedactionConfig{}
	if eb.Enabled != nil {
		result.Enabled = eb.Enabled
	} else if global != nil {
		result.Enabled = global.Enabled
	}
	if eb.Threshold > 0 {
		result.Threshold = eb.Threshold
	} else if global != nil {
		result.Threshold = global.Threshold
	}
	if eb.MinLength > 0 {
		result.MinLength = eb.MinLength
	} else if global != nil {
		result.MinLength = global.MinLength
	}
	return result
}

type Config struct {
	ProviderOrder    []string                   `yaml:"provider_order"`
	Providers        map[string]*ProviderConfig `yaml:"providers"`
	TimeoutSeconds   int                        `yaml:"timeout_seconds"`
	CacheEnabled     *bool                      `yaml:"cache_enabled"`
	CacheMaxSize     int                        `yaml:"cache_max_size"`
	CommandWhitelist []string                   `yaml:"command_whitelist"`
	Redaction        RedactionConfig            `yaml:"redaction"`
}

func (c *Config) Get(name string) (*ProviderConfig, error) {
	p, ok := c.Providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found in config", name)
	}
	if p.APIKey == "" && name != "ollama" {
		return nil, fmt.Errorf("api_key is not set for provider %q", name)
	}
	if p.Model == "" {
		p.Model = DefaultModel(name)
	}
	if p.Host == "" {
		p.Host = DefaultHost(name)
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "provider_order:" || strings.HasPrefix(trimmed, "provider_order:") {
			if trimmed != "provider_order:" {
				lines[i] = "provider_order:"
			}
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "provider_order:" || strings.HasPrefix(trimmed, "provider_order:") {
			// Normalize inline "provider_order: []" to block form
			if trimmed != "provider_order:" {
				lines[i] = "provider_order:"
			}
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "providers:" || strings.HasPrefix(trimmed, "providers:") {
			// Normalize inline "providers: {}" to block form
			if trimmed != "providers:" {
				lines[i] = "providers:"
			}
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
		return "gemini-3.1-flash-lite-preview"
	case "claude":
		return "claude-haiku-4-5-20251001"
	case "openai":
		return "gpt-4o-mini"
	case "deepseek":
		return "deepseek-chat"
	case "ollama":
		return "llama3.2"
	default:
		return ""
	}
}

// UpdateProviderPatternRedaction sets redaction.pattern_based.enabled for a specific provider.
func UpdateProviderPatternRedaction(name string, enabled bool) error {
	return updateProviderRedactionEnabled(name, "pattern_based", enabled)
}

// UpdateProviderEntropyRedaction sets redaction.shannon_entropy_based.enabled for a specific provider.
// If the block does not exist it is created with default threshold and min_length.
func UpdateProviderEntropyRedaction(name string, enabled bool) error {
	return updateProviderRedactionEnabled(name, "shannon_entropy_based", enabled)
}

// updateProviderRedactionEnabled writes the enabled flag for a redaction type under
// providers.NAME.redaction.TYPE, creating intermediate blocks as needed.
//
// Provider block indent: 2sp (name) / 4sp (fields)
// redaction block:       4sp (header) / 6sp (type headers) / 8sp (type fields)
func updateProviderRedactionEnabled(name, redactionType string, enabled bool) error {
	cfgPath := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("config not found at %s", cfgPath)
	}
	lines := strings.Split(string(data), "\n")
	pStart, pEnd := findProviderBlock(lines, name)
	if pStart < 0 {
		return fmt.Errorf("provider %q not found in config", name)
	}

	val := "true"
	if !enabled {
		val = "false"
	}

	// Find "    redaction:" within provider block
	redactionLine := -1
	for i := pStart + 1; i < pEnd; i++ {
		if lines[i] == "    redaction:" {
			redactionLine = i
			break
		}
	}

	if redactionLine >= 0 {
		// End of redaction block: first line without 6-space prefix
		redactionEnd := redactionLine + 1
		for redactionEnd < pEnd && strings.HasPrefix(lines[redactionEnd], "      ") {
			redactionEnd++
		}

		// Find "      TYPE:" within redaction block
		typeLine := -1
		for i := redactionLine + 1; i < redactionEnd; i++ {
			if lines[i] == "      "+redactionType+":" {
				typeLine = i
				break
			}
		}

		if typeLine >= 0 {
			// End of type sub-block: first line without 8-space prefix
			typeEnd := typeLine + 1
			for typeEnd < redactionEnd && strings.HasPrefix(lines[typeEnd], "        ") {
				typeEnd++
			}
			// Find "        enabled:"
			for i := typeLine + 1; i < typeEnd; i++ {
				if strings.HasPrefix(strings.TrimSpace(lines[i]), "enabled:") {
					lines[i] = "        enabled: " + val
					return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0600)
				}
			}
			// Not found — insert after type header
			result := insertLines(lines, typeLine+1, []string{"        enabled: " + val})
			return os.WriteFile(cfgPath, []byte(strings.Join(result, "\n")), 0600)
		}

		// Type block not found — insert at start of redaction block
		result := insertLines(lines, redactionLine+1, newTypeBlock(redactionType, val))
		return os.WriteFile(cfgPath, []byte(strings.Join(result, "\n")), 0600)
	}

	// No redaction block — insert before end of provider block
	newLines := append([]string{"    redaction:"}, newTypeBlock(redactionType, val)...)
	result := insertLines(lines, pEnd, newLines)
	return os.WriteFile(cfgPath, []byte(strings.Join(result, "\n")), 0600)
}

func newTypeBlock(redactionType, enabledVal string) []string {
	block := []string{
		"      " + redactionType + ":",
		"        enabled: " + enabledVal,
	}
	if redactionType == "shannon_entropy_based" {
		block = append(block,
			"        threshold: 4.5",
			"        min_length: 20",
		)
	}
	return block
}

func insertLines(lines []string, at int, insert []string) []string {
	result := make([]string, 0, len(lines)+len(insert))
	result = append(result, lines[:at]...)
	result = append(result, insert...)
	result = append(result, lines[at:]...)
	return result
}

func DefaultHost(provider string) string {
	switch provider {
	case "ollama":
		return "http://localhost:11434/api/chat"
	default:
		return ""
	}
}

// AddOllamaProvider adds or updates the ollama provider entry in config (full URL + model, no api_key).
func AddOllamaProvider(host, model string) error {
	cfgPath := filepath.Join(Dir(), "config.yaml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("config not found at %s", cfgPath)
	}
	lines := strings.Split(string(data), "\n")
	lines = addToProviderOrder(lines, "ollama")
	lines = upsertOllamaBlock(lines, host, model)
	return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0600)
}

func upsertOllamaBlock(lines []string, host, model string) []string {
	start, end := findProviderBlock(lines, "ollama")
	newBlock := []string{
		"  ollama:",
		"    host: " + host,
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
		trimmed := strings.TrimSpace(line)
		if trimmed == "providers:" || strings.HasPrefix(trimmed, "providers:") {
			if trimmed != "providers:" {
				lines[i] = "providers:"
			}
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
