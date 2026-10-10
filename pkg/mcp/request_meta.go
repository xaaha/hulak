package mcp

import (
	"os"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/utils"
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
// '/' inside {{ }} does not end the host.
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
