package cache

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCache_GetSet(t *testing.T) {
	dir := t.TempDir()
	c := Load(dir, 10)

	c.Set("rm -rf /", "Deletes everything")
	got, ok := c.Get("rm -rf /")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got != "Deletes everything" {
		t.Errorf("got %q, want %q", got, "Deletes everything")
	}
}

func TestCache_Miss(t *testing.T) {
	c := Load(t.TempDir(), 10)
	_, ok := c.Get("unknown command")
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestCache_NilWhenDisabled(t *testing.T) {
	c := Load(t.TempDir(), 0)
	if c != nil {
		t.Error("expected nil cache when maxSize=0")
	}
}

func TestCache_Eviction(t *testing.T) {
	dir := t.TempDir()
	c := Load(dir, 2)

	c.Set("cmd1", "r1")
	c.Set("cmd2", "r2")
	c.Set("cmd3", "r3") // should evict the oldest

	count := 0
	for _, key := range []string{"cmd1", "cmd2", "cmd3"} {
		if _, ok := c.Get(key); ok {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 entries after eviction, got %d", count)
	}
}

func TestCache_Persist(t *testing.T) {
	dir := t.TempDir()

	c1 := Load(dir, 10)
	c1.Set("ls -la", "OK")

	// reload from disk
	c2 := Load(dir, 10)
	got, ok := c2.Get("ls -la")
	if !ok {
		t.Fatal("expected cache hit after reload, got miss")
	}
	if got != "OK" {
		t.Errorf("got %q, want %q", got, "OK")
	}
}

func TestCache_PersistToDisk(t *testing.T) {
	dir := t.TempDir()
	c := Load(dir, 10)
	c.Set("git push --force", "Force pushes to remote")

	data, err := os.ReadFile(dir + "/cache.json")
	if err != nil {
		t.Fatalf("cache file not written: %v", err)
	}
	var entries map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("invalid JSON in cache file: %v", err)
	}
	if _, ok := entries["git push --force"]; !ok {
		t.Error("expected key in cache file, not found")
	}
}
