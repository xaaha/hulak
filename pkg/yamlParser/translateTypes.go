package yamlParser

import (
	"fmt"
	"regexp"
	"strings"
)

type ActionType string

// These consts represent functions users can take in a yaml file
// except Invalid, which represents error
const (
	DotString  ActionType = "DotString"
	GetValueOf ActionType = "GetValueOf"
	Invalid    ActionType = "Invalid"
)

type Action struct {
	Type       ActionType
	DotString  string
	GetValueOf []string
}

// checks whether string matches exactly "{{value}}"
// and retuns whether the string matches the delimiter criteria and the associated content
// So, the "{{ .value }}" returns "true, .value". Space is trimmed around the return string
func stringHasDelimiter(value string) (bool, string) {
	if len(value) < 4 || !strings.HasPrefix(value, "{{") || !strings.HasSuffix(value, "}}") {
		return false, ""
	}
	if strings.Count(value[:3], "{") > 2 || strings.Count(value[len(value)-3:], "}") > 2 {
		return false, ""
	}
	content := value[2 : len(value)-2]
	re := regexp.MustCompile(`^\s+$`)
	onlyHasEmptySpace := re.Match([]byte(value))
	if len(content) == 0 || onlyHasEmptySpace {
		return false, ""
	}
	content = strings.TrimSpace(content)
	return len(content) > 0, content
}

// Accepts a delimiterString and returns an Action struct to help the afterMap
// navigate to the appropriate dictionary for value replacement.
// Additionally, it removes double quotes, single quotes, dots (.),
// and backticks (`) from the action.
func delimiterLogicAndCleanup(delimiterString string) Action {
	strHasDelimiter, innerStr := stringHasDelimiter(delimiterString)
	if !strHasDelimiter {
		return Action{Type: Invalid}
	}

	innerStrChunks := strings.Split(innerStr, " ")

	// Check for DotString action
	if len(innerStrChunks) == 1 && strings.HasPrefix(innerStrChunks[0], ".") {
		dotStr := strings.TrimPrefix(innerStrChunks[0], ".")
		return Action{Type: DotString, DotString: dotStr}
	}

	if len(innerStrChunks) == 3 && innerStrChunks[0] == "getValueOf" {
		cleanedChunks := cleanStrings(innerStrChunks[1:])
		return Action{
			Type:       GetValueOf,
			GetValueOf: append([]string{innerStrChunks[0]}, cleanedChunks...),
		}
	}

	return Action{Type: Invalid}
}

// Recurses through the raw map prior to actions, beforeMap,
// and finds the key and it's path that needs type conversion.
// The resulting map helps us determine exact location to replace the values in afterMap
func findPathFromMap(
	beforeMap map[string]interface{},
	parentKey string,
) map[ActionType][]string {
	cmprt := make(map[ActionType][]string)
	for bKey, bValue := range beforeMap {
		currentKey := bKey
		if parentKey != "" {
			currentKey = parentKey + " -> " + bKey
		}
		switch bTypeVal := bValue.(type) {
		case string:
			action := delimiterLogicAndCleanup(bTypeVal)
			if action.Type != Invalid {
				// since we only have two actions, we can keep it here.
				// but this could be a problem on large number of cases
				switch action.Type {
				case DotString:
					cmprt[DotString] = append(cmprt[DotString], currentKey)
				case GetValueOf:
					cmprt[GetValueOf] = append(cmprt[GetValueOf], currentKey)
				}
			}
		case map[string]interface{}:
			subMap := findPathFromMap(bTypeVal, currentKey)
			for key, values := range subMap {
				cmprt[key] = append(cmprt[key], values...)
			}
		case []map[string]interface{}:
			for idx, val := range bTypeVal {
				arrayKey := fmt.Sprintf("%s[%d]", currentKey, idx)
				subMap := findPathFromMap(val, arrayKey)
				for key, values := range subMap {
					cmprt[key] = append(cmprt[key], values...)
				}
			}
		default:
			// No action needed for now. We should keep expanding cases above
			// as they appear
		}
	}

	return cmprt
}

// Helper function to clean strings of backtick (`), double qoutes(""), and single qoutes (”)
// around the string
func cleanStrings(stringsToClean []string) []string {
	cleaned := make([]string, len(stringsToClean))
	for i, str := range stringsToClean {
		cleaned[i] = strings.NewReplacer(`"`, "", "`", "").Replace(str)
	}
	return cleaned
}
