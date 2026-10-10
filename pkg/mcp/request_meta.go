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

// rawHost strips scheme, userinfo, path, query, and fragment, leaving template actions unresolved.
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
		if strings.IndexByte(`/?#\`, s[i]) >= 0 {
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

// actionMask marks the bytes inside each closed {{ }} action; an unclosed {{ is literal.
func actionMask(s string) []bool {
	mask := make([]bool, len(s))
	for i := 0; i < len(s); {
		open := strings.Index(s[i:], "{{")
		if open < 0 {
			break
		}
		open += i
		end := strings.Index(s[open+2:], "}}")
		if end < 0 {
			break
		}
		end += open + 2 + len("}}")
		for j := open; j < end; j++ {
			mask[j] = true
		}
		i = end
	}
	return mask
}

// requestAuth returns e.g. "oauth2", "bearer from login", or "header x-api-key", or "" for no auth.
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

// Only these are echoed: any other first word may be the credential itself.
var authSchemes = []string{
	"aws4-hmac-sha256", "basic", "bearer", "digest", "dpop", "hoba",
	"mutual", "negotiate", "ntlm", "oauth", "token",
}

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

// templateActions returns each known function action as its canonical name and arguments.
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
