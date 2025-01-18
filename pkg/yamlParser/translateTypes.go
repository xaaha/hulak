package yamlParser

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
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

// TODO: Fix the logic
// Translates value types user picked in the secretsMap (.env) and
func TranslateType(
	beforeMap, afterMap, secretsMap map[string]interface{},
	getValueOfInterface interface{},
) (map[string]interface{}, error) {
	// Find the path map from beforeMap
	pathMap := findPathFromMap(beforeMap, "")

	// Iterate through the paths grouped by their action type
	for actionKey, pathArr := range pathMap {
		for _, str := range pathArr {
			// Parse the path string into a structured path array
			path, err := parsePath(str)
			if err != nil {
				return nil, err
			}

			current := afterMap
			var parent interface{}
			var lastKey interface{} // last item in the path array

			for i, key := range path {
				// Stop before the last key to prepare for value update
				if i == len(path)-1 {
					lastKey = key
					break
				}

				switch typedKey := key.(type) {
				case string:
					// Navigate through maps
					if nextMap, ok := current[typedKey].(map[string]interface{}); ok {
						parent = current
						current = nextMap
					} else if arr, ok := current[typedKey].([]interface{}); ok {
						parent = arr
						current = nil // Prepare for array index handling
					} else {
						// Key does not exist or is not a map/array, skip
						current = nil
					}
				case int:
					// Navigate through arrays
					if parentArr, ok := parent.([]interface{}); ok {
						if typedKey >= 0 && typedKey < len(parentArr) {
							parent = current
							current = parentArr[typedKey].(map[string]interface{})
						} else {
							current = nil // Index out of bounds
						}
					} else {
						current = nil
					}
				default:
					current = nil
				}

				if current == nil {
					break
				}
			}

			// If we successfully navigated to the last key, process the value
			if current != nil {
				var compareVal interface{}
				if lastKeyStr, ok := lastKey.(string); ok {
					if actionKey == DotString {
						compareVal, ok = secretsMap[lastKeyStr]
						if !ok {
							// Skip if the key does not exist in secretsMap
							continue
						}
					} else if actionKey == GetValueOf {
						compareVal = getValueOfInterface
					}
					if reflect.TypeOf(current[lastKeyStr]) != reflect.TypeOf(compareVal) {
						convertedVal, err := convertType(current[lastKeyStr], compareVal)
						if err == nil {
							current[lastKeyStr] = convertedVal
						}
					}
				}
			}
		}
	}
	return afterMap, nil
}

// dynamically finds type for other actions (currently only getValueOf)
func convertType(value, targetType interface{}) (interface{}, error) {
	switch targetType.(type) {
	case int:
		switch v := value.(type) {
		case string:
			return strconv.Atoi(v)
		case float64:
			return int(v), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to int", value)
		}
	case string:
		return fmt.Sprintf("%v", value), nil
	case float64:
		switch v := value.(type) {
		case string:
			return strconv.ParseFloat(v, 64)
		case int:
			return float64(v), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to float64", value)
		}
	default:
		return nil, fmt.Errorf("unsupported target type %T", targetType)
	}
}

//	func getValueAtPath(m map[string]interface{}, path []interface{}) (interface{}, bool) {
//		current := m
//		for i, key := range path {
//			switch typedKey := key.(type) {
//			case string:
//				if next, ok := current[typedKey].(map[string]interface{}); ok {
//					current = next
//				} else if i == len(path)-1 {
//					return current[typedKey], true
//				} else {
//					return nil, false
//				}
//			case int:
//				array, ok := current[path[i-1].(string)].([]interface{})
//				if !ok || typedKey < 0 || typedKey >= len(array) {
//					return nil, false
//				}
//				if i == len(path)-1 {
//					return array[typedKey], true
//				}
//				next, ok := array[typedKey].(map[string]interface{})
//				if !ok {
//					return nil, false
//				}
//				current = next
//			default:
//				return nil, false
//			}
//		}
//		return nil, false
//	}
//
//	func setValueAtPath(m map[string]interface{}, path []interface{}, value interface{}) {
//		current := m
//		for i, key := range path {
//			switch typedKey := key.(type) {
//			case string:
//				if i == len(path)-1 {
//					current[typedKey] = value
//					return
//				}
//				if next, ok := current[typedKey].(map[string]interface{}); ok {
//					current = next
//				} else {
//					newMap := make(map[string]interface{})
//					current[typedKey] = newMap
//					current = newMap
//				}
//			case int:
//				arrayKey := path[i-1].(string)
//				array, ok := current[arrayKey].([]interface{})
//				if !ok {
//					array = make([]interface{}, typedKey+1)
//					current[arrayKey] = array
//				}
//				if typedKey >= len(array) {
//					newArray := make([]interface{}, typedKey+1)
//					copy(newArray, array)
//					array = newArray
//					current[arrayKey] = array
//				}
//				if i == len(path)-1 {
//					array[typedKey] = value
//					return
//				}
//				if next, ok := array[typedKey].(map[string]interface{}); ok {
//					current = next
//				} else {
//					newMap := make(map[string]interface{})
//					array[typedKey] = newMap
//					current = newMap
//				}
//			}
//		}
//	}
//
// Helper function to clean strings of backtick (`), double qoutes(""), and single qoutes (”)
// around the string
func cleanStrings(stringsToClean []string) []string {
	cleaned := make([]string, len(stringsToClean))
	for i, str := range stringsToClean {
		cleaned[i] = strings.NewReplacer(`"`, "", "`", "").Replace(str)
	}
	return cleaned
}

// Helper function for the replace in place. Parses the string with -> and array indexed strings
// Returns an array of []interface{} ["key1", "value", 0]
func parsePath(path string) ([]interface{}, error) {
	var keys []interface{}

	if len(path) == 0 {
		return keys, utils.ColorError("path should not be empty")
	}

	rawKeys := strings.Split(path, "->")
	for i, segment := range rawKeys {
		trimmedKey := strings.TrimSpace(segment)
		if trimmedKey == "" {
			msg := fmt.Sprintf("Invalid format: empty key at position %d", i+1)
			return nil, utils.ColorError(msg)
		}
		isArrayKey, keyPart, index := utils.ParseArrayKey(trimmedKey)
		if isArrayKey {
			keys = append(keys, keyPart)
			keys = append(keys, index)
		} else {
			keys = append(keys, trimmedKey)
		}
	}
	return keys, nil
}
