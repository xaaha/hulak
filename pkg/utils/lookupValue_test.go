package utils

import (
	"strings"
	"testing"
)

func TestGetValueOf(t *testing.T) {
	myDict := map[string]any{
		"company.inc": "Test Company",
		"name":        "Pratik",
		"age":         32,
		"years":       111,
		"marathon":    false,
		"profession": map[string]any{
			"company.info": "Earth Based Human Led",
			"title":        "Senior Test SE",
			"years":        5,
		},
		"myArr": []any{
			map[string]any{"Name": "xaaha", "Age": 22, "Years": 11},
			map[string]any{"Name": "pt", "Age": 35, "Years": 88},
		},
		"myArr2": []any{
			map[string]any{"Name": "xaaha", "Age": 22, "Years": 11},
			map[string]any{"Name": "pt", "Age": 35, "Years": 88},
		},
		// Adding a root-level array as another entry in the map
		"rootArray": []any{
			map[string]any{
				"info": map[string]any{
					"name": "xaaha",
				},
			},
			map[string]any{},
		},
	}

	// Create a separate array to test direct array access
	rootLevelArray := []any{
		map[string]any{
			"info": map[string]any{
				"name": "xaaha",
			},
		},
		map[string]any{},
	}

	tests := []struct {
		key          string
		expected     any
		errorMessage string
		expectErr    bool
		useRootArray bool // Flag to indicate whether to use rootLevelArray
	}{
		{"age", 32, "", false, false},
		{"marathon", false, "", false, false},
		{
			"profession",
			`{"company.info":"Earth Based Human Led","title":"Senior Test SE","years":5}`,
			"",
			false,
			false,
		},
		{
			"myArr2",
			`[{"Age":22,"Name":"xaaha","Years":11},{"Age":35,"Name":"pt","Years":88}]`,
			"",
			false,
			false,
		},
		{"{company.inc}", "Test Company", "", false, false},
		{"profession.{company.info}", "Earth Based Human Led", "", false, false},
		{"myArr[1]", `{"Age":35,"Name":"pt","Years":88}`, "", false, false},
		{"myArr[1].Name", "pt", "", false, false},
		{"myArr[10]", "", IndexOutOfBounds + "myArr[10]", true, false},
		{"nonexistentKey", "", KeyNotFound + "nonexistentKey", true, false},
		{"myArr[0].InvalidKey", "", KeyNotFound + "InvalidKey", true, false},
		// New test cases for a nested array in a map
		{"rootArray[0].info.name", "xaaha", "", false, false},
		{"rootArray[1]", `{}`, "", false, false},
		// Test cases for direct array access where the array itself is the root
		{"[0].info.name", "xaaha", "", false, true},
		{"[1]", `{}`, "", false, true},
		{"[2]", "", IndexOutOfBounds + "[2]", true, true},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			var result any
			var err error

			if test.useRootArray {
				// Convert the array to map for LookupValue
				arrayAsMap := map[string]any{
					"": rootLevelArray,
				}
				result, err = LookupValue(test.key, arrayAsMap)
			} else {
				result, err = LookupValue(test.key, myDict)
			}

			if test.expectErr {
				if err == nil {
					t.Fatalf("expected error but got none for key: %s", test.key)
				}
				if !strings.Contains(err.Error(), test.errorMessage) {
					t.Fatalf(
						"expected error message '%s' but got '%s'",
						test.errorMessage,
						err.Error(),
					)
				}
			} else {
				if err != nil {
					t.Fatalf("did not expect error but got: %s for key: %s", err.Error(), test.key)
				}
				if result != test.expected {
					t.Fatalf("expected result '%s' but got '%s' for key: %s", test.expected, result, test.key)
				}
			}
		})
	}
}
