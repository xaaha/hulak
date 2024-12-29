package utils

import (
	"strings"
	"testing"
)

func TestGetValueOf(t *testing.T) {
	myDict := map[string]interface{}{
		"company.inc": "Test Company",
		"name":        "Pratik",
		"age":         32,
		"years":       111,
		"marathon":    false,
		"profession": map[string]interface{}{
			"company.info": "Earth Based Human Led",
			"title":        "Senior Test SE",
			"years":        5,
		},
		"myArr": []interface{}{
			map[string]interface{}{"Name": "xaaha", "Age": 22, "Years": 11},
			map[string]interface{}{"Name": "pt", "Age": 35, "Years": 88},
		},
		"myArr2": []interface{}{
			map[string]interface{}{"Name": "xaaha", "Age": 22, "Years": 11},
			map[string]interface{}{"Name": "pt", "Age": 35, "Years": 88},
		},
	}

	tests := []struct {
		key          string
		expected     string
		errorMessage string
		expectErr    bool
	}{
		{"age", "32", "", false},
		{"marathon", "false", "", false},
		{
			"profession",
			`{"company.info":"Earth Based Human Led","title":"Senior Test SE","years":5}`,
			"",
			false,
		},
		{
			"myArr2",
			`[{"Age":22,"Name":"xaaha","Years":11},{"Age":35,"Name":"pt","Years":88}]`,
			"",
			false,
		},
		{"{company.inc}", `"Test Company"`, "", false},
		{"profession.{company.info}", `"Earth Based Human Led"`, "", false},
		{"myArr[1]", `{"Age":35,"Name":"pt","Years":88}`, "", false},
		{"myArr[1].Name", `"pt"`, "", false},
		{"myArr[10]", "", IndexOutOfBounds + "myArr[10]", true},
		{"nonexistentKey", "", KeyNotFound + "nonexistentKey", true},
		{"myArr[0].InvalidKey", "", KeyNotFound + "InvalidKey", true},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			result, err := LookupValue(test.key, myDict)

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
