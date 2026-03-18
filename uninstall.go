package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Berdan/guard-sh/internal/config"
)

func runUninstall(args []string) {
	purge := false
	for _, arg := range args {
		if arg == "--purge" {
			purge = true
		}
	}

	dir := config.Dir()

	fmt.Printf("  %sguard-sh uninstall%s\n\n", bold+cyan, reset)

	// --- Shell rc files ---
	home, _ := os.UserHomeDir()
	xdgCfg := os.Getenv("XDG_CONFIG_HOME")
	if xdgCfg == "" {
		xdgCfg = filepath.Join(home, ".config")
	}
	rcFiles := []struct{ name, path string }{
		{"bash", filepath.Join(home, ".bashrc")},
		{"zsh", filepath.Join(home, ".zshrc")},
		{"fish", filepath.Join(xdgCfg, "fish", "config.fish")},
	}
	for _, rc := range rcFiles {
		if removeShellIntegration(rc.path) {
			fmt.Printf("  rc     %s%s (cleaned)%s\n", dim, rc.path, reset)
		} else {
			fmt.Printf("  rc     %s%s (not present)%s\n", dim, rc.path, reset)
		}
	}

	fmt.Println()

	// --- Shell scripts ---
	for _, name := range []string{"guard.bash", "guard.zsh", "guard.fish"} {
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err == nil {
			fmt.Printf("  shell  %s%s%s\n", dim, path, reset)
		}
	}

	// --- Config dir (--purge only) ---
	if purge {
		fmt.Println()
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "guard-sh: could not remove %s: %v\n", dir, reset)
		} else {
			fmt.Printf("  config %s%s (removed)%s\n", dim, dir, reset)
		}
	}

	// --- Binary removal hint ---
	fmt.Println()
	bin, _ := os.Executable()
	fmt.Printf("  %snext%s  remove the binary manually:\n", bold, reset)
	fmt.Printf("  %s      rm %s%s\n\n", dim, bin, reset)
}

// removeShellIntegration strips guard-sh integration lines from an rc file.
// Returns true if any lines were removed.
func removeShellIntegration(rcFile string) bool {
	data, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	filtered := lines[:0]
	removed := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "# guard-sh" ||
			strings.Contains(line, "guard.bash") ||
			strings.Contains(line, "guard.zsh") ||
			strings.Contains(line, "guard.fish") ||
			trimmed == "guard-sh on" {
			removed = true
			continue
		}
		filtered = append(filtered, line)
	}

	if !removed {
		return false
	}

	// Collapse consecutive blank lines left behind
	collapsed := filtered[:0]
	prevBlank := false
	for _, line := range filtered {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue
		}
		collapsed = append(collapsed, line)
		prevBlank = isBlank
	}

	return os.WriteFile(rcFile, []byte(strings.Join(collapsed, "\n")), 0644) == nil
}
