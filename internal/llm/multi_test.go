package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/Berdan/guard-sh/internal/cache"
	"github.com/Berdan/guard-sh/internal/guard"
	"github.com/Berdan/guard-sh/internal/redact"
)

type mockProvider struct {
	response    string
	err         error
	called      int
	lastCommand string
}

func (m *mockProvider) Query(_ context.Context, _, cmd string) (string, error) {
	m.called++
	m.lastCommand = cmd
	return m.response, m.err
}

func TestMulti_FirstProviderSucceeds(t *testing.T) {
	p1 := &mockProvider{response: "OK"}
	p2 := &mockProvider{response: "fallback"}
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil, nil, nil, nil, nil)

	result, err := m.Query(context.Background(), "", "ls")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "OK" {
		t.Errorf("got %q, want %q", result, "OK")
	}
	if p1.called != 1 {
		t.Errorf("expected p1 called once, got %d", p1.called)
	}
	if p2.called != 0 {
		t.Errorf("expected p2 not called, got %d", p2.called)
	}
}

func TestMulti_FallbackOnError(t *testing.T) {
	p1 := &mockProvider{err: errors.New("rate limit")}
	p2 := &mockProvider{response: "Deletes everything"}
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil, nil, nil, nil, nil)

	result, err := m.Query(context.Background(), "", "rm -rf /")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Deletes everything" {
		t.Errorf("got %q, want %q", result, "Deletes everything")
	}
	if p1.called != 1 {
		t.Errorf("expected p1 called once, got %d", p1.called)
	}
	if p2.called != 1 {
		t.Errorf("expected p2 called once, got %d", p2.called)
	}
}

func TestMulti_AllFail(t *testing.T) {
	p1 := &mockProvider{err: errors.New("error 1")}
	p2 := &mockProvider{err: errors.New("error 2")}
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil, nil, nil, nil, nil)

	_, err := m.Query(context.Background(), "", "rm -rf /")
	if err == nil {
		t.Error("expected error when all providers fail, got nil")
	}
	if p1.called != 1 || p2.called != 1 {
		t.Errorf("expected both providers tried, p1=%d p2=%d", p1.called, p2.called)
	}
}

func TestMulti_EmptyProviders(t *testing.T) {
	m := NewMulti(nil, nil, nil, nil, nil, nil, nil)
	_, err := m.Query(context.Background(), "", "ls")
	if err == nil {
		t.Error("expected error with no providers, got nil")
	}
}

func TestMulti_PatternRedactionApplied(t *testing.T) {
	p := &mockProvider{response: "OK"}
	r, err := redact.New([]string{`(?i)password=\S+`})
	if err != nil {
		t.Fatal(err)
	}
	patternRedactors := map[string]*redact.Redactor{"p1": r}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, patternRedactors, nil, nil, nil)

	_, err = m.Query(context.Background(), "", "mysql password=secret")
	if err != nil {
		t.Fatal(err)
	}
	if p.lastCommand != "mysql [REDACTED]" {
		t.Errorf("expected redacted command, got %q", p.lastCommand)
	}
}

func TestMulti_PatternRedactionDisabledForProvider(t *testing.T) {
	p := &mockProvider{response: "OK"}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, nil, nil, nil)

	_, err := m.Query(context.Background(), "", "mysql password=secret")
	if err != nil {
		t.Fatal(err)
	}
	if p.lastCommand != "mysql password=secret" {
		t.Errorf("expected original command when pattern redaction disabled, got %q", p.lastCommand)
	}
}

func TestMulti_PatternRedactionPerProvider(t *testing.T) {
	p1 := &mockProvider{response: "OK"}
	p2 := &mockProvider{err: errors.New("fail")}
	p3 := &mockProvider{response: "OK"}
	r, err := redact.New([]string{`(?i)password=\S+`})
	if err != nil {
		t.Fatal(err)
	}
	patternRedactors := map[string]*redact.Redactor{"p1": r}
	m := NewMulti([]string{"p1", "p2", "p3"}, []guard.Provider{p1, p2, p3}, nil, patternRedactors, nil, nil, nil)

	m.Query(context.Background(), "", "mysql password=secret")

	if p1.lastCommand != "mysql [REDACTED]" {
		t.Errorf("p1: expected redacted, got %q", p1.lastCommand)
	}
}

