package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains encrypted store persistence and environment secret key management.

// StoreVersion is the current schema version of the encrypted JSON store.
// Bumped only on backwards-incompatible changes to the on-disk layout.
const StoreVersion = 1

// versionFieldName is the JSON key for the schema version. The leading
// underscore reserves it from collision with user-defined env names
// (which utils.ValidateEnvName rejects when starting with '_').
const versionFieldName = "_version"

// MaxStoreSizeWarnBytes is the soft threshold for the decrypted store size.
// Above this, ReadStore prints a one-time stderr warning that performance
// may degrade. Not a hard limit — large stores still work.
const MaxStoreSizeWarnBytes = 1 << 20 // 1 MiB

// storeSizeWarnOnce gates the size warning to once per process, so users
// don't get spammed when a single command issues multiple reads.
var storeSizeWarnOnce sync.Once

// Env is the user's environment like 'staging', 'prod',
type Env map[string]any

// Store holds all environments and their key-value pairs.
// Each top-level key is an environment name (e.g. "global", "prod"),
// and its value is a map of secret key-value pairs.
type Store struct {
	Envs map[string]Env
}

// MarshalJSON serializes the store as a flat object with `_version` alongside
// the named environments: {"_version": 1, "global": {...}, "prod": {...}}.
// Uses SetEscapeHTML(false) so characters like &, <, > in values (e.g. URLs)
// are kept literal instead of being escaped to \u003c etc.
func (s *Store) MarshalJSON() ([]byte, error) {
	out := make(map[string]any, len(s.Envs)+1)
	out[versionFieldName] = StoreVersion
	for name, env := range s.Envs {
		out[name] = env
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return nil, err
	}
	// Encode appends exactly one trailing newline; strip it for MarshalJSON contract.
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// UnmarshalJSON parses the flat object form, validates the version, and
// populates Envs. A missing `_version` field is treated as version 1 (legacy
// stores written before versioning was introduced).
func (s *Store) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse store: %w", err)
	}

	// Default to 1 (not StoreVersion) — a missing _version field means the
	// store was written by pre-versioning hulak, which is by definition v1.
	// This must stay 1 forever, even when StoreVersion increments.
	version := 1
	if vRaw, ok := raw[versionFieldName]; ok {
		if err := json.Unmarshal(vRaw, &version); err != nil {
			return fmt.Errorf("invalid %s field: %w", versionFieldName, err)
		}
	}
	if version > StoreVersion {
		return fmt.Errorf(
			"store was written by a newer hulak (version %d, this version supports %d) — upgrade with `brew upgrade hulak`",
			version,
			StoreVersion,
		)
	}

	// skip version, parse env
	s.Envs = make(map[string]Env, len(raw))
	for name, envRaw := range raw {
		if name == versionFieldName {
			continue
		}
		env, err := decodeEnv(envRaw)
		if err != nil {
			return fmt.Errorf("failed to parse env %q: %w", name, err)
		}
		s.Envs[name] = env
	}
	return nil
}

// decodeEnv parses a single environment's raw JSON bytes into an Env map.
// Numbers are kept as json.Number to preserve int/float distinction and avoid
// precision loss for large integers.
func decodeEnv(raw json.RawMessage) (Env, error) {
	var env Env
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&env); err != nil {
		return nil, err
	}
	return env, nil
}

// GetEnv returns the key-value map for a given environment name.
// Returns nil if the environment does not exist.
func (s *Store) GetEnv(envName string) map[string]any {
	return s.Envs[envName]
}

// ListEnvs returns a sorted list of environment names in the store.
func (s *Store) ListEnvs() []string {
	envs := make([]string, 0, len(s.Envs))
	for name := range s.Envs {
		envs = append(envs, name)
	}
	sort.Strings(envs)
	return envs
}

// EnsureSection creates an empty section for envName if absent. Returns true
// when a new section was created, false when one already existed. Lets
// callers seed environments without reaching into the Envs map directly.
func (s *Store) EnsureSection(envName string) bool {
	if s.Envs == nil {
		s.Envs = make(map[string]Env)
	}
	if _, ok := s.Envs[envName]; ok {
		return false
	}
	s.Envs[envName] = make(Env)
	return true
}

