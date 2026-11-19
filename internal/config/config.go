package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the server's on-disk configuration.
type Config struct {
	Listen      string   `yaml:"listen"`
	HostKeyPath string   `yaml:"host_key_path"`
	AllowCIDRs  []string `yaml:"allow_cidrs"`
	Shell       string   `yaml:"shell"`
	networks    []*net.IPNet
}

// Load parses and validates a YAML configuration file. Relative host-key paths
// are resolved relative to the configuration file.
func Load(path string) (Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Listen == "" || cfg.HostKeyPath == "" {
		return Config{}, fmt.Errorf("listen and host_key_path are required")
	}
	if _, _, err := net.SplitHostPort(cfg.Listen); err != nil {
		return Config{}, fmt.Errorf("invalid listen address %q: %w", cfg.Listen, err)
	}
	if !filepath.IsAbs(cfg.HostKeyPath) {
		cfg.HostKeyPath = filepath.Join(filepath.Dir(path), cfg.HostKeyPath)
	}
	if cfg.Shell == "" {
		cfg.Shell = "/bin/sh"
	}
	for _, raw := range cfg.AllowCIDRs {
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid allow_cidrs entry %q: %w", raw, err)
		}
		cfg.networks = append(cfg.networks, network)
	}
	return cfg, nil
}

// Allows reports whether ip is permitted by the configured source allowlist.
// An empty list permits all sources.
func (c Config) Allows(ip net.IP) bool {
	if len(c.networks) == 0 {
		return true
	}
	for _, network := range c.networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
