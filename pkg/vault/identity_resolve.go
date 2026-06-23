package vault

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains the multi-source identity probe: gathering every configured
// identity (master-key env, identity.txt, SSH env, default SSH key), resolving
// which one to use for read/write paths, and the error helpers that go with
// those flows. Identity file CRUD lives in keys.go; this file is policy.

// sourceKind discriminates configured identity sources so callers can apply
// policy (e.g. hard-fail on broken explicit config, silent fall-through on
// auto-detected sources) without string-matching on labels.
type sourceKind int

const (
	sourceMasterKey    sourceKind = iota // $HULAK_MASTER_KEY (strict, no fallback)
	sourceIdentityFile                   // identity.txt on disk
	sourceSSHEnv                         // $HULAK_SSH_IDENTITY
	sourceSSHDefault                     // auto-detected ~/.ssh/id_ed25519
)

// identitySource describes one configured identity. Powers every "what
// identities does this machine have?" query (probe loop, list-identity,
// doctor). loadErr is set when the source is configured but failed to parse;
// callers decide whether to hard-fail or skip past it based on kind.
type identitySource struct {
	kind      sourceKind
	label     string       // human-readable diagnostic label
	path      string       // file path or "$ENV_VAR" — for list-identity PATH column
	identity  age.Identity // nil when loadErr is set
	publicKey string       // derived pubkey: "age1..." or "ssh-ed25519 AAA..."
	loadErr   error
}

// gatherSources collects every configured identity in precedence order.
// includeMaster gates HULAK_MASTER_KEY: include it for inspection (list-
// identity, SourcesThatDecrypt); omit it when callers handle master-key's
// strict short-circuit semantics before calling (ResolveIdentity, resolveAndDecrypt).
func gatherSources(includeMaster bool) []identitySource {
	var sources []identitySource

	if includeMaster {
		if raw := strings.TrimSpace(os.Getenv(utils.MasterKey)); raw != "" {
			if id, err := parseMasterKey(raw); err == nil {
				sources = append(sources, identitySource{
					kind:      sourceMasterKey,
					label:     "$" + utils.MasterKey,
					path:      "$" + utils.MasterKey,
					identity:  id,
					publicKey: id.Recipient().String(),
				})
			}
		}
	}

	if IdentityExists() {
		idPath, _ := IdentityPath()
		id, err := LoadIdentity()
		s := identitySource{
			kind:     sourceIdentityFile,
			label:    "identity.txt",
			path:     idPath,
			identity: id,
			loadErr:  err,
		}
		if err == nil {
			s.publicKey = id.Recipient().String()
		}
		sources = append(sources, s)
	}

	if sshPath := strings.TrimSpace(os.Getenv(utils.SSHIdentityEnvVar)); sshPath != "" {
		id, pub, err := LoadSSHIdentityWithPubKey(sshPath)
		sources = append(sources, identitySource{
			kind:      sourceSSHEnv,
			label:     fmt.Sprintf("$%s (%s)", utils.SSHIdentityEnvVar, sshPath),
			path:      sshPath,
			identity:  id,
			publicKey: pub,
			loadErr:   err,
		})
	}

	if defaultPath := DefaultSSHIdentityPath(); defaultPath != "" && utils.FileExists(defaultPath) {
		id, pub, err := LoadSSHIdentityWithPubKey(defaultPath)
		sources = append(sources, identitySource{
			kind:      sourceSSHDefault,
			label:     defaultPath,
			path:      defaultPath,
			identity:  id,
			publicKey: pub,
			loadErr:   err,
		})
	}

	return sources
}

// ResolveIdentity returns the identity to use for decryption (no ciphertext
// to probe against). Used by write paths and rotate-key flows that need an
// identity before any decrypt attempt.
//
// Precedence: HULAK_MASTER_KEY → identity.txt → HULAK_SSH_IDENTITY → ~/.ssh/id_ed25519.
//
// Strict-fail on explicit config (identity.txt present-but-broken, or
// $HULAK_SSH_IDENTITY set-but-unloadable). Silent fall-through on the
// auto-detected ~/.ssh/id_ed25519 default.
func ResolveIdentity() (age.Identity, error) {
	if raw := strings.TrimSpace(os.Getenv(utils.MasterKey)); raw != "" {
		return parseMasterKey(raw)
	}

	for _, s := range gatherSources(false) {
		if s.loadErr == nil {
			switch s.kind {
			case sourceSSHEnv:
				utils.PrintInfoStderr(fmt.Sprintf("Using SSH identity %s", s.path))
			case sourceSSHDefault:
				utils.PrintInfoStderr(fmt.Sprintf("Using SSH identity %s (no identity.txt found)", s.path))
			}
			return s.identity, nil
		}
		switch s.kind {
		case sourceIdentityFile:
			return nil, s.loadErr
		case sourceSSHEnv:
			return nil, fmt.Errorf("%s is set but key could not be loaded: %w", utils.SSHIdentityEnvVar, s.loadErr)
		}
		// sourceSSHDefault load failure: silently fall through.
	}
	return nil, noIdentityError()
}

// HasAnyIdentity reports whether at least one usable identity source exists
// (master key env, parseable identity.txt, or loadable SSH key). Configured-
// but-broken sources don't count — they can't actually decrypt anything.
func HasAnyIdentity() bool {
	if strings.TrimSpace(os.Getenv(utils.MasterKey)) != "" {
		return true
	}
	for _, s := range gatherSources(false) {
		if s.loadErr == nil {
			return true
		}
	}
	return false
}

