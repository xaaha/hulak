package userflags

import (
	"flag"
	"strings"
)

// registerNameFlag binds --name on fs to a single underlying string. Returns
// a pointer so the handler reads the parsed value after flag.Parse. Pair with
// resolveRecipientName at call time to keep "user-supplied name, else OS
// username" semantics consistent across every command that labels a recipient.
func registerNameFlag(fs *flag.FlagSet, usage string) *string {
	var name string
	fs.StringVar(&name, "name", "", usage)
	return &name
}

// resolveRecipientName returns the first non-empty value in order, treating
// whitespace-only strings as empty. Use to centralize the "user-supplied
// name, else a sensible default" rule that varies by command:
//
//	gen-identity / import-key: resolveRecipientName(*name, utils.Username())
//	add-recipient:             resolveRecipientName(*name, gitHubUser)
//
// Returns "" only if every value is empty.
func resolveRecipientName(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
