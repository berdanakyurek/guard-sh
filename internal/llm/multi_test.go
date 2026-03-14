package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/Berdan/guard-sh/internal/guard"
)

type mockProvider struct {
	response string
	err      error
	called   int
}

func (m *mockProvider) Query(_ context.Context, _, _ string) (string, error) {
	m.called++
	return m.response, m.err
}

func TestMulti_FirstProviderSucceeds(t *testing.T) {
	p1 := &mockProvider{response: "OK"}
	p2 := &mockProvider{response: "fallback"}
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil)

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
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil)

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
	m := NewMulti([]string{"p1", "p2"}, []guard.Provider{p1, p2}, nil)

	_, err := m.Query(context.Background(), "", "rm -rf /")
	if err == nil {
		t.Error("expected error when all providers fail, got nil")
	}
	if p1.called != 1 || p2.called != 1 {
		t.Errorf("expected both providers tried, p1=%d p2=%d", p1.called, p2.called)
	}
}

func TestMulti_EmptyProviders(t *testing.T) {
	m := NewMulti(nil, nil, nil)
	_, err := m.Query(context.Background(), "", "ls")
	if err == nil {
		t.Error("expected error with no providers, got nil")
	}
}
