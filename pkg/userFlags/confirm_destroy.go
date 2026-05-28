// Shared confirm-before-destroy helper. Wraps utils.ConfirmAction with the
// canonical hulak phrasing so every destructive command prints the same shape
// of prompt. Indirected through a package variable so tests can stub the
// prompt without manipulating os.Stdin.
package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
)

// confirmActionFn is the prompt routine called by confirmDestroy. Tests
// swap this for a fake; production code leaves it pointed at the real
// ConfirmAction.
var confirmActionFn = utils.ConfirmAction

// confirmDestroy gates a destructive action behind an interactive Y/N
// prompt, with two short-circuits:
//
//  1. force = true (typically wired to --yes): skip the prompt and proceed.
//     For scripts, CI, and `xargs ... | hulak ...` pipelines.
//  2. count = 0: nothing to destroy, so prompting would be noise. Skip it.
//
// Standard prompt format:
//
//	This will permanently delete N <description>. Continue? [y/N]
//
// description should NOT include the count — confirmDestroy prepends it.
// Plural agreement is the caller's job since "+s" is unreliable across
// nouns (callers pass "key in <env>" / "keys in <env>" themselves).
func confirmDestroy(description string, count int, force bool) (bool, error) {
	if force {
		return true, nil
	}
	if count == 0 {
		return true, nil
	}
	prompt := fmt.Sprintf(
		"This will permanently delete %d %s. Continue? [y/N] ",
		count, description,
	)
	return confirmActionFn(prompt)
}
