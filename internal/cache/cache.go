package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type entry struct {
	Response string `json:"response"`
	Ts       int64  `json:"ts"`
}

type Cache struct {
	path    string
	maxSize int
	entries map[string]entry
}

func Load(dir string, maxSize int) *Cache {
	if maxSize <= 0 {
		return nil
	}
	c := &Cache{
		path:    filepath.Join(dir, "cache.json"),
		maxSize: maxSize,
		entries: make(map[string]entry),
	}
	data, err := os.ReadFile(c.path)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c.entries)
	c.evict()
	return c
}

func (c *Cache) Get(cmd string) (string, bool) {
	e, ok := c.entries[cmd]
	if !ok {
		return "", false
	}
	e.Ts = time.Now().Unix()
	c.entries[cmd] = e
	c.save()
	return e.Response, true
}

func (c *Cache) Set(cmd, response string) {
	c.entries[cmd] = entry{Response: response, Ts: time.Now().Unix()}
	c.save()
}

func (c *Cache) evict() {
	for len(c.entries) > c.maxSize {
		oldest := int64(^uint64(0) >> 1) // max int64
		oldestCmd := ""
		for cmd, e := range c.entries {
			if e.Ts < oldest {
				oldest = e.Ts
				oldestCmd = cmd
			}
		}
		delete(c.entries, oldestCmd)
	}
	c.save()
}

func (c *Cache) save() {
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.path, data, 0600)
}
