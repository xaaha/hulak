package utils

import "io/fs"

// Colors
const (
	Red        = "\033[91m"
	Green      = "\033[92m"
	Yellow     = "\033[93m"
	Grey       = "\033[90m"
	ColorReset = "\033[0m"
	Blue       = "\033[94m"
)

// Environment
const (
	EnvironmentFolder    = "env"
	EnvKey               = "hulakEnv"
	DefaultEnvVal        = "global"
	DefaultEnvFileSuffix = ".env"
)

// Errors message
const (
	UnResolvedVariable = "unresolved variable "
	EmptyVariables     = "variable string can't be empty"
	IndexOutOfBounds   = "array index out of bounds: "
	KeyNotFound        = "key not found: "
)

// acceptable file patterns
const (
	YAML = ".yaml"
	YML  = ".yml"
	JSON = ".json"
)

// example file with all options
const ApiOptions = "apiOptions.hk.yaml"

// response pattern for files saved
const (
	ResponseBase       = "_response"
	ResponseFileSuffix = ".json"
	ResponseFileName   = ResponseBase + ResponseFileSuffix
)

// JSON supported types
const (
	JSONString = "string"
	JSONNumber = "number"
	JSONInt    = "int"
	JSONBool   = "bool"
	JSONNull   = "null"
)

// ResponseType is Auth2.0 ResponseType
const ResponseType = "code"

// Permissions for creating directory and files
const (
	DirPer  fs.FileMode = 0o755
	FilePer fs.FileMode = 0o644
)

// tick mark and x for success and failure
const (
	CheckMark = "\u2713"
	CrossMark = "\u2717"
)
