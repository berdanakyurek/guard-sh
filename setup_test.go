package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupEnv(t *testing.T, shell string) (xdg, home string) {
	t.Helper()
	xdg = t.TempDir()
	home = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", home)
	t.Setenv("SHELL", shell)
	return
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	return string(data)
}

func TestSetup_FreshInstall_Bash(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"

	runSetup()

	for _, name := range []string{"config.yaml", "prompt.txt", "guard.bash", "guard.zsh"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not created: %v", name, err)
		}
	}

	rc := readFile(t, filepath.Join(home, ".bashrc"))
	bashScript := filepath.Join(dir, "guard.bash")
	if !strings.Contains(rc, `source "`+bashScript+`"`) {
		t.Error(".bashrc missing correct source line")
	}
	if !strings.Contains(rc, "guard-sh on") {
		t.Error(".bashrc missing guard-sh on")
	}
}

func TestSetup_FreshInstall_Zsh(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/zsh")
	dir := xdg + "/guard-sh"

	runSetup()

	rc := readFile(t, filepath.Join(home, ".zshrc"))
	zshScript := filepath.Join(dir, "guard.zsh")
	if !strings.Contains(rc, `source "`+zshScript+`"`) {
		t.Error(".zshrc missing correct source line")
	}
	if !strings.Contains(rc, "guard-sh on") {
		t.Error(".zshrc missing guard-sh on")
	}
}

func TestSetup_ExistingConfig_NotOverwritten(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	original := "# my custom config\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	runSetup()

	got := readFile(t, configPath)
	if got != original {
		t.Errorf("config.yaml was overwritten: got %q, want %q", got, original)
	}
}

func TestSetup_ExistingPrompt_NotOverwritten(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	promptPath := filepath.Join(dir, "prompt.txt")
	original := "my custom prompt\n"
	if err := os.WriteFile(promptPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	runSetup()

	got := readFile(t, promptPath)
	if got != original {
		t.Errorf("prompt.txt was overwritten: got %q, want %q", got, original)
	}
}

func TestSetup_ShellIntegration_AlreadyPresent(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	bashScript := filepath.Join(dir, "guard.bash")

	// Pre-populate .bashrc with integration
	bashrc := filepath.Join(home, ".bashrc")
	existing := "\n# guard-sh\nsource \"" + bashScript + "\"\nguard-sh on\n"
	if err := os.WriteFile(bashrc, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	runSetup()

	rc := readFile(t, bashrc)
	count := strings.Count(rc, "guard-sh on")
	if count != 1 {
		t.Errorf("guard-sh on duplicated: appears %d times", count)
	}
	count = strings.Count(rc, `source "`+bashScript+`"`)
	if count != 1 {
		t.Errorf("source line duplicated: appears %d times", count)
	}
}

func TestSetup_ShellIntegration_SourcePresent_OnMissing(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	bashScript := filepath.Join(dir, "guard.bash")

	// .bashrc has source line but no guard-sh on
	bashrc := filepath.Join(home, ".bashrc")
	existing := "\n# guard-sh\nsource \"" + bashScript + "\"\n"
	if err := os.WriteFile(bashrc, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	runSetup()

	rc := readFile(t, bashrc)
	if !strings.Contains(rc, "guard-sh on") {
		t.Error("guard-sh on not added when source was present but on was missing")
	}
}

func TestSetup_UnsupportedShell_NoRcWritten(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/fish")

	runSetup()

	// No rc file should be created for unsupported shells
	for _, name := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s should not have been created for unsupported shell", name)
		}
	}

	// Config files should still be created
	dir := xdg + "/guard-sh"
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Error("config.yaml should still be created for unsupported shell")
	}
}

func TestSetup_ShellScripts_AlwaysOverwritten(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	// Write stale content to shell scripts
	for _, name := range []string{"guard.bash", "guard.zsh"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("stale"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	runSetup()

	// Shell scripts should be refreshed
	bash := readFile(t, filepath.Join(dir, "guard.bash"))
	if bash == "stale" {
		t.Error("guard.bash was not updated")
	}
	if !strings.Contains(bash, "_guard_debug_trap") {
		t.Error("guard.bash does not look like a valid bash integration script")
	}

	zsh := readFile(t, filepath.Join(dir, "guard.zsh"))
	if zsh == "stale" {
		t.Error("guard.zsh was not updated")
	}
	if !strings.Contains(zsh, "_guard_zsh_accept_line") {
		t.Error("guard.zsh does not look like a valid zsh integration script")
	}
}
