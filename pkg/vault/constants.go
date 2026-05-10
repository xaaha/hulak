package vault

const (
	AgePrefix = "age1"
	Age       = "age"

	// SSH key type strings as returned by the ssh library / authorized_keys format.
	sshEd25519 = "ssh-ed25519"
	sshRSA     = "ssh-rsa"

	// Default SSH key location components — joined with filepath.Join at runtime
	// for cross-platform path separators.
	sshKeyDir  = ".ssh"
	sshKeyFile = "id_ed25519"
)
