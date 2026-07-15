// Package yamlparser handles YAML configuration parsing for hulak.
package yamlparser

import (
	"fmt"
	"os"
	"strings"
	"time"

	yaml "github.com/goccy/go-yaml"
)

// Kind represents the type of YAML flow hulak should follow.
type Kind string

// Allowed configuration kinds.
const (
	KindAuth    Kind = "Auth"
	KindAPI     Kind = "API"
	KindGraphQL Kind = "GraphQL"
)

// Holds the registered kinds and default selection logic.
type kindRegistry struct {
	validKinds  map[string]Kind
	defaultKind Kind
}

func newKindRegistry() *kindRegistry {
	r := &kindRegistry{
		validKinds:  make(map[string]Kind),
		defaultKind: KindAPI, // default kind
	}
	r.register(KindAuth)
	r.register(KindAPI)
	r.register(KindGraphQL)
	return r
}

func (r *kindRegistry) register(k Kind) {
	r.validKinds[strings.ToLower(string(k))] = k
}

var registry = newKindRegistry()

// ConfigType is the root YAML configuration structure.
type ConfigType struct {
	Kind Kind `json:"kind,omitempty"    yaml:"kind,omitempty"`
	// Timeout is the optional per-request timeout, e.g. "5m", "90s", "2m30s".
	// Parsed via time.ParseDuration; see ParsedTimeout for resolution. Empty
	// string falls through to the runner's flag/env/default chain.
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// ParsedTimeout returns the configured per-request timeout, or 0 if unset.
// Returns an error when Timeout is set but not a valid positive Go duration —
// callers should surface this as a config error so the user fixes the YAML
// instead of silently falling back to the default.
func (c *ConfigType) ParsedTimeout() (time.Duration, error) {
	if c == nil || c.Timeout == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout %q: %w", c.Timeout, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("timeout must be positive, got %q", c.Timeout)
	}
	return d, nil
}

// PeekConfig reads a request file's top-level config (kind, timeout) without
// resolving templates or secrets. Other request fields (url, body, ...) are
// ignored, so a file with unresolved template vars still peeks cleanly. Kind
// is normalized (defaulting to API). Use it to classify or time a file
// without a full parse.
func PeekConfig(filePath string) (*ConfigType, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cfg ConfigType
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}
	cfg.Kind = cfg.Kind.normalize()
	return &cfg, nil
}

// PeekKind reads only the `kind` field from a request file (see PeekConfig).
func PeekKind(filePath string) (Kind, error) {
	cfg, err := PeekConfig(filePath)
	if err != nil {
		return "", err
	}
	return cfg.Kind, nil
}

// normalize resolves case insensitivity and defaulting.
func (k *Kind) normalize() Kind {
	if k == nil || *k == "" {
		return registry.defaultKind
	}

	key := strings.ToLower(string(*k))
	if canonical, ok := registry.validKinds[key]; ok {
		return canonical
	}

	return Kind(key)
}

// getKind returns the normalized Kind.
func (c *ConfigType) getKind() Kind {
	return c.Kind.normalize()
}

// IsAuth returns true when the configuration kind is "Auth".
func (c *ConfigType) IsAuth() bool {
	return strings.EqualFold(string(c.getKind()), string(KindAuth))
}

// IsAPI returns true when the configuration kind is "API".
func (c *ConfigType) IsAPI() bool {
	return strings.EqualFold(string(c.getKind()), string(KindAPI))
}

// IsGraphQL returns true when the configuration kind is "GraphQL".
func (c *ConfigType) IsGraphql() bool {
	return strings.EqualFold(string(c.getKind()), string(KindGraphQL))
}

// ParseConfig parses a YAML file into ConfigType.
func ParseConfig(filePath string, secretsMap map[string]any) (*ConfigType, error) {
	// checkYamlFile errors already carry the file path; don't wrap with a
	// generic "error reading YAML file" prefix that just adds noise.
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return nil, err
	}

	var cfg ConfigType
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", filePath, err)
	}

	cfg.Kind = cfg.Kind.normalize()

	// Validate the timeout eagerly so a malformed value (e.g. `timeout: 60`
	// with no unit) fails the file with a clear config error instead of
	// silently falling back to the default.
	if _, err := cfg.ParsedTimeout(); err != nil {
		return nil, fmt.Errorf("in %s: %w", filePath, err)
	}

	return &cfg, nil
}
