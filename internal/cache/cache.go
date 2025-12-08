package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Entry struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type Cache struct {
	Entries []Entry `json:"entries"`
	path    string
	maxSize int
}

const DefaultMaxSize = 5 * 1024 * 1024 // 5MB

func CacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hnk")
}

func New(maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}

	dir := CacheDir()
	if dir == "" {
		return &Cache{maxSize: maxSize}
	}

	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "cache.json")

	c := &Cache{
		path:    path,
		maxSize: maxSize,
	}
	c.load()
	return c
}

func (c *Cache) load() {
	if c.path == "" {
		return
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}

	json.Unmarshal(data, c)
}

func (c *Cache) save() {
	if c.path == "" {
		return
	}

	data, _ := json.Marshal(c)
	os.WriteFile(c.path, data, 0644)
}

func HashKey(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:16])
}

func (c *Cache) Get(key string) (string, bool) {
	for _, e := range c.Entries {
		if e.Key == key {
			return e.Value, true
		}
	}
	return "", false
}

func (c *Cache) Set(key, value string) {
	size := len(key) + len(value)

	for i, e := range c.Entries {
		if e.Key == key {
			c.Entries = append(c.Entries[:i], c.Entries[i+1:]...)
			break
		}
	}

	c.Entries = append(c.Entries, Entry{
		Key:       key,
		Value:     value,
		Size:      size,
		CreatedAt: time.Now(),
	})

	c.evict()
	c.save()
}

func (c *Cache) evict() {
	sort.Slice(c.Entries, func(i, j int) bool {
		return c.Entries[i].CreatedAt.Before(c.Entries[j].CreatedAt)
	})

	total := 0
	for _, e := range c.Entries {
		total += e.Size
	}

	for total > c.maxSize && len(c.Entries) > 0 {
		total -= c.Entries[0].Size
		c.Entries = c.Entries[1:]
	}
}

func (c *Cache) Clear() {
	c.Entries = nil
	c.save()
}

func (c *Cache) TotalSize() int {
	total := 0
	for _, e := range c.Entries {
		total += e.Size
	}
	return total
}
