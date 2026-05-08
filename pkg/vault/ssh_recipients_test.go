package vault

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// testSSHEd25519PubKey generates a fresh SSH ed25519 public key string
// in authorized_keys format (e.g. "ssh-ed25519 AAAA...").
func testSSHEd25519PubKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(priv.Public())
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

// testSSHRSAPubKey generates a fresh SSH RSA public key string
// in authorized_keys format (e.g. "ssh-rsa AAAA...").
func testSSHRSAPubKey(t *testing.T) string {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
}

func TestClassifyKeyType(t *testing.T) {
	ed25519Key := testSSHEd25519PubKey(t)
	rsaKey := testSSHRSAPubKey(t)

	tests := []struct {
		name string
		key  string
		want KeyType
	}{
		{
			name: "age X25519 key",
			key:  "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
			want: "age",
		},
		{
			name: "ssh-ed25519 key",
			key:  ed25519Key,
			want: "ssh-ed25519",
		},
		{
			name: "ssh-rsa key",
			key:  rsaKey,
			want: "ssh-rsa",
		},
		{
			name: "ecdsa key",
			key:  "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=",
			want: "ecdsa-sha2-nistp256",
		},
		{
			name: "unknown key",
			key:  "not-a-real-key",
			want: "",
		},
		{
			name: "empty string",
			key:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyKeyType(tt.key)
			if got != tt.want {
				t.Errorf("ClassifyKeyType(%q) = %v, want %v", truncateKey(tt.key), got, tt.want)
			}
		})
	}
}

func TestParseRecipientKey(t *testing.T) {
	ed25519Key := testSSHEd25519PubKey(t)
	rsaKey := testSSHRSAPubKey(t)

	t.Run("parse age X25519 key", func(t *testing.T) {
		key := "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"
		r, kt, err := ParseRecipientKey(key, false)
		if err != nil {
			t.Fatalf("ParseRecipientKey() error: %v", err)
		}
		if r == nil {
			t.Fatal("ParseRecipientKey() returned nil recipient")
		}
		if kt != "age" {
			t.Errorf("ParseRecipientKey() KeyType = %v, want age", kt)
		}
	})

	t.Run("parse ssh-ed25519 key", func(t *testing.T) {
		r, kt, err := ParseRecipientKey(ed25519Key, false)
		if err != nil {
			t.Fatalf("ParseRecipientKey() error: %v", err)
		}
		if r == nil {
			t.Fatal("ParseRecipientKey() returned nil recipient")
		}
		if kt != "ssh-ed25519" {
			t.Errorf("ParseRecipientKey() KeyType = %v, want ssh-ed25519", kt)
		}
	})

	t.Run("reject ssh-rsa by default", func(t *testing.T) {
		_, _, err := ParseRecipientKey(rsaKey, false)
		if err == nil {
			t.Fatal("ParseRecipientKey() should reject ssh-rsa by default")
		}
		if !strings.Contains(err.Error(), "--allow-rsa") {
			t.Errorf("error should mention --allow-rsa, got: %v", err)
		}
	})

	t.Run("reject ecdsa", func(t *testing.T) {
		_, _, err := ParseRecipientKey("ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=", false)
		if err == nil {
			t.Fatal("ParseRecipientKey() should reject ecdsa")
		}
		if !strings.Contains(err.Error(), "ecdsa") {
			t.Errorf("error should mention ecdsa, got: %v", err)
		}
	})

	t.Run("reject unrecognized format", func(t *testing.T) {
		_, _, err := ParseRecipientKey("not-a-real-key-format", false)
		if err == nil {
			t.Fatal("ParseRecipientKey() should reject unrecognized format")
		}
		if !strings.Contains(err.Error(), "unrecognized recipient format") {
			t.Errorf("error should mention unrecognized, got: %v", err)
		}
	})

	t.Run("trim whitespace", func(t *testing.T) {
		key := "  age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p  \n"
		r, kt, err := ParseRecipientKey(key, false)
		if err != nil {
			t.Fatalf("ParseRecipientKey() error: %v", err)
		}
		if r == nil {
			t.Fatal("ParseRecipientKey() returned nil recipient")
		}
		if kt != "age" {
			t.Errorf("ParseRecipientKey() KeyType = %v, want age", kt)
		}
	})

	t.Run("trim whitespace on ssh-ed25519", func(t *testing.T) {
		key := "  " + ed25519Key + "  \n"
		r, kt, err := ParseRecipientKey(key, false)
		if err != nil {
			t.Fatalf("ParseRecipientKey() error: %v", err)
		}
		if r == nil {
			t.Fatal("ParseRecipientKey() returned nil recipient")
		}
		if kt != "ssh-ed25519" {
			t.Errorf("ParseRecipientKey() KeyType = %v, want ssh-ed25519", kt)
		}
	})
}

func TestParseRecipientKeyAllowRSA(t *testing.T) {
	rsaKey := testSSHRSAPubKey(t)

	t.Run("accept ssh-rsa when allowed", func(t *testing.T) {
		r, kt, err := ParseRecipientKey(rsaKey, true)
		if err != nil {
			t.Fatalf("ParseRecipientKey(allowRSA=true) error: %v", err)
		}
		if r == nil {
			t.Fatal("ParseRecipientKey(allowRSA=true) returned nil recipient")
		}
		if kt != "ssh-rsa" {
			t.Errorf("ParseRecipientKey(allowRSA=true) KeyType = %v, want ssh-rsa", kt)
		}
	})

	t.Run("still reject ecdsa when rsa allowed", func(t *testing.T) {
		_, _, err := ParseRecipientKey("ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY=", true)
		if err == nil {
			t.Fatal("ParseRecipientKey(allowRSA=true) should still reject ecdsa")
		}
	})
}

func TestTruncateKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short key",
			input: "age1abc",
			want:  "age1abc",
		},
		{
			name:  "exactly 40 chars",
			input: strings.Repeat("a", 40),
			want:  strings.Repeat("a", 40),
		},
		{
			name:  "longer than 40 chars",
			input: strings.Repeat("a", 50),
			want:  strings.Repeat("a", 40) + "...",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateKey(tt.input); got != tt.want {
				t.Errorf("truncateKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
