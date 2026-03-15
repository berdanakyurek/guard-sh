package config

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	xdg := t.TempDir()
	dir := xdg + "/guard-sh"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := dir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
	return path
}

const baseConfig = `provider_order:
  - gemini
  - openai

providers:
  gemini:
    api_key: key1
    model: gemini-3.1-flash-lite-preview

  openai:
    api_key: key2
    model: gpt-4o-mini

command_whitelist:
  - ls
  - cat
`

func TestLoad_Valid(t *testing.T) {
	writeConfig(t, baseConfig)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.ProviderOrder) != 2 {
		t.Errorf("expected 2 providers, got %d", len(cfg.ProviderOrder))
	}
	if cfg.ProviderOrder[0] != "gemini" {
		t.Errorf("expected first provider gemini, got %q", cfg.ProviderOrder[0])
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := Load()
	if err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}

func TestLoad_EmptyProviderOrder(t *testing.T) {
	writeConfig(t, "provider_order: []\n")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error for empty provider_order: %v", err)
	}
	if len(cfg.ProviderOrder) != 0 {
		t.Errorf("expected empty provider_order, got %v", cfg.ProviderOrder)
	}
}

func TestDefaultModel(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"gemini", "gemini-3.1-flash-lite-preview"},
		{"claude", "claude-haiku-4-5-20251001"},
		{"openai", "gpt-4o-mini"},
		{"deepseek", "deepseek-chat"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		got := DefaultModel(tt.provider)
		if got != tt.want {
			t.Errorf("DefaultModel(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestUpdateWhitelist(t *testing.T) {
	writeConfig(t, baseConfig)
	newList := []string{"ls", "cat", "git", "echo"}
	if err := UpdateWhitelist(newList); err != nil {
		t.Fatalf("UpdateWhitelist: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load after update: %v", err)
	}
	if !slices.Equal(cfg.CommandWhitelist, newList) {
		t.Errorf("got %v, want %v", cfg.CommandWhitelist, newList)
	}
}

func TestAddProvider_New(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := AddProvider("claude", "mykey", "claude-haiku-4-5-20251001"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.ProviderOrder, "claude") {
		t.Error("expected claude in provider_order")
	}
	if p := cfg.Providers["claude"]; p == nil || p.APIKey != "mykey" {
		t.Error("expected claude provider block with correct api_key")
	}
}

func TestAddProvider_Upsert(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := AddProvider("gemini", "newkey", "gemini-2.0-flash"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// should not duplicate in provider_order
	count := 0
	for _, name := range cfg.ProviderOrder {
		if name == "gemini" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected gemini once in provider_order, got %d", count)
	}
	if p := cfg.Providers["gemini"]; p == nil || p.APIKey != "newkey" {
		t.Error("expected updated api_key for gemini")
	}
}

func TestRemoveProvider(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := RemoveProvider("openai"); err != nil {
		t.Fatalf("RemoveProvider: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(cfg.ProviderOrder, "openai") {
		t.Error("expected openai removed from provider_order")
	}
	if _, ok := cfg.Providers["openai"]; ok {
		t.Error("expected openai provider block removed")
	}
}

func TestUpdateProviderOrder(t *testing.T) {
	writeConfig(t, baseConfig)
	newOrder := []string{"openai", "gemini"}
	if err := UpdateProviderOrder(newOrder); err != nil {
		t.Fatalf("UpdateProviderOrder: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.ProviderOrder, newOrder) {
		t.Errorf("got %v, want %v", cfg.ProviderOrder, newOrder)
	}
}

func TestUpdateCacheEnabled(t *testing.T) {
	writeConfig(t, baseConfig+"cache_enabled: true\n")
	if err := UpdateCacheEnabled(false); err != nil {
		t.Fatalf("UpdateCacheEnabled: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheEnabled == nil || *cfg.CacheEnabled {
		t.Error("expected cache_enabled=false")
	}
}

func TestUpdateCacheMaxSize(t *testing.T) {
	writeConfig(t, baseConfig+"cache_max_size: 1000\n")
	if err := UpdateCacheMaxSize(500); err != nil {
		t.Fatalf("UpdateCacheMaxSize: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CacheMaxSize != 500 {
		t.Errorf("got %d, want 500", cfg.CacheMaxSize)
	}
}

// --- edge cases ---

// Comments in the config file must survive any manipulation unchanged.
func TestComments_PreservedAfterManipulation(t *testing.T) {
	content := `# top comment
provider_order:
  # order comment
  - gemini

providers:
  # providers comment
  gemini:
    api_key: key1
    model: gemini-3.1-flash-lite-preview

# whitelist comment
command_whitelist:
  - ls
`
	path := writeConfig(t, content)

	if err := AddProvider("claude", "key2", "claude-haiku-4-5-20251001"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := UpdateWhitelist([]string{"ls", "cat"}); err != nil {
		t.Fatalf("UpdateWhitelist: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = string(data)
	for _, comment := range []string{"# top comment", "# order comment", "# providers comment", "# whitelist comment"} {
		if !strings.Contains(content, comment) {
			t.Errorf("comment %q was lost after manipulation", comment)
		}
	}
}

// Unrelated sections (timeout, cache settings) must survive provider manipulation.
func TestOtherSections_PreservedAfterProviderAdd(t *testing.T) {
	content := baseConfig + `timeout_seconds: 15
cache_enabled: true
cache_max_size: 500
`
	path := writeConfig(t, content)

	if err := AddProvider("claude", "key3", "claude-haiku-4-5-20251001"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TimeoutSeconds != 15 {
		t.Errorf("timeout_seconds lost: got %d, want 15", cfg.TimeoutSeconds)
	}
	if cfg.CacheMaxSize != 500 {
		t.Errorf("cache_max_size lost: got %d, want 500", cfg.CacheMaxSize)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "cache_enabled: true") {
		t.Error("cache_enabled lost after AddProvider")
	}
}

// UpdateWhitelist with an empty list should result in an empty whitelist section.
func TestUpdateWhitelist_Empty(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := UpdateWhitelist([]string{}); err != nil {
		t.Fatalf("UpdateWhitelist: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CommandWhitelist) != 0 {
		t.Errorf("expected empty whitelist, got %v", cfg.CommandWhitelist)
	}
}

// RemoveProvider on an unknown name should not error or corrupt the config.
func TestRemoveProvider_Unknown(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := RemoveProvider("unknown"); err != nil {
		t.Fatalf("RemoveProvider unknown: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.ProviderOrder, []string{"gemini", "openai"}) {
		t.Errorf("provider_order changed unexpectedly: %v", cfg.ProviderOrder)
	}
}

func TestIsSendWorkingDirectoryEnabled(t *testing.T) {
	trueVal := true
	falseVal := false

	cases := []struct {
		cfg  *Config
		want bool
	}{
		{&Config{}, true},                                  // nil pointer → default true
		{&Config{SendWorkingDirectory: &trueVal}, true},   // explicit true
		{&Config{SendWorkingDirectory: &falseVal}, false}, // explicit false
	}
	for _, c := range cases {
		got := c.cfg.IsSendWorkingDirectoryEnabled()
		if got != c.want {
			t.Errorf("IsSendWorkingDirectoryEnabled() = %v, want %v", got, c.want)
		}
	}
}

func TestEffectivePatternRedaction(t *testing.T) {
	enabled := true
	disabled := false
	global := &PatternRedactionConfig{Enabled: &enabled, Patterns: []string{"pat1"}}

	// nil provider → falls back to global
	var p *ProviderConfig
	got := p.EffectivePatternRedaction(global)
	if got != global {
		t.Error("nil provider should return global config")
	}

	// provider with no redaction override → falls back to global
	p2 := &ProviderConfig{}
	got2 := p2.EffectivePatternRedaction(global)
	if got2 != global {
		t.Error("provider with no redaction should return global config")
	}

	// provider overrides enabled=false, inherits patterns from global
	p3 := &ProviderConfig{Redaction: &RedactionConfig{PatternBased: &PatternRedactionConfig{Enabled: &disabled}}}
	got3 := p3.EffectivePatternRedaction(global)
	if got3.IsEnabled() {
		t.Error("expected provider override to disable pattern redaction")
	}
	if len(got3.GetPatterns()) == 0 {
		t.Error("expected patterns inherited from global")
	}

	// provider overrides patterns, inherits enabled from global
	p4 := &ProviderConfig{Redaction: &RedactionConfig{PatternBased: &PatternRedactionConfig{Patterns: []string{"pat2", "pat3"}}}}
	got4 := p4.EffectivePatternRedaction(global)
	if !got4.IsEnabled() {
		t.Error("expected enabled inherited from global")
	}
	if len(got4.GetPatterns()) != 2 || got4.GetPatterns()[0] != "pat2" {
		t.Errorf("expected provider patterns, got %v", got4.GetPatterns())
	}
}

func TestEffectiveEntropyRedaction(t *testing.T) {
	enabled := true
	disabled := false
	global := &EntropyRedactionConfig{Enabled: &enabled, Threshold: 4.5, MinLength: 20}

	// nil provider → falls back to global
	var p *ProviderConfig
	got := p.EffectiveEntropyRedaction(global)
	if got != global {
		t.Error("nil provider should return global config")
	}

	// provider overrides threshold only
	p2 := &ProviderConfig{Redaction: &RedactionConfig{ShannonEntropyBased: &EntropyRedactionConfig{Threshold: 3.5}}}
	got2 := p2.EffectiveEntropyRedaction(global)
	if got2.GetThreshold() != 3.5 {
		t.Errorf("expected threshold 3.5, got %f", got2.GetThreshold())
	}
	if got2.GetMinLength() != 20 {
		t.Errorf("expected min_length inherited from global (20), got %d", got2.GetMinLength())
	}

	// provider disables entropy
	p3 := &ProviderConfig{Redaction: &RedactionConfig{ShannonEntropyBased: &EntropyRedactionConfig{Enabled: &disabled}}}
	got3 := p3.EffectiveEntropyRedaction(global)
	if got3.IsEnabled() {
		t.Error("expected provider override to disable entropy redaction")
	}
}

func TestUpdateProviderPatternRedaction(t *testing.T) {
	writeConfig(t, baseConfig)

	// Enable for gemini (no existing redaction block)
	if err := UpdateProviderPatternRedaction("gemini", true); err != nil {
		t.Fatalf("UpdateProviderPatternRedaction enable: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	eff := cfg.Providers["gemini"].EffectivePatternRedaction(nil)
	if !eff.IsEnabled() {
		t.Error("expected pattern redaction enabled for gemini")
	}

	// Disable it
	if err := UpdateProviderPatternRedaction("gemini", false); err != nil {
		t.Fatalf("UpdateProviderPatternRedaction disable: %v", err)
	}
	cfg2, _ := Load()
	eff2 := cfg2.Providers["gemini"].EffectivePatternRedaction(nil)
	if eff2.IsEnabled() {
		t.Error("expected pattern redaction disabled for gemini")
	}

	// Unknown provider → error
	if err := UpdateProviderPatternRedaction("unknown", true); err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestUpdateProviderEntropyRedaction(t *testing.T) {
	writeConfig(t, baseConfig)

	if err := UpdateProviderEntropyRedaction("openai", true); err != nil {
		t.Fatalf("UpdateProviderEntropyRedaction: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	eff := cfg.Providers["openai"].EffectiveEntropyRedaction(nil)
	if !eff.IsEnabled() {
		t.Error("expected entropy redaction enabled for openai")
	}
	// default threshold and min_length should be written
	if eff.GetThreshold() != 4.5 {
		t.Errorf("expected default threshold 4.5, got %f", eff.GetThreshold())
	}
	if eff.GetMinLength() != 20 {
		t.Errorf("expected default min_length 20, got %d", eff.GetMinLength())
	}
}

func TestAddOllamaProvider(t *testing.T) {
	writeConfig(t, baseConfig)

	if err := AddOllamaProvider("http://localhost:11434/api/chat", "llama3.2"); err != nil {
		t.Fatalf("AddOllamaProvider: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(cfg.ProviderOrder, "ollama") {
		t.Error("ollama not in provider_order after add")
	}
	p, ok := cfg.Providers["ollama"]
	if !ok {
		t.Fatal("ollama provider block not found")
	}
	if p.Host != "http://localhost:11434/api/chat" {
		t.Errorf("unexpected host: %q", p.Host)
	}
	if p.Model != "llama3.2" {
		t.Errorf("unexpected model: %q", p.Model)
	}

	// Calling again should upsert (not duplicate)
	if err := AddOllamaProvider("http://localhost:11434/api/chat", "llama3.2"); err != nil {
		t.Fatalf("AddOllamaProvider upsert: %v", err)
	}
	cfg2, _ := Load()
	count := 0
	for _, name := range cfg2.ProviderOrder {
		if name == "ollama" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("ollama appears %d times in provider_order, want 1", count)
	}
}

// AddProvider followed by RemoveProvider should leave the config clean.
func TestAddThenRemoveProvider(t *testing.T) {
	writeConfig(t, baseConfig)
	if err := AddProvider("deepseek", "dskey", "deepseek-chat"); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := RemoveProvider("deepseek"); err != nil {
		t.Fatalf("RemoveProvider: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(cfg.ProviderOrder, "deepseek") {
		t.Error("deepseek still in provider_order after remove")
	}
	if _, ok := cfg.Providers["deepseek"]; ok {
		t.Error("deepseek provider block still present after remove")
	}
}

