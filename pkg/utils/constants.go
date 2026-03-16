package utils

import "io/fs"

// CLI colors — bright ANSI (90-97) for direct fmt.Printf output.
// These match the semantic ANSI colors in tui/styles.go:
//
//	Red    = bright 1 (ColorError)
//	Green  = bright 2 (ColorSuccess)
//	Yellow = bright 3 (ColorWarn)
//	Blue   = bright 4 (ColorPrimary)
//
// Terminal themes remap these automatically for light/dark backgrounds.
const (
	Red        = "\033[91m"
	Green      = "\033[92m"
	Yellow     = "\033[93m"
	Blue       = "\033[94m"
	ColorReset = "\033[0m"
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
const APIOptions = "apiOptions.hk.yaml"

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
	DirPer    fs.FileMode = 0o755
	FilePer   fs.FileMode = 0o644
	SecretPer fs.FileMode = 0o600
)

// tick mark and x for success and failure
const (
	CheckMark           = "\u2713"  // tick
	CrossMark           = "\u2717"  // x
	ChevronRight        = "\uf054 " // >
	ChevronRightCircled = "\uf138"
	ChevronDownCircled  = "\uf13a"
	Asterisk            = "\uf069"
	Connector           = "└─"
	ConnectorVertical   = "|"
)

const Ellipsis = "..."
