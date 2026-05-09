package vault

const (
	AgePrefix = "age1"
	Age       = "age"

	// Default SSH key location components — joined with filepath.Join at runtime
	// for cross-platform path separators.
	sshKeyDir  = ".ssh"
	sshKeyFile = "id_ed25519"
)
