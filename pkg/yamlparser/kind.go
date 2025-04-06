package yamlparser

import (
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
)

// Kind represents the type of yaml flow hulak should follow
type Kind string

// Available configuration kinds
const (
	KindAuth Kind = "Auth"
	KindAPI  Kind = "API"
)

// KindConfig holds configuration for handling different kinds
type KindConfig struct {
	// Map of normalized (lowercase) kind names to their canonical forms
	validKinds map[string]Kind
	// Default kind to use when none is specified
	defaultKind Kind
}

// Global instance of KindConfig
var kindConfig = newKindConfig()

// newKindConfig initializes the kind configuration
func newKindConfig() *KindConfig {
	kc := &KindConfig{
		validKinds:  make(map[string]Kind),
		defaultKind: KindAPI, // Set API as default
	}

	// Register default kinds
	kc.registerKind(KindAuth)
	kc.registerKind(KindAPI)

	return kc
}

// registerKind adds a new kind to the valid kinds map
func (kc *KindConfig) registerKind(k Kind) {
	kc.validKinds[strings.ToLower(string(k))] = k
}

// GetValidKinds returns a slice of all valid kinds
func (kc *KindConfig) GetValidKinds() []Kind {
	kinds := make([]Kind, 0, len(kc.validKinds))
	for _, k := range kc.validKinds {
		kinds = append(kinds, k)
	}
	return kinds
}

// ConfigType represents the configuration structure
type ConfigType struct {
	Kind Kind `json:"kind,omitempty" yaml:"kind,omitempty"`
}

// normalize standardizes the kind value
func (k *Kind) normalize() Kind {
	if k == nil || *k == "" {
		return kindConfig.defaultKind
	}

	normalized := strings.ToLower(string(*k))
	if canonical, exists := kindConfig.validKinds[normalized]; exists {
		return canonical
	}
	return Kind(normalized)
}

// IsValid checks if the kind is valid according to defined rules
func (conf *ConfigType) IsValid() bool {
	// empty mean  API
	if conf.Kind == "" {
		return true
	}

	normalized := strings.ToLower(string(conf.GetKind()))
	_, exists := kindConfig.validKinds[normalized]
	return exists
}

// GetKind returns the normalized kind
func (conf *ConfigType) GetKind() Kind {
	return conf.Kind.normalize()
}

// IsKind checks if the configuration is of a specific kind
func (conf *ConfigType) IsKind(k Kind) bool {
	return strings.EqualFold(string(conf.GetKind()), string(k))
}

// IsAuth checks if the kind is Auth
func (conf *ConfigType) IsAuth() bool {
	return conf.IsKind(KindAuth)
}

// IsAPI checks if the kind is API
func (conf *ConfigType) IsAPI() bool {
	return conf.IsKind(KindAPI)
}

// ValidateKinds checks if all kinds in the slice are valid
func ValidateKinds(kinds []Kind) ([]string, bool) {
	var invalidKinds []string
	isValid := true

	for _, kind := range kinds {
		config := ConfigType{Kind: kind}
		if !config.IsValid() {
			invalidKinds = append(invalidKinds, string(kind))
			isValid = false
		}
	}

	return invalidKinds, isValid
}

// parses a YAML file and returns the configuration type
func ParseConfig(filePath string, secretsMap map[string]any) (*ConfigType, error) {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		return nil, utils.ColorError("error reading YAML file: %w", err)
	}

	var config ConfigType
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&config); err != nil {
		return nil, utils.ColorError("error decoding YAML: %w", err)
	}

	return &config, nil
}

// MustParseConfig parses a YAML file and panics on error
func MustParseConfig(filePath string, secretsMap map[string]any) ConfigType {
	config, err := ParseConfig(filePath, secretsMap)
	if err != nil {
		utils.PanicRedAndExit("#kind.go: %v", err)
	}
	return *config
}
