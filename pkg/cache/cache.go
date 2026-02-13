package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"inspectgo/pkg/core"
)

const defaultTTL = 7 * 24 * time.Hour

type Cache struct {
	Dir string
	TTL time.Duration
}

func New(dir string, ttl time.Duration) (*Cache, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".inspectgo", "cache")
	}
	if ttl <= 0 {
		ttl = defaultTTL
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Cache{Dir: dir, TTL: ttl}, nil
}

type cacheEntry struct {
	Response   core.Response `json:"response"`
	CachedAt   time.Time     `json:"cached_at"`
	ModelName  string        `json:"model_name"`
}

func key(modelName, prompt string, opts core.GenerateOptions) string {
	parts := []string{
		modelName,
		prompt,
		opts.SystemPrompt,
		fmt.Sprintf("%.6f", opts.Temperature),
		fmt.Sprintf("%d", opts.MaxTokens),
		fmt.Sprintf("%.6f", opts.TopP),
	}
	if len(opts.Stop) > 0 {
		parts = append(parts, strings.Join(opts.Stop, "|"))
	}
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:])
}

func (c *Cache) path(k string) string {
	return filepath.Join(c.Dir, k+".json.gz")
}

func (c *Cache) Get(modelName, prompt string, opts core.GenerateOptions) (core.Response, bool) {
	k := key(modelName, prompt, opts)
	p := c.path(k)
	f, err := os.Open(p)
	if err != nil {
		return core.Response{}, false
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return core.Response{}, false
	}
	defer gz.Close()
	var e cacheEntry
	if err := json.NewDecoder(gz).Decode(&e); err != nil {
		return core.Response{}, false
	}
	if c.TTL > 0 && time.Since(e.CachedAt) > c.TTL {
		_ = os.Remove(p)
		return core.Response{}, false
	}
	return e.Response, true
}

func (c *Cache) Set(modelName, prompt string, opts core.GenerateOptions, resp core.Response) error {
	k := key(modelName, prompt, opts)
	p := c.path(k)
	e := cacheEntry{Response: resp, CachedAt: time.Now(), ModelName: modelName}
	f, err := os.CreateTemp(c.Dir, "tmp-*.json.gz")
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	if err := json.NewEncoder(gz).Encode(e); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := gz.Close(); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return err
	}
	if err := os.Rename(f.Name(), p); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}
