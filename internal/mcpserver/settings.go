package mcpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DefaultPort is the localhost port the MCP HTTP transport binds to.
	DefaultPort = 8787
	settingsFile = "mcp.json"
)

// Config is the persisted MCP preference blob (~/.jumpstart/mcp.json).
type Config struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Token   string `json:"token"`
}

var settingsMu sync.Mutex

func settingsPath(dir string) string {
	return filepath.Join(dir, settingsFile)
}

// DefaultConfig returns a disabled server with a fresh token and default port.
func DefaultConfig() Config {
	return Config{
		Enabled: false,
		Port:    DefaultPort,
		Token:   newToken(),
	}
}

// LoadConfig reads MCP settings. A missing file is treated as disabled with
// a newly generated token so the first enable has something to show.
func LoadConfig(dir string) Config {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return loadConfigLocked(dir)
}

func loadConfigLocked(dir string) Config {
	data, err := os.ReadFile(settingsPath(dir))
	if err != nil {
		return DefaultConfig()
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return DefaultConfig()
	}
	return normalize(c)
}

// SaveConfig persists MCP settings.
func SaveConfig(dir string, c Config) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	c = normalize(c)
	if c.Token == "" {
		c.Token = newToken()
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := settingsPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, settingsPath(dir))
}

func normalize(c Config) Config {
	if c.Port <= 0 || c.Port > 65535 {
		c.Port = DefaultPort
	}
	return c
}

func newToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("js-%d", os.Getpid())
	}
	return hex.EncodeToString(b[:])
}

// RotateToken replaces the bearer token and saves the config.
func RotateToken(dir string) (Config, error) {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	c := loadConfigLocked(dir)
	c.Token = newToken()
	data, err := json.MarshalIndent(normalize(c), "", "  ")
	if err != nil {
		return c, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return c, err
	}
	tmp := settingsPath(dir) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return c, err
	}
	if err := os.Rename(tmp, settingsPath(dir)); err != nil {
		return c, err
	}
	return c, nil
}
