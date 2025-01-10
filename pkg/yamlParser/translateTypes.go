package yamlParser

import (
	"fmt"
	"regexp"
	"strings"
)

type ActionType string

// These consts represent functions users can take in a yaml file
const (
	DotString  ActionType = "DotString"
	GetValueOf ActionType = "GetValueOf"
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

// Returns ActionType
func DelimiterLogic(delimiterString string) (Action, bool) {
	strHasDelimiter, innerStr := stringHasDelimiter(delimiterString)
	if !strHasDelimiter {
		return Action{}, false
	}

	innerStrChunks := strings.Split(innerStr, " ")

	// Check for DotString action
	if len(innerStrChunks) == 1 && strings.HasPrefix(innerStrChunks[0], ".") {
		dotStr := strings.TrimPrefix(innerStrChunks[0], ".")
		return Action{Type: DotString, DotString: dotStr}, true
	}

	// Check for GetValueOf action
	if len(innerStrChunks) == 3 && innerStrChunks[0] == "getValueOf" {
		cleanedChunks := cleanStrings(innerStrChunks[1:])
		return Action{
			Type:       GetValueOf,
			GetValueOf: append([]string{innerStrChunks[0]}, cleanedChunks...),
		}, true
	}

	// Invalid case
	return Action{}, false
}

// Recurses through the raw map prior to actions, beforeMap,
// and finds the key and it's path that needs type conversion.
// The resulting map helps us determine exact location to replace the values in afterMap
/*
{
  "miles": "modified_value", // flat map
  "person -> age": "modified_value" // for nested maps
  "company[0]" -> "position" // for arrays
}
*/
func FindPathInMap(
	beforeMap map[string]interface{},
	parentKey string,
) map[string]interface{} {
	cmprt := make(map[string]interface{})
	for bKey, bValue := range beforeMap {
		currentKey := bKey
		// if the parentKey has something.
		if parentKey != "" {
			currentKey = parentKey + " -> " + bKey
		}

		switch bTypeVal := bValue.(type) {
		case string:
			ok, _ := stringHasDelimiter(bTypeVal)
			if ok {
				cmprt[currentKey] = "modified_value"
			}
		case map[string]interface{}:
			cmprt = mergeMaps(cmprt, FindPathInMap(bTypeVal, currentKey))
		case []map[string]interface{}:
			for idx, val := range bTypeVal {
				key := fmt.Sprintf("%s[%d]", currentKey, idx)
				cmprt = mergeMaps(cmprt, FindPathInMap(val, key))
			}
		default:
			fmt.Println("uncovered type")
			cmprt[currentKey] = bValue
		}
	}
	return cmprt
}

// Helper functions to merge two maps
func mergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	for k, v := range map2 {
		map1[k] = v
	}
	return map1
}

// Helper function to clean strings of backtick (“), and double qoutes ("")
func cleanStrings(stringsToClean []string) []string {
	cleaned := make([]string, len(stringsToClean))
	for i, str := range stringsToClean {
		cleaned[i] = strings.NewReplacer(`"`, "", "`", "").Replace(str)
	}
	return cleaned
}
