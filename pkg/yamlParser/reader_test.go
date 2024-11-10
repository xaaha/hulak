package yamlParser

import (
	"os"
	"reflect"
	"testing"

	yaml "github.com/goccy/go-yaml"
)

func createTempYamlFile(content string) (string, error) {
	file, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.Write([]byte(content)); err != nil {
		return "", nil
	}
	return file.Name(), nil
}

func deepEqualWithoutType(a, b interface{}) bool {
	switch a := a.(type) {
	case map[string]interface{}:
		bMap, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(a) != len(bMap) {
			return false
		}
		for k, v := range a {
			if !deepEqualWithoutType(v, bMap[k]) {
				return false
			}
		}
	case []interface{}:
		bSlice, ok := b.([]interface{})
		if !ok || len(a) != len(bSlice) {
			return false
		}
		for i := range a {
			if !deepEqualWithoutType(a[i], bSlice[i]) {
				return false
			}
		}
	default:
		return reflect.DeepEqual(a, b)
	}
	return true
}

func TestHandleYamlFile(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name: "Simple YAML structure",
			input: `
KeyOne: value1
KeyTwo: value2
`,
			expected: map[string]interface{}{
				"keyone": "value1",
				"keytwo": "value2",
			},
		},
		{
			name: "Nested YAML structure",
			input: `
KeyOuter:
  KeyInner: innerValue
AnotherKey: anotherValue
`,
			expected: map[string]interface{}{
				"keyouter": map[string]interface{}{
					"keyinner": "innerValue",
				},
				"anotherkey": "anotherValue",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := createTempYamlFile(tc.input)
			if err != nil {
				t.Fatalf("Failed to create temp env file: %v", err)
			}
			defer os.Remove(tmpFile)
			buf, _ := handleYamlFile(tmpFile)
			var result map[string]interface{}
			if err := yaml.NewDecoder(buf).Decode(&result); err != nil {
				t.Fatalf("Failed to decode YAML from buffer: %v", err)
			}
			// Compare result with expected output
			if !deepEqualWithoutType(result, tc.expected) {
				t.Errorf("Test %s failed. Expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func TestReadingYamlWithStruct(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		expectErr bool
	}{
		{
			name: "Valid YAML with POST method, valid URL, and GraphQL body",
			content: `
method: post
url: https://graphql.postman-echo.com/graphql
headers:
  Content-Type: application/json
body:
  graphql:
    query: |
      query Hello {
        hello(person: { name: "pratik", age: 11 })
      }
`,
			expectErr: false,
		},
		{
			name: "Valid YAML with GET method and FormData body",
			content: `
method: GET
url: https://api.example.com/data
urlparams:
  key1: value1
headers:
  Accept: application/json
body:
  formdata:
    field1: data1
    field2: data2
`,
			expectErr: false,
		},
		// 		{
		// 			name: "Invalid YAML with missing URL",
		// 			content: `
		// method: POST
		// headers:
		//   Content-Type: application/json
		// body:
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Invalid URL in YAML",
		// 			content: `
		// method: POST
		// url: "invalid-url"
		// headers:
		//   Content-Type: application/json
		// body:
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Invalid HTTP Method",
		// 			content: `
		// method: INVALID
		// url: https://api.example.com/data
		// body:
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Missing HTTP Method",
		// 			content: `
		// url: https://api.example.com/data
		// body:
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Missing body",
		// 			content: `
		// method: POST
		// url: https://api.example.com/data
		// headers:
		//   Content-Type: application/json
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Invalid body structure",
		// 			content: `
		// method: POST
		// url: https://api.example.com/data
		// body:
		//   randomKey:
		//     field1: data1
		// `,
		// 			expectErr: true,
		// 		},
		// 		{
		// 			name: "Optional GraphQL variable in body",
		// 			content: `
		// method: POST
		// url: https://graphql.example.com
		// body:
		//   graphql:
		//     query: |
		//       query ExampleQuery { example }
		// `,
		// 			expectErr: false,
		// 		},
		// 		{
		// 			name: "Uppercase HTTP Method",
		// 			content: `
		// method: GET
		// url: https://api.example.com/data
		// body:
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: false,
		// 		},
		// 		{
		// 			name: "FormData and GraphQL mixed in body (invalid)",
		// 			content: `
		// method: POST
		// url: https://api.example.com/data
		// body:
		//   formdata:
		//     field1: data1
		//   graphql:
		//     query: |
		//       query Test {
		//         test
		//       }
		// `,
		// 			expectErr: true,
		// 		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filepath, err := createTempYamlFile(tc.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(filepath)

			// Defer a function to recover from panic
			defer func() {
				if r := recover(); r != nil {
					if !tc.expectErr {
						t.Errorf("Unexpected panic for test %s: %v", tc.name, r)
					}
				} else if tc.expectErr {
					t.Errorf("Expected panic but got none for test %s", tc.name)
				}
			}()

			// Call the function that may panic
			result := ReadYamlForHttpRequest(filepath)
			if !tc.expectErr && result == "" {
				t.Errorf("Expected result but got empty string for test %s", tc.name)
			}
		})
	}
}
