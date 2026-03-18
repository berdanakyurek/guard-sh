package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveShellIntegration_RemovesLines(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	content := "# existing content\n\n# guard-sh\nsource \"/home/user/.config/guard-sh/guard.bash\"\nguard-sh on\n\n# other stuff\n"
	if err := os.WriteFile(bashrc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	removed := removeShellIntegration(bashrc)

	if !removed {
		t.Fatal("expected removeShellIntegration to return true")
	}
	got := readFile(t, bashrc)
	if strings.Contains(got, "guard-sh") {
		t.Errorf("rc file still contains guard-sh content: %q", got)
	}
	if !strings.Contains(got, "# existing content") {
		t.Error("rc file lost non-guard-sh content")
	}
	if !strings.Contains(got, "# other stuff") {
		t.Error("rc file lost trailing non-guard-sh content")
	}
}

func TestRemoveShellIntegration_ReturnsFalseWhenAbsent(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	content := "# unrelated content\nexport PATH=$PATH:~/bin\n"
	if err := os.WriteFile(bashrc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	removed := removeShellIntegration(bashrc)

	if removed {
		t.Error("expected false for file with no guard-sh content")
	}
	if got := readFile(t, bashrc); got != content {
		t.Errorf("file was modified unexpectedly: %q", got)
	}
}

func TestRemoveShellIntegration_ReturnsFalseForMissingFile(t *testing.T) {
	removed := removeShellIntegration("/nonexistent/path/.bashrc")
	if removed {
		t.Error("expected false for missing file")
	}
}

func TestRemoveShellIntegration_CollapsesBlankLines(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	content := "line1\n\n# guard-sh\nsource \"x/guard.bash\"\nguard-sh on\n\nline2\n"
	if err := os.WriteFile(bashrc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	removeShellIntegration(bashrc)

	got := readFile(t, bashrc)
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("file has triple blank lines after removal: %q", got)
	}
}

func TestRemoveShellIntegration_AllShellMarkers(t *testing.T) {
	for _, marker := range []string{"guard.bash", "guard.zsh", "guard.fish"} {
		t.Run(marker, func(t *testing.T) {
			home := t.TempDir()
			rc := filepath.Join(home, "rc")
			content := "# guard-sh\nsource \"~/.config/guard-sh/" + marker + "\"\nguard-sh on\n"
			if err := os.WriteFile(rc, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if !removeShellIntegration(rc) {
				t.Errorf("expected removal for marker %s", marker)
			}
			got := readFile(t, rc)
			if strings.Contains(got, "guard-sh") {
				t.Errorf("marker %s still present after removal", marker)
			}
		})
	}
}

func TestRunUninstall_CleansRcFiles(t *testing.T) {
	xdg, home := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, "provider_order: []\nproviders: {}\n")

	bashrc := filepath.Join(home, ".bashrc")
	content := "# guard-sh\nsource \"" + dir + "/guard.bash\"\nguard-sh on\n"
	if err := os.WriteFile(bashrc, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	captureStdout(func() { runUninstall(nil) })

	got := readFile(t, bashrc)
	if strings.Contains(got, "guard-sh") {
		t.Errorf(".bashrc not cleaned: %q", got)
	}
}

func TestRunUninstall_Purge_RemovesConfigDir(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, "provider_order: []\nproviders: {}\n")

	captureStdout(func() { runUninstall([]string{"--purge"}) })

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("config dir should be removed with --purge")
	}
}

func TestRunUninstall_NoPurge_KeepsConfigDir(t *testing.T) {
	xdg, _ := setupEnv(t, "/bin/bash")
	dir := xdg + "/guard-sh"
	writeConfig(t, dir, "provider_order: []\nproviders: {}\n")

	captureStdout(func() { runUninstall(nil) })

	if _, err := os.Stat(dir); err != nil {
		t.Error("config dir should be kept without --purge")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Error("config.yaml should be kept without --purge")
	}
}
