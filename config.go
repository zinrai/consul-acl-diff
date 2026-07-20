package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// LoadConfig reads and validates the YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func validate(cfg *Config) error {
	names := make(map[string]bool)
	for _, p := range cfg.Policies {
		if p.Name == "" {
			return fmt.Errorf("policy name cannot be empty")
		}
		if names[p.Name] {
			return fmt.Errorf("duplicate policy name: %s", p.Name)
		}
		names[p.Name] = true
	}

	accessors := make(map[string]bool)
	for i, t := range cfg.Tokens {
		if t.AccessorID == "" {
			return fmt.Errorf("token #%d (%q) has no accessor_id, which is its identity key", i+1, t.Description)
		}
		if accessors[t.AccessorID] {
			return fmt.Errorf("duplicate token accessor_id: %s", t.AccessorID)
		}
		accessors[t.AccessorID] = true
	}
	return nil
}

// normalizeRules mirrors consul-acl-sync so rule comparison ignores cosmetic
// whitespace differences rather than reporting false drift.
func normalizeRules(rules string) string {
	rules = strings.TrimSpace(rules)
	rules = strings.ReplaceAll(rules, "\r\n", "\n")
	lines := strings.Split(rules, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
