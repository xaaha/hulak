// Centralizes the prelude that every secrets handler runs before touching
// the vault: project marker check → interactive picker fallback → env name
// validation → existence assertion. Lifted here so each handler shrinks
// from five repeated statements to one, and a future change to the prelude
// shape lands in one place.
package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// resolveAndValidateEnv runs the shared prelude for secrets handlers:
//
//  1. requireVaultProject — refuse to operate outside a hulak project.
//  2. resolveEnv          — fall back to the interactive picker when
//     envName is empty.
//  3. ValidateEnvName     — reject names containing path separators,
//     leading underscores, etc.
//
// Returns the resolved env name, plus the cancelled bool from the picker so
// callers can `return nil` cleanly on Esc instead of falling through to the
// validator (which would otherwise emit a misleading "name cannot be empty"
// error).
//
// Step 1 (project check) runs first so users see "not a hulak project" before
// the picker tries to read the vault and produces a confusing decrypt error.
func resolveAndValidateEnv(envName string) (resolved string, cancelled bool, err error) {
	if err := requireVaultProject(); err != nil {
		return "", false, err
	}

	resolved, cancelled, err = resolveEnv(envName)
	if err != nil {
		return "", false, err
	}
	if cancelled {
		return "", true, nil
	}

	if err := utils.ValidateEnvName(resolved); err != nil {
		return "", false, err
	}
	return resolved, false, nil
}

// requireEnvExists fetches the named env from store and returns the
// canonical "environment %q not found" error when it does not.
// Callers that need an existing env (get, delete, list keys) use this
// to keep the error wording consistent across commands.
func requireEnvExists(store *vault.Store, envName string) (vault.Env, error) {
	env := store.GetEnv(envName)
	if env == nil {
		return nil, fmt.Errorf("environment %q not found in vault store", envName)
	}
	return env, nil
}