// SetKey upserts a key-value pair in the given environment.
// Creates the environment if it does not exist.
func (s *Store) SetKey(envName, key string, value any) {
	if s.Envs == nil {
		s.Envs = make(map[string]Env)
	}
	if s.Envs[envName] == nil {
		s.Envs[envName] = make(Env)
	}
	s.Envs[envName][key] = value
}

// DeleteKey removes a key from the given environment.
// No-op if the environment or key does not exist.
func (s *Store) DeleteKey(envName, key string) {
	env := s.Envs[envName]
	if env == nil {
		return
	}
	delete(env, key)
}

// storePath returns the absolute path to .hulak/store.age in the project root.
func storePath() (string, error) {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return "", err
	}
	return filepath.Join(markerPath, utils.StoreFile), nil
}

// ReadStore decrypts store.age and returns the Store.
// Uses json.Decoder.UseNumber() to preserve int/float distinction.
func ReadStore(identity age.Identity) (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}

	cipherText, err := os.ReadFile(path)
	if err != nil {
		// empty store.age if it does not exist
		if os.IsNotExist(err) {
			return &Store{Envs: make(map[string]Env)}, nil
		}
		return nil, fmt.Errorf("failed to read store: %w", err)
	}

	plainText, err := DecryptText(cipherText, identity)
	if err != nil {
		return nil, WrapDecryptError(fmt.Errorf("failed to decrypt store: %w", err))
	}

	maybeWarnStoreSize(len(plainText))

	store := &Store{}
	if err := json.Unmarshal(plainText, store); err != nil {
		return nil, err
	}
	return store, nil
}

// maybeWarnStoreSize prints a one-time stderr warning if the decrypted store
// exceeds MaxStoreSizeWarnBytes. Stderr (not stdout) so callers like
// `hulak env get` captured via $(...) stay clean.
func maybeWarnStoreSize(size int) {
	if size <= MaxStoreSizeWarnBytes {
		return
	}
	storeSizeWarnOnce.Do(func() {
		utils.PrintWarningStderr(fmt.Sprintf(
			"decrypted store is %.1f MB — large stores can slow `hulak env` operations; consider {{getFile \"path\"}} for big blobs",
			float64(size)/(1<<20),
		))
	})
}

// WriteStore marshals the store to JSON, encrypts it, and writes to store.age.
// Uses atomic write: writes to .tmp first, then renames to prevent corruption.
func WriteStore(store *Store, recipients ...age.Recipient) error {
	path, err := storePath()
	if err != nil {
		return err
	}

	// Encoder with SetEscapeHTML(false) keeps &, <, > literal in values
	// (e.g. URLs with query params) instead of escaping to \u003c etc.
	// Indented for readability in hulak env edit.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(store); err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}
	plainText := buf.Bytes()

	cipherText, err := EncryptText(plainText, recipients...)
	if err != nil {
		return fmt.Errorf("failed to encrypt store: %w", err)
	}

	return utils.AtomicWriteFile(path, cipherText, utils.SecretPer, utils.DirPer)
}

// WriteStoreToRecipients loads recipients from .hulak/recipients.txt and
// encrypts the store to all of them. Most write-path callers should use
// this instead of passing recipients manually.
func WriteStoreToRecipients(store *Store) error {
	recipients, err := LoadRecipients()
	if err != nil {
		return fmt.Errorf("failed to load recipients: %w", err)
	}
	return WriteStore(store, recipients...)
}

type StoreType int

const (
	StoreNone StoreType = iota
	StoreAge
	StoreClassic
)

// DetectStore checks which storage backend is available.
// .hulak/store.age takes priority over env/ if both exist.
func DetectStore() StoreType {
	if path, err := storePath(); err == nil && utils.FileExists(path) {
		return StoreAge
	}

	projectRoot, ok := utils.FindProjectRoot()
	if !ok {
		return StoreNone
	}

	envDir := filepath.Join(projectRoot, utils.EnvironmentFolder)
	if utils.DirExists(envDir) {
		return StoreClassic
	}
	return StoreNone
}
