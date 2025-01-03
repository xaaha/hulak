package utils

// Colors
const (
	Red        = "\033[31;1m"
	Green      = "\033[32;1m"
	Yellow     = "\033[33;1m"
	Grey       = "\033[90;1m"
	ColorReset = "\033[0m"
)

// Environment
const (
	EnvKey               = "hulakEnv"
	DefaultEnvVal        = "global"
	DefaultEnvFileSuffix = ".env"
)

// Errors message
const (
	UnResolvedVariable = "unresolved variable "
	EmptyVariables     = "variable string can't be empty"
)

// acceptable file patterns
const (
	YAML = ".yaml"
	YML  = ".yml"
)
