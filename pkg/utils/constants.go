package utils

import (
	"io/fs"
)

const (
	ProjectName       = "hulak"
	HiddenProjectName = "." + ProjectName
)

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

const (
	StoreFile      = "store.age"
	RecipientsFile = "recipients.txt"
	IdentityFile   = "identity.txt"

	// MasterKey is the environment variable that overrides the on-disk identity.
	// Intended for CI where the identity file isn't practical.
	MasterKey = "HULAK_MASTER_KEY"
)

// SSH identity support
const (
	SSHIdentityEnvVar = "HULAK_SSH_IDENTITY"
	SSHKeyDir         = ".ssh"
	SSHKeyFile        = "id_ed25519"
)

// Editor is the fallback editor used when $EDITOR is unset. POSIX guarantees
// `vi`, so this works in bare/minimal environments (Alpine, distroless, etc.)
// where vim/nano may not be installed. Users who prefer something else should
// set $EDITOR in their shell.
const Editor = "vi"

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
	DirPer fs.FileMode = 0o755
	// SecretDirPer is owner-only on the directory holding secret material
	// (e.g. ~/.config/hulak that contains identity.txt). Group/other have
	// no access — defense-in-depth so the directory listing alone can't leak
	// the existence of an identity file to other users on a shared host.
	SecretDirPer fs.FileMode = 0o700
	FilePer      fs.FileMode = 0o644
	SecretPer    fs.FileMode = 0o600
)

// tick mark and x for success and failure
const (
	CheckMark           = "\u2714"  // tick
	CrossMark           = "\u2716"  // x
	ChevronRight        = "\uf054 " // >
	ChevronRightCircled = "\uf138"
	ChevronDownCircled  = "\uf13a"
	Asterisk            = "\uf069"
	Connector           = "└─"
	ConnectorVertical   = "|"
)

const (
	Ellipsis = "..."
	Comment  = "#"
)