// DecryptingIdentity describes one identity source that successfully
// decrypts a given ciphertext. Returned by SourcesThatDecrypt for the
// `secrets identity list` subcommand.
type DecryptingIdentity struct {
	Path      string // file path or "$ENV_VAR"
	PublicKey string // "age1..." or "ssh-ed25519 AAA..."
}

// SourcesThatDecrypt returns every configured identity source that can
// decrypt the given ciphertext, in precedence order. The first element is
// the "default" — the identity hulak would actually use for read paths.
func SourcesThatDecrypt(ciphertext []byte) []DecryptingIdentity {
	var hits []DecryptingIdentity
	for _, s := range gatherSources(true) {
		if s.loadErr != nil {
			continue
		}
		if _, err := DecryptText(ciphertext, s.identity); err == nil {
			hits = append(hits, DecryptingIdentity{Path: s.path, PublicKey: s.publicKey})
		}
	}
	return hits
}

// resolveAndDecrypt is the single-pass identity resolver used by all read
// paths. Probes each source against ciphertext and returns both the matching
// identity and the decrypted plaintext — avoiding the double-decrypt of
// "resolve, then re-decrypt." Same precedence as ResolveIdentityFor;
// HULAK_MASTER_KEY short-circuits with strict semantics.
func resolveAndDecrypt(ciphertext []byte) (age.Identity, []byte, error) {
	if raw := strings.TrimSpace(os.Getenv(utils.MasterKey)); raw != "" {
		id, err := parseMasterKey(raw)
		if err != nil {
			return nil, nil, err
		}
		plain, err := DecryptText(ciphertext, id)
		if err != nil {
			return nil, nil, WrapDecryptError(fmt.Errorf("failed to decrypt store: %w", err))
		}
		return id, plain, nil
	}

	sources := gatherSources(false)
	if len(sources) == 0 {
		return nil, nil, noIdentityError()
	}

	tried := make([]string, 0, len(sources))
	for _, s := range sources {
		if s.loadErr != nil {
			tried = append(tried, fmt.Sprintf("%s [load failed: %v]", s.label, s.loadErr))
			continue
		}
		plain, err := DecryptText(ciphertext, s.identity)
		if err == nil {
			return s.identity, plain, nil
		}
		tried = append(tried, s.label)
	}

	return nil, nil, fmt.Errorf(
		"no available identity could decrypt the store. Tried: %s.\n"+
			"If you should have access, ask a current vault member to add your public key:\n"+
			"  hulak secrets identity add-recipient <your-pubkey>",
		strings.Join(tried, ", "),
	)
}

// ResolveIdentityFor returns the first identity (across all configured sources)
// that can decrypt the given ciphertext. Convenience wrapper around
// resolveAndDecrypt for callers that only need the identity (e.g. import-key
// validation). Read paths should use ReadStore instead — it composes
// resolveAndDecrypt with JSON decoding in a single pass.
//
// Precedence:
//
//  1. HULAK_MASTER_KEY env var — strict, no fallback
//  2. identity.txt (if present)
//  3. $HULAK_SSH_IDENTITY (if set)
//  4. ~/.ssh/id_ed25519 (auto-detected)
//
// On failure, enumerates every source that was tried, including configured-
// but-broken ones (e.g. HULAK_SSH_IDENTITY pointing at a missing file).
func ResolveIdentityFor(ciphertext []byte) (age.Identity, error) {
	id, _, err := resolveAndDecrypt(ciphertext)
	return id, err
}

// noIdentityError returns a helpful error when no identity could be resolved.
func noIdentityError() error {
	return utils.HelpfulError(
		"no identity found — cannot decrypt the vault",
		"Set up an identity using one of these methods",
		[]string{
			"Run 'hulak init' to generate an age keypair",
			"Import an existing key with 'hulak secrets identity import'",
			fmt.Sprintf("Set %s to an AGE-SECRET-KEY- value (for CI)", utils.MasterKey),
			fmt.Sprintf("Set %s to point at your SSH private key", utils.SSHIdentityEnvVar),
			"Place an ed25519 SSH key at ~/.ssh/id_ed25519 (auto-detected)",
		},
	)
}

// parseMasterKey parses the HULAK_MASTER_KEY value with friendly errors.
func parseMasterKey(raw string) (*age.X25519Identity, error) {
	identity, err := age.ParseX25519Identity(raw)
	if err != nil {
		if strings.HasPrefix(raw, AgePrefix) {
			return nil, fmt.Errorf(
				"%s contains what looks like a public key (age1...), not a private key. "+
					"Private keys start with AGE-SECRET-KEY-",
				utils.MasterKey,
			)
		}
		return nil, fmt.Errorf(
			"%s is set but could not be parsed as an age private key: %w\n"+
				"Expected format: AGE-SECRET-KEY-1... ",
			utils.MasterKey, err,
		)
	}
	return identity, nil
}

// WrapDecryptError checks if HULAK_MASTER_KEY is set and the error looks like
// an age "recipient mismatch" failure. If so, wraps it with actionable hints.
// Otherwise returns the original error unchanged.
func WrapDecryptError(err error) error {
	if os.Getenv(utils.MasterKey) == "" {
		return err
	}
	if !strings.Contains(err.Error(), "did not match any of the recipients") {
		return err
	}
	return fmt.Errorf(
		"failed to decrypt store: %s is set but does not match any recipient in this project.\n"+
			"Common causes:\n"+
			"  - This key is for a different project\n"+
			"  - You were removed as a recipient\n"+
			"  - Stale whitespace or quotes from copy-paste",
		utils.MasterKey,
	)
}
