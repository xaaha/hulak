package mcp

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// readRequestDoc decodes a request file into a map with lowercased keys.
func readRequestDoc(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	return utils.ConvertKeysToLowerCase(doc), nil
}

// rawHost returns the host part of a request URL as written, without scheme,
// userinfo, path, query, or fragment. Template actions stay unresolved, and a
// '/' inside {{ }} does not end the host. A literal host followed later by an
// '@' is empty: an unencoded '/', '?' or '#' in a password would put part of
// it in the host.
func rawHost(url string) string {
	s := strings.TrimSpace(url)
	inAction := actionMask(s)
	for i := range len(s) {
		if inAction[i] {
			continue
		}
		if strings.HasPrefix(s[i:], "://") {
			return rawHost(s[i+len("://"):])
		}
		if strings.IndexByte("/?#", s[i]) >= 0 {
			if hasLiteral(inAction[:i]) && literalIndex(s[i:], inAction[i:], '@') >= 0 {
				return ""
			}
			s = s[:i]
			break
		}
	}
	for i := len(s) - 1; i >= 0; i-- {
		if !inAction[i] && s[i] == '@' {
			return s[i+1:]
		}
	}
	return s
}

func hasLiteral(inAction []bool) bool {
	return slices.Contains(inAction, false)
}

func literalIndex(s string, inAction []bool, c byte) int {
	for i := range len(s) {
		if !inAction[i] && s[i] == c {
			return i
		}
	}
	return -1
}

// actionMask reports, per byte of s, whether it lies inside a {{ }} action.
func actionMask(s string) []bool {
	mask := make([]bool, len(s))
	depth := 0
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], "{{"):
			depth++
			mask[i], mask[i+1] = true, true
			i++
		case depth > 0 && strings.HasPrefix(s[i:], "}}"):
			depth--
			mask[i], mask[i+1] = true, true
			i++
		default:
			mask[i] = depth > 0
		}
	}
	return mask
}

// requestAuth names how a request authenticates: "oauth2" for an Auth kind,
// the Authorization scheme ("bearer", "basic"), or "header <name>" for another
// credential header. When the credential comes from getValueOf, " from
// <request>" names the request it is read from. Empty means no auth found.
func requestAuth(kind string, doc map[string]any) string {
	if kind == string(yamlparser.KindAuth) {
		return "oauth2"
	}
	headers, _ := doc["headers"].(map[string]any)
	credentials := map[string]any{}
	for name, value := range headers {
		if utils.IsSensitiveHeader(name) {
			credentials[strings.ToLower(name)] = value
		}
	}
	if len(credentials) == 0 {
		return ""
	}
	name := slices.Min(slices.Collect(maps.Keys(credentials)))
	value, _ := credentials[name].(string)
	label := "header " + name
	if name == "authorization" {
		if scheme := authScheme(value); scheme != "" {
			label = scheme
		}
	}
	for _, args := range templateActions(value) {
		if args[0] == utils.TemplateFuncGetValueOf && len(args) == 3 {
			return label + " from " + args[2]
		}
	}
	return label
}

// authSchemes are the Authorization schemes reported by name. Any other first
// word may be the credential itself, so it is never echoed.
var authSchemes = []string{
	"aws4-hmac-sha256", "basic", "bearer", "digest", "dpop", "hoba",
	"mutual", "negotiate", "ntlm", "oauth", "token",
}

// authScheme returns the lowercased scheme of an Authorization value, from a
// known first word followed by a credential, or a leading basicAuth action.
// Empty when unknown.
func authScheme(value string) string {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(v, "{{") {
		if actions := templateActions(v); len(actions) > 0 && actions[0][0] == utils.TemplateFuncBasicAuth {
			return "basic"
		}
		return ""
	}
	word, rest, _ := strings.Cut(v, " ")
	scheme := strings.ToLower(word)
	if rest == "" || !slices.Contains(authSchemes, scheme) {
		return ""
	}
	return scheme
}

// templateActions returns each function-call action in s as its canonical
// function name followed by its quoted string arguments.
func templateActions(s string) [][]string {
	var out [][]string
	for {
		_, rest, ok := strings.Cut(s, "{{")
		if !ok {
			return out
		}
		body, after, ok := strings.Cut(rest, "}}")
		if !ok {
			return out
		}
		s = after
		fields := splitActionArgs(strings.Trim(strings.TrimSpace(body), "-"))
		if len(fields) == 0 {
			continue
		}
		if name, ok := utils.CanonicalActionName(fields[0]); ok {
			out = append(out, append([]string{name}, fields[1:]...))
		}
	}
}

// splitActionArgs splits an action body on spaces, keeping a quoted argument
// whole and returning it without its quotes.
func splitActionArgs(body string) []string {
	var fields []string
	for body = strings.TrimSpace(body); body != ""; body = strings.TrimSpace(body) {
		quote := body[0]
		if quote != '"' && quote != '\'' && quote != '`' {
			word, rest, _ := strings.Cut(body, " ")
			fields = append(fields, word)
			body = rest
			continue
		}
		end := 1
		for end < len(body) && body[end] != quote {
			if body[end] == '\\' && quote == '"' {
				end++
			}
			end++
		}
		fields = append(fields, body[1:min(end, len(body))])
		body = body[min(end+1, len(body)):]
	}
	return fields
}
