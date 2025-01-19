package utils

import (
	"reflect"
	"strconv"
	"strings"
)

// LookupValue retrieves the value for a given key or path from the map.
// A key can be a simple string like "name", or a path like "user.name", where "user" is an object and "name" is a key inside it.
// Use {} to escape dots in keys (e.g., "{user.name}").
// For arrays, reference an index with square brackets (e.g., myArr[0] for the first element).
// You can also access nested properties like myArr[0].name for the "name" key of the first array element.
func LookupValue(key string, data map[string]interface{}) (interface{}, error) {
	// Step 1: Check for direct key match
	if value, exists := data[key]; exists {
		return MarshalToJSON(value)
	}

	pathSeparator := "."
	// Step 2: Parse the key into segments
	segments := parseKeySegments(key, pathSeparator)

	// Step 3: Initialize the current context
	current := interface{}(data)

	// Step 4: Iterate through key segments
	for i, segment := range segments {
		isArrayKey, keyPart, index := ParseArrayKey(segment)

		// Step 5: Ensure current context is a map
		currMap, ok := current.(map[string]interface{})
		if !ok {
			currMap, ok = structToMap(current)
			if !ok {
				return "", ColorError(
					"invalid path, segment is not a map: " + strings.Join(
						segments[:i+1],
						pathSeparator,
					),
				)
			}
		}

		if isArrayKey {
			// Step 6: Handle array keys
			value, exists := currMap[keyPart]
			if !exists {
				return "", ColorError(KeyNotFound + keyPart)
			}

			rv := reflect.ValueOf(value)
			if rv.Kind() != reflect.Slice || index < 0 || index >= rv.Len() {
				return "", ColorError(IndexOutOfBounds + segment)
			}

			current = rv.Index(index).Interface()
		} else {
			// Step 7: Handle map keys
			value, exists := currMap[segment]
			if !exists {
				return "", ColorError(KeyNotFound + segment)
			}
			current = value
		}

		// Step 8: Check for the last segment
		if i == len(segments)-1 {
			return MarshalToJSON(current)
		}
	}

	// Step 9: Return error if unexpected
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

func ParseArrayKey(segment string) (bool, string, int) {
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
