package vault

import (
	"sort"
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
