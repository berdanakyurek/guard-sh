package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Berdan/guard-sh/internal/config"
)

func runSetup() {
	dir := config.Dir()

	fmt.Printf("  %sguard-sh setup%s\n\n", bold+cyan, reset)

	// --- Config dir ---
	if err := os.MkdirAll(dir, 0700); err != nil {
		fatalf("could not create config dir: %v", err)
	}

	// --- config.yaml ---
	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
			fatalf("could not write config: %v", err)
		}
		fmt.Printf("  config    %s%s%s\n", dim, configPath, reset)
	} else {
		fmt.Printf("  config    %s%s (already exists, skipped)%s\n", dim, configPath, reset)
	}

	// --- prompt.txt ---
	promptPath := filepath.Join(dir, "prompt.txt")
	if _, err := os.Stat(promptPath); os.IsNotExist(err) {
		if err := os.WriteFile(promptPath, []byte(defaultPrompt), 0600); err != nil {
			fatalf("could not write prompt: %v", err)
		}
		fmt.Printf("  prompt    %s%s%s\n", dim, promptPath, reset)
	} else {
		fmt.Printf("  prompt    %s%s (already exists, skipped)%s\n", dim, promptPath, reset)
	}

	// --- Shell scripts ---
	bashScript := filepath.Join(dir, "guard.bash")
	if err := os.WriteFile(bashScript, []byte(shellBash), 0644); err != nil {
		fatalf("could not write guard.bash: %v", err)
	}
	fmt.Printf("  shell     %s%s%s\n", dim, bashScript, reset)

	zshScript := filepath.Join(dir, "guard.zsh")
	if err := os.WriteFile(zshScript, []byte(shellZsh), 0644); err != nil {
		fatalf("could not write guard.zsh: %v", err)
	}
	fmt.Printf("  shell     %s%s%s\n", dim, zshScript, reset)

	// --- Shell integration ---
	fmt.Println()
	shellName := filepath.Base(os.Getenv("SHELL"))
	var rcFile, scriptPath string
	switch shellName {
	case "zsh":
		rcFile = filepath.Join(os.Getenv("HOME"), ".zshrc")
		scriptPath = zshScript
	case "bash":
		rcFile = filepath.Join(os.Getenv("HOME"), ".bashrc")
		scriptPath = bashScript
	default:
		fmt.Printf("  %sUnsupported shell: %s%s\n", dim, shellName, reset)
		fmt.Printf("  %sManually source the appropriate file from %s%s\n", dim, dir, reset)
		printNext(configPath)
		return
	}

	sourceLine := `source "` + scriptPath + `"`
	onLine := "guard-sh on"

	rcData, _ := os.ReadFile(rcFile)
	rc := string(rcData)

	if strings.Contains(rc, sourceLine) {
		if !strings.Contains(rc, onLine) {
			f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fatalf("could not update %s: %v", rcFile, err)
			}
			fmt.Fprintf(f, "\n%s\n", onLine)
			f.Close()
			fmt.Printf("  rc        %s%s (updated)%s\n", dim, rcFile, reset)
		} else {
			fmt.Printf("  rc        %s%s (already present)%s\n", dim, rcFile, reset)
		}
	} else {
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fatalf("could not update %s: %v", rcFile, err)
		}
		fmt.Fprintf(f, "\n# guard-sh\n%s\n%s\n", sourceLine, onLine)
		f.Close()
		fmt.Printf("  rc        %s%s%s\n", dim, rcFile, reset)
	}

	fmt.Println()
	printNext(configPath)
}

func printNext(configPath string) {
	fmt.Printf("  %snext%s  edit %s and set your api_key\n", bold, reset, configPath)
	fmt.Printf("  %s      then restart your shell or run: source ~/.bashrc%s\n\n", dim, reset)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "guard-sh: "+format+"\n", args...)
	os.Exit(1)
}
