// Package apicalls has all things related to api call
package apicalls

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// NewSecretRedactor builds the value masker for the request at path from the
// secrets it was resolved with. Anything out of the encrypted vault is a
// secret by provenance; with plain env files only key names that look like
// credentials are. The variables path references scope the unresolved footer.
func NewSecretRedactor(path string, secrets map[string]any) (*utils.ValueRedactor, error) {
	referenced, err := utils.FileTemplateVarNames(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return utils.NewValueRedactor(secrets, vault.DetectStore() == vault.StoreAge, referenced), nil
}
