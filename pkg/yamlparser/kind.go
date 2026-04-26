// Package yamlparser handles YAML configuration parsing for hulak.
package yamlparser

import (
	"fmt"
	"strings"

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
	Kind Kind `json:"kind,omitempty" yaml:"kind,omitempty"`
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

	return &cfg, nil
}
