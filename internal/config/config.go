package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds user-tunable defaults.
type Config struct {
	Connections    int    `toml:"connections"`
	Retries        int    `toml:"retries"`
	Dir            string `toml:"dir"`
	Color          bool   `toml:"color"`
	Theme          string `toml:"theme"`
	PackageManager string `toml:"package_manager"`
}

func Defaults() Config {
	return Config{Connections: 8, Retries: 5, Dir: ".", Color: true, Theme: "catppuccin"}
}

// Keys are the settable config keys, in display order.
func Keys() []string {
	return []string{"connections", "retries", "dir", "color", "theme", "package_manager"}
}

// Get returns the string form of a config key.
func (c Config) Get(key string) (string, error) {
	switch key {
	case "connections":
		return strconv.Itoa(c.Connections), nil
	case "retries":
		return strconv.Itoa(c.Retries), nil
	case "dir":
		return c.Dir, nil
	case "color":
		return strconv.FormatBool(c.Color), nil
	case "theme":
		return c.Theme, nil
	case "package_manager":
		return c.PackageManager, nil
	default:
		return "", fmt.Errorf("unknown key %q (known: %s)", key, strings.Join(Keys(), ", "))
	}
}

// Set parses and assigns a config key from its string value.
func (c *Config) Set(key, value string) error {
	switch key {
	case "connections":
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return fmt.Errorf("connections must be a positive integer")
		}
		c.Connections = n
	case "retries":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("retries must be a non-negative integer")
		}
		c.Retries = n
	case "dir":
		c.Dir = value
	case "color":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("color must be true or false")
		}
		c.Color = b
	case "theme":
		c.Theme = value
	case "package_manager":
		c.PackageManager = value
	default:
		return fmt.Errorf("unknown key %q (known: %s)", key, strings.Join(Keys(), ", "))
	}
	return nil
}

// LoadFile reads only the config file (defaults if missing), without env
// overrides or ~ expansion — for round-tripping via `yank config set`.
func LoadFile() (Config, error) {
	c := Defaults()
	if b, err := os.ReadFile(Path()); err == nil {
		if err := toml.Unmarshal(b, &c); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	return c, nil
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

// Save writes c to the standard config path, creating parent directories.
func Save(c Config) error { return saveTo(Path(), c) }

func saveTo(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

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
	c.Dir = ExpandPath(c.Dir) // expand ~ from file/env/default before use
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
	if v := os.Getenv("YANK_THEME"); v != "" {
		c.Theme = v
	}
}
