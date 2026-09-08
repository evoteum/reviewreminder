// Package config loads reviewreminder's config.yaml.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/evoteum/reviewreminder/internal/paths"
	"gopkg.in/yaml.v3"
)

// Forge describes how to reach the git forge.
type Forge struct {
	Type      string `yaml:"type"`
	URL       string `yaml:"url"`
	Username  string `yaml:"username"`
	TokenFile string `yaml:"token_file"`
}

// Schedule controls automatic runs.
type Schedule struct {
	Enabled    bool   `yaml:"enabled"`
	Expression string `yaml:"expression"`
}

// Config is the parsed config.yaml.
type Config struct {
	Forge         Forge    `yaml:"forge"`
	WebhookURL    string   `yaml:"webhook_url"`
	NudgeInterval string   `yaml:"nudge_interval"`
	Schedule      Schedule `yaml:"schedule"`
}

// Load reads and parses a config file.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &c, nil
}

// Interval is the minimum time between nudges for a pull request; it defaults to
// 24h when unset.
func (c *Config) Interval() (time.Duration, error) {
	s := strings.TrimSpace(c.NudgeInterval)
	if s == "" {
		return 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid nudge_interval %q: %w", s, err)
	}
	return d, nil
}

// Token reads the access token from the configured token_file.
func (c *Config) Token() (string, error) {
	if strings.TrimSpace(c.Forge.TokenFile) == "" {
		return "", fmt.Errorf("forge.token_file is not set")
	}
	path := paths.Expand(c.Forge.TokenFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading token file: %w", err)
	}
	t := strings.TrimSpace(string(b))
	if t == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return t, nil
}
