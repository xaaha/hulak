package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// LookupValue retrieves the value for a given key or path from the map.
// A key can be a simple string like "name", or a path like "user.name", where "user" is an object and "name" is a key inside it.
// Use {} to escape dots in keys (e.g., "user{.}name").
// For arrays, reference an index with square brackets (e.g., myArr[0] for the first element).
// You can also access nested properties like myArr[0].name for the "name" key of the first array element.
func LookupValue(key string, data map[string]interface{}) (string, error) {
	if value, exists := data[key]; exists {
		return MarshalToJSON(value)
	}

	pathSeparator := "."
	segments := parseKeySegments(key, pathSeparator)

	current := interface{}(data)

	for i, segment := range segments {
		isArrayKey, keyPart, index := parseArrayKey(segment)

		if isArrayKey {
			// Ensure the current value is a map so we can access the array key
			currMap, ok := current.(map[string]interface{})
			if !ok {
				return "", ColorError("current is not a map, cannot access key: " + keyPart)
			}

			// Retrieve the value associated with the keyPart (e.g., "myArr")
			value, exists := currMap[keyPart]
			if !exists {
				return "", ColorError("key not found: " + keyPart)
			}

			// Verify the retrieved value is a slice
			rv := reflect.ValueOf(value)
			if rv.Kind() != reflect.Slice {
				return "", ColorError("value is not an array: " + keyPart)
			}

			// Check index bounds
			if index < 0 || index >= rv.Len() {
				return "", ColorError("array index out of bounds: " + segment)
			}

			// Get the element at the specified index
			current = rv.Index(index).Interface()

			// If more segments remain, recurse
			if i < len(segments)-1 {
				remainingKey := strings.Join(segments[i+1:], pathSeparator)

				// Convert `current` to a map for further processing
				currMap, ok := current.(map[string]interface{})
				if !ok {
					currMap, ok = structToMap(current)
					if !ok {
						return "", ColorError(
							"invalid path, cannot process segment: " + remainingKey,
						)
					}
				}
				return LookupValue(remainingKey, currMap)
			}
		} else {
			// Treat as map key
			currMap, ok := current.(map[string]interface{})
			if !ok {
				// Handle struct conversion
				currMap, ok = structToMap(current)
				if !ok {
					return "", ColorError("invalid path, segment is not a map: " + strings.Join(segments[:i+1], pathSeparator))
				}
			}

			value, exists := currMap[segment]
			if !exists {
				return "", ColorError("key not found: " + strings.Join(segments[:i+1], pathSeparator))
			}
			current = value
		}

		// If this is the last segment, marshal and return the value
		if i == len(segments)-1 {
			return MarshalToJSON(current)
		}
	}

	return "", ColorError("unexpected error")
}

func structToMap(value interface{}) (map[string]interface{}, bool) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Struct {
		mapData := make(map[string]interface{})
		valType := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := valType.Field(i)
			mapData[field.Name] = val.Field(i).Interface()
		}
		return mapData, true
	}
	return nil, false
}

func parseKeySegments(key, pathSeparator string) []string {
	var segments []string
	current := strings.Builder{}
	inBracket := false

	for _, char := range key {
		switch {
		case char == '{':
			inBracket = true
		case char == '}':
			inBracket = false
			segments = append(segments, current.String())
			current.Reset()
		case char == rune(pathSeparator[0]) && !inBracket:
			segments = append(segments, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	// Append the last segment, if any
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}

func parseArrayKey(segment string) (bool, string, int) {
	if strings.HasSuffix(segment, "]") && strings.Contains(segment, "[") {
		openBracket := strings.LastIndex(segment, "[")
		closeBracket := strings.LastIndex(segment, "]")
		indexStr := segment[openBracket+1 : closeBracket]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return false, segment, -1 // Invalid index
		}
		return true, segment[:openBracket], index
	}
	return false, segment, -1
}

func MarshalToJSON(value interface{}) (string, error) {
	if arr, ok := value.([]interface{}); ok {
		var jsonArray []string
		for _, item := range arr {
			jsonStr, err := json.Marshal(item)
			if err != nil {
				return "", err
			}
			jsonArray = append(jsonArray, string(jsonStr))
		}
		return fmt.Sprintf("[%s]", strings.Join(jsonArray, ",")), nil
	}
	jsonStr, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}
