package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Env is the user's environment like 'staging', 'prod',
type Env map[string]any

// Store holds all environments and their key-value pairs.
// Each top-level key is an environment name (e.g. "global", "prod"),
// and its value is a map of secret key-value pairs.
type Store struct {
	Envs map[string]Env
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
		return nil, fmt.Errorf("failed to decrypt store: %w", err)
	}

	var envs map[string]Env
	dec := json.NewDecoder(bytes.NewReader(plainText))
	dec.UseNumber() // avoids float64, get exact original text
	if err := dec.Decode(&envs); err != nil {
		return nil, fmt.Errorf("failed to parse store: %w", err)
	}

	return &Store{Envs: envs}, nil
}

// WriteStore marshals the store to JSON, encrypts it, and writes to store.age.
// Uses atomic write: writes to .tmp first, then renames to prevent corruption.
func WriteStore(store *Store, recipients ...age.Recipient) error {
	path, err := storePath()
	if err != nil {
		return err
	}

	plainText, err := json.MarshalIndent(store.Envs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	cipherText, err := EncryptText(plainText, recipients...)
	if err != nil {
		return fmt.Errorf("failed to encrypt store: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, cipherText, utils.SecretPer); err != nil {
		os.Remove(tmpPath) // remove tmp, might be corrupted
		return fmt.Errorf("failed to write store: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize store: %w", err)
	}

	return nil
}
