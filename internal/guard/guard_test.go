package guard

import (
	"context"
	"errors"
	"os"
	"testing"
)

type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Query(_ context.Context, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.response == "" {
		return "mock warning", nil
	}
	return m.response, nil
}

func TestWhitelist(t *testing.T) {
	tests := []struct {
		cmd      string
		wl       []string
		wantSafe bool
	}{
		{"ls", []string{"ls"}, true},
		{"ls -la", []string{"ls"}, true},
		{"ls /tmp/foo", []string{"ls"}, true},
		{"ls -la && rm -rf /", []string{"ls"}, false},
		{"ls | grep foo", []string{"ls", "grep"}, true},
		{"ls | grep foo", []string{"ls"}, false},
		{"cd /tmp && ls", []string{"ls", "cd"}, true},
		{"cd /tmp && rm -rf .", []string{"ls", "cd"}, false},
		{"clear", []string{"ls", "clear"}, true},
		{"FOO=bar ls", []string{"ls"}, true},
		{"(ls)", []string{"ls"}, true},
		{"ls; cat file", []string{"ls", "cat"}, true},
		{"ls; cat file", []string{"ls"}, false},
		{"rm -rf /", []string{"ls"}, false},
		{"ls || echo hi", []string{"ls", "echo"}, true},
		{"ls || rm -rf /", []string{"ls", "echo"}, false},
	}

	for _, tt := range tests {
		g := New(&mockProvider{}, "", "", tt.wl, 0, nil)
		safe, _ := g.Check(context.Background(), tt.cmd)
		if safe != tt.wantSafe {
			t.Errorf("cmd=%q whitelist=%v: got safe=%v, want %v", tt.cmd, tt.wl, safe, tt.wantSafe)
		}
	}
}

func TestExtractBaseCommands(t *testing.T) {
	tests := []struct {
		cmd  string
		want []string
	}{
		{"ls", []string{"ls"}},
		{"ls -la", []string{"ls"}},
		{"ls && rm -rf /", []string{"ls", "rm"}},
		{"ls || echo hi", []string{"ls", "echo"}},
		{"ls; cat file", []string{"ls", "cat"}},
		{"ls | grep foo", []string{"ls", "grep"}},
		{"FOO=bar ls", []string{"ls"}},
		{"(ls)", []string{"ls"}},
		{"ls -la && rm -rf / ; echo done", []string{"ls", "rm", "echo"}},
	}

	for _, tt := range tests {
		got := extractBaseCommands(tt.cmd)
		if len(got) != len(tt.want) {
			t.Errorf("extractBaseCommands(%q): got %v, want %v", tt.cmd, got, tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("extractBaseCommands(%q)[%d]: got %q, want %q", tt.cmd, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCheck_SafeResponse(t *testing.T) {
	g := New(&mockProvider{response: "OK"}, "", "", nil, 0, nil)
	safe, warning := g.Check(context.Background(), "rm -rf /")
	if !safe {
		t.Errorf("expected safe=true, got false (warning=%q)", warning)
	}
}

func TestCheck_UnsafeResponse(t *testing.T) {
	g := New(&mockProvider{response: "Deletes everything"}, "", "", nil, 0, nil)
	safe, warning := g.Check(context.Background(), "rm -rf /")
	if safe {
		t.Error("expected safe=false, got true")
	}
	if warning != "Deletes everything" {
		t.Errorf("got warning=%q, want %q", warning, "Deletes everything")
	}
}

func TestCheck_ProviderError_FailsOpen(t *testing.T) {
	g := New(&mockProvider{err: errors.New("network error")}, "", "", nil, 0, nil)
	safe, _ := g.Check(context.Background(), "rm -rf /")
	if safe {
		t.Error("expected safe=false when provider errors (fail open still prompts)")
	}
}

func TestCheck_CacheHit(t *testing.T) {
	dir := t.TempDir()
	called := 0
	p := &countingProvider{response: "OK", onCall: func() { called++ }}
	g := New(p, "", dir, nil, 100, nil)

	g.Check(context.Background(), "ls -la")
	g.Check(context.Background(), "ls -la")

	if called != 1 {
		t.Errorf("expected provider called once (cache hit on second), got %d", called)
	}
}

func TestCheck_CacheDisabled(t *testing.T) {
	called := 0
	p := &countingProvider{response: "OK", onCall: func() { called++ }}
	g := New(p, "", "", nil, 0, nil) // cacheMaxSize=0 disables cache

	g.Check(context.Background(), "ls -la")
	g.Check(context.Background(), "ls -la")

	if called != 2 {
		t.Errorf("expected provider called twice (no cache), got %d", called)
	}
}

func TestCheck_CustomPrompt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/prompt.txt", []byte("custom prompt"), 0600); err != nil {
		t.Fatal(err)
	}
	var receivedPrompt string
	p := &capturingProvider{onQuery: func(prompt, _ string) { receivedPrompt = prompt }}
	g := New(p, "default prompt", dir, nil, 0, nil)
	g.Check(context.Background(), "ls")
	if receivedPrompt != "custom prompt" {
		t.Errorf("expected custom prompt, got %q", receivedPrompt)
	}
}

func TestCheck_DefaultPrompt(t *testing.T) {
	var receivedPrompt string
	p := &capturingProvider{onQuery: func(prompt, _ string) { receivedPrompt = prompt }}
	g := New(p, "default prompt", t.TempDir(), nil, 0, nil)
	g.Check(context.Background(), "ls")
	if receivedPrompt != "default prompt" {
		t.Errorf("expected default prompt, got %q", receivedPrompt)
	}
}

type countingProvider struct {
	response string
	onCall   func()
}

func (p *countingProvider) Query(_ context.Context, _, _ string) (string, error) {
	p.onCall()
	return p.response, nil
}

type capturingProvider struct {
	onQuery func(prompt, cmd string)
}

func (p *capturingProvider) Query(_ context.Context, prompt, cmd string) (string, error) {
	p.onQuery(prompt, cmd)
	return "OK", nil
}
