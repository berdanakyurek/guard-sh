package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs f and returns whatever it printed to os.Stdout.
func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRunStatus_RedactionPatterns_Listed(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, `
provider_order: []
providers: {}
redaction:
  pattern_based:
    enabled: true
    patterns:
      - 'pattern-one'
      - 'pattern-two'
      - 'pattern-three'
`)

	out := captureStdout(func() { runStatus(nil) })

	if !strings.Contains(out, "pattern-one") {
		t.Error("status output missing pattern-one")
	}
	if !strings.Contains(out, "pattern-two") {
		t.Error("status output missing pattern-two")
	}
	if !strings.Contains(out, "pattern-three") {
		t.Error("status output missing pattern-three")
	}
	if strings.Contains(out, "global patterns") {
		t.Error("status output should not contain old 'global patterns' count line")
	}
}

func TestRunStatus_RedactionPatterns_TruncatedAt10(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, `
provider_order: []
providers: {}
redaction:
  pattern_based:
    enabled: true
    patterns:
      - 'p1'
      - 'p2'
      - 'p3'
      - 'p4'
      - 'p5'
      - 'p6'
      - 'p7'
      - 'p8'
      - 'p9'
      - 'p10'
      - 'p11'
      - 'p12'
`)

	out := captureStdout(func() { runStatus(nil) })

	if !strings.Contains(out, "p10") {
		t.Error("status output missing p10 (should show up to 10)")
	}
	if strings.Contains(out, "p11") {
		t.Error("status output should not show p11 (truncated at 10)")
	}
	if !strings.Contains(out, "+2 more") {
		t.Error("status output missing '+2 more' truncation line")
	}
	if !strings.Contains(out, "guard-sh redact list") {
		t.Error("status truncation line should mention 'guard-sh redact list'")
	}
}

func TestRunStatus_RedactionPatterns_NoSection_WhenEmpty(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, `
provider_order: []
providers: {}
redaction:
  pattern_based:
    enabled: true
    patterns: []
`)

	out := captureStdout(func() { runStatus(nil) })

	if strings.Contains(out, "redaction") {
		t.Error("status should not show redaction section when there are no patterns")
	}
}

func TestRunRedactList(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, `
provider_order: []
providers: {}
redaction:
  pattern_based:
    enabled: true
    patterns:
      - 'alpha'
      - 'beta'
      - 'gamma'
`)

	out := captureStdout(runRedactList)

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "alpha" || lines[1] != "beta" || lines[2] != "gamma" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestRunRedactList_Empty(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, `
provider_order: []
providers: {}
redaction:
  pattern_based:
    enabled: true
    patterns: []
`)

	out := captureStdout(runRedactList)

	if out != "" {
		t.Errorf("expected empty output for empty patterns, got %q", out)
	}
}
