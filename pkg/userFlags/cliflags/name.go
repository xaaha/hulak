package cliflags

import (
	"flag"
	"strings"
)

// RegisterName binds --name on fs to a single underlying string. Returns
// a pointer so the handler reads the parsed value after flag.Parse. Pair with
// ResolveRecipientName at call time to keep "user-supplied name, else OS
// username" semantics consistent across every command that labels a recipient.
func RegisterName(fs *flag.FlagSet, usage string) *string {
	var name string
	fs.StringVar(&name, "name", "", usage)
	return &name
}

// ResolveRecipientName returns the first non-empty value in order, treating
// whitespace-only strings as empty. Use to centralize the "user-supplied
// name, else a sensible default" rule that varies by command:
//
//	gen-identity / import-key: ResolveRecipientName(*name, utils.Username())
//	add-recipient:             ResolveRecipientName(*name, gitHubUser)
//
// Returns "" only if every value is empty.
func ResolveRecipientName(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
