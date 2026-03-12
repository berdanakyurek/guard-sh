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
		g := New(&mockProvider{}, "", "", tt.wl, nil)
		safe, _ := g.Check(context.Background(), tt.cmd)
		if safe != tt.wantSafe {
			t.Errorf("cmd=%q whitelist=%v: got safe=%v, want %v", tt.cmd, tt.wl, safe, tt.wantSafe)
		}
	}
}

func TestBlacklist(t *testing.T) {
	tests := []struct {
		cmd      string
		bl       []string
		wantSafe bool
	}{
		{"rm -rf /", []string{"rm"}, false},
		{"rm file.txt", []string{"rm"}, false},
		{"ls", []string{"rm"}, false},          // goes to mock provider → mock warning → unsafe
		{"ls && rm -rf /", []string{"rm"}, false}, // rm is blacklisted
		{"cd /tmp && ls", []string{"rm"}, false},  // neither blacklisted, goes to provider → mock warning
		{"mkfs /dev/sda", []string{"mkfs"}, false},
		{"ls | rm -rf .", []string{"rm"}, false}, // rm blacklisted
	}

	for _, tt := range tests {
		g := New(&mockProvider{}, "", "", nil, tt.bl)
		safe, warning := g.Check(context.Background(), tt.cmd)
		if safe != tt.wantSafe {
			t.Errorf("cmd=%q blacklist=%v: got safe=%v, want %v", tt.cmd, tt.bl, safe, tt.wantSafe)
		}
		// When blacklisted, warning should mention the command
		if tt.cmd == "rm -rf /" || tt.cmd == "rm file.txt" || tt.cmd == "ls && rm -rf /" || tt.cmd == "ls | rm -rf ." {
			if warning == "" {
				t.Errorf("cmd=%q: expected non-empty blacklist warning", tt.cmd)
			}
		}
	}
}

func TestBlacklistBeatsWhitelist(t *testing.T) {
	// A command in both lists: blacklist wins
	g := New(&mockProvider{}, "", "", []string{"rm"}, []string{"rm"})
	safe, warning := g.Check(context.Background(), "rm file.txt")
	if safe {
		t.Error("blacklisted command should not be safe even if whitelisted")
	}
	if warning == "" {
		t.Error("expected blacklist warning")
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
