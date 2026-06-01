package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Config holds user-tunable defaults.
type Config struct {
	Connections int    `toml:"connections"`
	Retries     int    `toml:"retries"`
	Dir         string `toml:"dir"`
	LimitRate   string `toml:"limit_rate"`
	Color       bool   `toml:"color"`
}

func Defaults() Config {
	return Config{Connections: 8, Retries: 5, Dir: ".", Color: true}
}

// Path returns the config file path honoring XDG_CONFIG_HOME.
func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "yank", "config.toml")
}

// Load reads the standard config path; missing file yields defaults.
func Load() (Config, error) { return loadFrom(Path()) }

func loadFrom(path string) (Config, error) {
	c := Defaults()
	if b, err := os.ReadFile(path); err == nil {
		if err := toml.Unmarshal(b, &c); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	applyEnv(&c)
	return c, nil
}

func applyEnv(c *Config) {
	if v := os.Getenv("YANK_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Connections = n
		}
	}
	if v := os.Getenv("YANK_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Retries = n
		}
	}
	if v := os.Getenv("YANK_DIR"); v != "" {
		c.Dir = v
	}
}
