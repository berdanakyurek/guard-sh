package guard

import (
	"context"
	"testing"
)

type mockProvider struct{}

func (m *mockProvider) Query(_ context.Context, _, _ string) (string, error) {
	return "mock warning", nil
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
