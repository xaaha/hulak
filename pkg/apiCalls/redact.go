// Package apicalls has all things related to api call
package apicalls

import (
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// NewSecretRedactor builds the value masker for a request's rendered output
// from the secrets it was resolved with. Anything out of the encrypted vault
// is a secret by provenance; with plain env files only key names that look
// like credentials are.
func NewSecretRedactor(secrets map[string]any) *utils.ValueRedactor {
	return utils.NewValueRedactor(secrets, vault.DetectStore() == vault.StoreAge)
}
