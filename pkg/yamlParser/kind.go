package yamlParser

import "strings"

// kind field in the yaml file. Depending on the kind, we jump to different flow
type Kind string

const (
	KindAuth Kind = "Auth"
	KindAPI  Kind = "Api"
)

var validKinds = [2]Kind{KindAuth, KindAPI}

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

// checks if the lowercased kind equals to provided
func (k *Kind) normalize() Kind {
	// if empty, default to API
	if *k == "" {
		return KindAPI
	}
	strKind := strings.ToLower(string(*k))
	switch strKind {
	case strings.ToLower(string(KindAPI)):
		return KindAPI
	case strings.ToLower(string(KindAuth)):
		return KindAuth
	default:
		return Kind(strKind)
	}
}

// Defines Kind, which represents the purpose of the file
type ConfigType struct {
	Kind Kind `json:"kind,omitempty" yaml:"kind,omitempty"`
}

// checks if the kind is valid according to the defined rules
func (conf *ConfigType) IsValid() bool {
	normalizedKind := conf.Kind.normalize()

	// Default to Api even if config has no kind field
	if conf.Kind == "" {
		return true
	}

	// check agains valid Kinds (case-insensitive)
	for _, validKind := range validKinds {
		if strings.EqualFold(string(normalizedKind), string(validKind)) {
			return true
		}
	}

	return false
}

// returns the normalized kind, defauling to API if not specified
func (conf *ConfigType) GetKind() Kind {
	return conf.Kind.normalize()
}

// IsAuth checks if the kind is Auth (case-insensitive)
func (conf *ConfigType) IsAuth() bool {
	return strings.EqualFold(string(conf.GetKind()), string(KindAuth))
}

// IsAPI checks if the kind is API (case-insensitive)
func (conf *ConfigType) IsAPI() bool {
	return strings.EqualFold(string(conf.GetKind()), string(KindAPI))
}
