package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onceprgm/cme/internal/store"
)

const currentVersion = 1

type Config struct {
	ConfigVersion int      `json:"configVersion"`
	Username      string   `json:"username,omitempty"`
	RAM           int      `json:"ram,omitempty"`
	JavaPath      string   `json:"javaPath,omitempty"`
	JVMArgs       []string `json:"jvmArgs,omitempty"`
}

func path() string {
	return filepath.Join(store.ConfigDir(), "config.json")
}

func Load() (*Config, error) {
	raw, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{ConfigVersion: currentVersion}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if c.ConfigVersion == 0 {
		c.ConfigVersion = currentVersion
	}
	return &c, nil
}

func Save(c *Config) error {
	c.ConfigVersion = currentVersion
	if err := store.Ensure(store.ConfigDir()); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	dst := path()
	tmp := dst + ".part"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func (c *Config) AddJVMArg(arg string) bool {
	if strings.TrimSpace(arg) == "" {
		return false
	}
	for _, a := range c.JVMArgs {
		if a == arg {
			return false
		}
	}
	c.JVMArgs = append(c.JVMArgs, arg)
	return true
}

func (c *Config) RemoveJVMArg(arg string) bool {
	for i, a := range c.JVMArgs {
		if a == arg {
			c.JVMArgs = append(c.JVMArgs[:i], c.JVMArgs[i+1:]...)
			return true
		}
	}
	return false
}