func TestMulti_EntropyRedactionApplied(t *testing.T) {
	p := &mockProvider{response: "OK"}
	er := redact.NewEntropyRedactor(4.5, 20)
	entropyRedactors := map[string]*redact.EntropyRedactor{"p1": er}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, entropyRedactors, nil, nil)

	secret := "ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
	_, err := m.Query(context.Background(), "", "curl -H Authorization:"+secret)
	if err != nil {
		t.Fatal(err)
	}
	if p.lastCommand == "curl -H Authorization:"+secret {
		t.Errorf("expected entropy redaction, command unchanged: %q", p.lastCommand)
	}
}

func TestMulti_EntropyRedactionDisabledForProvider(t *testing.T) {
	p := &mockProvider{response: "OK"}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, nil, nil, nil)

	input := "ls -la /tmp"
	_, err := m.Query(context.Background(), "", input)
	if err != nil {
		t.Fatal(err)
	}
	if p.lastCommand != input {
		t.Errorf("expected unchanged command when entropy disabled, got %q", p.lastCommand)
	}
}

func TestMulti_PatternAndEntropyRedactionBothApplied(t *testing.T) {
	p := &mockProvider{response: "OK"}
	r, err := redact.New([]string{`(?i)password=\S+`})
	if err != nil {
		t.Fatal(err)
	}
	er := redact.NewEntropyRedactor(4.5, 20)
	patternRedactors := map[string]*redact.Redactor{"p1": r}
	entropyRedactors := map[string]*redact.EntropyRedactor{"p1": er}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, patternRedactors, entropyRedactors, nil, nil)

	secret := "ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
	input := "curl password=foo token:" + secret
	_, err = m.Query(context.Background(), "", input)
	if err != nil {
		t.Fatal(err)
	}
	if p.lastCommand == input {
		t.Errorf("expected redaction to occur, got unchanged: %q", p.lastCommand)
	}
}

func TestMulti_CacheHit(t *testing.T) {
	p := &mockProvider{response: "OK"}
	c := cache.Load(t.TempDir(), 100)
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, nil, c, nil)

	m.Query(context.Background(), "", "ls -la")
	m.Query(context.Background(), "", "ls -la")

	if p.called != 1 {
		t.Errorf("expected provider called once (cache hit on second), got %d", p.called)
	}
}

func TestMulti_CacheKeyIsPerProvider(t *testing.T) {
	p1 := &mockProvider{err: errors.New("fail")}
	p2 := &mockProvider{response: "Risky"}
	c := cache.Load(t.TempDir(), 100)
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil, nil, nil, c, nil)

	// First call: p1 fails, p2 responds
	m.Query(context.Background(), "", "rm -rf /")

	// Second call: p1 should still be tried (its cache entry is empty), p2 cache should not bypass p1
	p1.err = nil
	p1.response = "OK"
	m.Query(context.Background(), "", "rm -rf /")

	if p1.called != 2 {
		t.Errorf("expected p1 tried on second call too (its own cache was empty), got p1.called=%d", p1.called)
	}
}

type promptCapturingProvider struct {
	lastPrompt string
}

func (p *promptCapturingProvider) Query(_ context.Context, prompt, _ string) (string, error) {
	p.lastPrompt = prompt
	return "OK", nil
}

func TestMulti_ProviderSpecificPromptOverridesGlobal(t *testing.T) {
	p := &promptCapturingProvider{}
	prompts := map[string]string{"p1": "custom prompt for p1"}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, prompts, nil, nil, nil, nil)
	m.Query(context.Background(), "global prompt", "ls")
	if p.lastPrompt != "custom prompt for p1" {
		t.Errorf("expected per-provider prompt, got %q", p.lastPrompt)
	}
}

func TestMulti_GlobalPromptUsedWhenNoOverride(t *testing.T) {
	p := &promptCapturingProvider{}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, nil, nil, nil)
	m.Query(context.Background(), "global prompt", "ls")
	if p.lastPrompt != "global prompt" {
		t.Errorf("expected global prompt, got %q", p.lastPrompt)
	}
}

func TestMulti_CacheDisabled(t *testing.T) {
	p := &mockProvider{response: "OK"}
	m := NewMulti([]string{"p1"}, []guard.Provider{p}, nil, nil, nil, nil, nil)

	m.Query(context.Background(), "", "ls -la")
	m.Query(context.Background(), "", "ls -la")

	if p.called != 2 {
		t.Errorf("expected provider called twice (no cache), got %d", p.called)
	}
}
