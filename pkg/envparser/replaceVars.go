// Package envparser contains environment parsing and functions around it
package envparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/xaaha/hulak/pkg/actions"
	"github.com/xaaha/hulak/pkg/utils"
)

// extractMissingKey parses a template error and extracts the missing key name
// Returns the key name if found, empty string otherwise
func extractMissingKey(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	// Template errors for missing keys look like:
	// "template: template:1:2: executing "template" at <.keyName>: map has no entry for key "keyName""

	// Look for the pattern: map has no entry for key "keyName"
	if strings.Contains(errMsg, "map has no entry for key") {
		start := strings.Index(errMsg, `map has no entry for key "`)
		if start != -1 {
			start += len(`map has no entry for key "`)
			end := strings.Index(errMsg[start:], `"`)
			if end != -1 {
				return errMsg[start : start+end]
			}
		}
	}
	return ""
}

// formatMissingKeyError creates a user-friendly error message for missing template variables
func formatMissingKeyError(keyName string) error {
	env := os.Getenv(utils.EnvKey)
	if env == "" {
		env = utils.DefaultEnvVal
	}

	return fmt.Errorf(
		"key %q not found in environment %q.\n"+
			"Run 'hulak secrets keys set %s <value> --env %s' to add it to the encrypted vault.\n"+
			"For classic env/ mode, add %s=<value> to env/%s.env.\n"+
			"Or use a different environment with the -env flag",
		keyName,
		env,
		keyName,
		env,
		keyName,
		env,
	)
}

// actionTokenRe matches the command identifier at the start of a template
// action ("{{", optional trim marker, optional spaces, then the identifier).
// Field accessors ({{.key}}) and variables ({{$x}}) don't match — they don't
// start with an identifier char — so only function-position tokens are touched.
var actionTokenRe = regexp.MustCompile(`(\{\{-?[ \t]*)([A-Za-z_][A-Za-z0-9_]*)`)

// canonicalizeActionNames rewrites the leading command token of each template
// action to its canonical spelling when it is a known action, so getfile,
// GetFile, and get_file all resolve to getFile. Only tokens whose normalized
// form is a known action are rewritten; every other identifier is left as-is.
func canonicalizeActionNames(s string) string {
	return actionTokenRe.ReplaceAllStringFunc(s, func(match string) string {
		groups := actionTokenRe.FindStringSubmatch(match)
		prefix, token := groups[1], groups[2]
		if canonical, ok := utils.CanonicalActionName(token); ok {
			return prefix + canonical
		}
		return match
	})
}

func replaceVariables(
	strToChange string,
	secretsMap map[string]any,
	currentFile string,
) (string, error) {
	if len(strToChange) == 0 {
		return "", nil
	}

	funcMap := template.FuncMap{
		utils.TemplateFuncGetValueOf: actions.GetValueOf,
		utils.TemplateFuncGetFile:    getFileFor(currentFile),
		utils.TemplateFuncBasicAuth:  actions.BasicAuth,
		utils.TemplateFuncOs:         os.Getenv,
	}

	tmpl, err := template.New("template").
		Funcs(funcMap).
		Option("missingkey=error").
		Parse(canonicalizeActionNames(strToChange))
	if err != nil {
		return "", err
	}
	var result bytes.Buffer
	err = tmpl.Execute(&result, secretsMap)
	if err != nil {
		// Check if this is a missing key error and format it nicely
		if missingKey := extractMissingKey(err); missingKey != "" {
			return "", formatMissingKeyError(missingKey)
		}
		// For other template errors, return as-is
		return "", fmt.Errorf("template error: %w", err)
	}
	return result.String(), nil
}

// Iterates over a map of secret values (secretsMap), resolving any string values
// containing template variables using the replaceVariables function.
// It ensures that non-string values (e.g., booleans, integers) are preserved and validates against unsupported types.
// Returns a new map with resolved values or an error if any resolution fails.
func prepareMap(secretsMap map[string]any, currentFile string) (map[string]any, error) {
	updatedMap := make(map[string]any)
	for key, val := range secretsMap {
		switch v := val.(type) {
		case string:
			changedValue, err := replaceVariables(v, secretsMap, currentFile)
			if err != nil {
				return nil, err
			}
			updatedMap[key] = changedValue
		case json.Number, bool, int, float64, nil:
			updatedMap[key] = v
		default:
			return nil, fmt.Errorf("unsupported type for key '%s': %T", key, val)
		}
	}
	return updatedMap, nil
}

// SubstituteVariables Substitutes template variables in a given string strToChange using the secretsMap.
// It first prepares the map by resolving all nested variables using prepareMap
// and then applies replaceVariables to the input string.
// currentFile is the path of the request file being processed; it anchors the
// getFile "*" sibling shorthand. Pass "" when there is no file context.
// Returns the final substituted string or an error if any step fails.
func SubstituteVariables(
	strToChange string,
	secretsMap map[string]any,
	currentFile string,
) (any, error) {
	finalMap, err := prepareMap(secretsMap, currentFile)
	if err != nil {
		return nil, err
	}

	result, err := replaceVariables(strToChange, finalMap, currentFile)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// getFileFor returns a getFile template function bound to the request file
// currently being processed. When the arg uses the "*" sibling shorthand it is
// expanded relative to currentFile (see utils.SiblingPath); otherwise it falls
// through to the normal path-based actions.GetFile unchanged.
func getFileFor(currentFile string) func(string) (string, error) {
	return func(arg string) (string, error) {
		sibling, ok := utils.SiblingPath(currentFile, arg)
		if !ok {
			return actions.GetFile(arg)
		}
		if currentFile == "" {
			return "", fmt.Errorf(
				`getFile %q sibling shorthand is only valid inside a request file`, arg,
			)
		}
		if arg == "*" {
			return "", fmt.Errorf(`getFile %q needs an extension, e.g. "*.gql"`, arg)
		}
		content, err := actions.GetFile(sibling)
		if err != nil {
			return "", fmt.Errorf(
				"no sibling file %s next to %s: %w",
				filepath.Base(sibling), filepath.Base(currentFile), err,
			)
		}
		return content, nil
	}
}
