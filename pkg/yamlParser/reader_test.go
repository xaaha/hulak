package yamlParser

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
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
		name      string
		input     string
		expected  map[string]interface{}
		expectErr bool
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
			expectErr: false,
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
			expectErr: false,
		},
		{
			name:      "Non-existent file",
			input:     "",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var tmpFile string
			var err error

			if !tc.expectErr {
				tmpFile, err = createTempYamlFile(tc.input)
				if err != nil {
					t.Fatalf("Failed to create temp YAML file: %v", err)
				}
				defer os.Remove(tmpFile)
			} else {
				tmpFile = "/non/existent/file.yaml"
			}

			defer func() {
				if r := recover(); r != nil {
					if tc.expectErr {
						expectedMsg := "File does not exist"
						if !strings.Contains(fmt.Sprintf("%v", r), expectedMsg) {
							t.Errorf(
								"Expected panic with message '%s', but got: %v",
								expectedMsg,
								r,
							)
						}
					} else {
						t.Errorf("Unexpected panic: %v", r)
					}
				} else if tc.expectErr {
					t.Errorf("Expected a panic but did not get one")
				}
			}()

			buf, _ := handleYamlFile(tmpFile)
			if !tc.expectErr {
				var result map[string]interface{}
				if err := yaml.NewDecoder(buf).Decode(&result); err != nil {
					t.Fatalf("Failed to decode YAML from buffer: %v", err)
				}

				// Compare result with expected output
				if !deepEqualWithoutType(result, tc.expected) {
					t.Errorf("Test %s failed. Expected %v, got %v", tc.name, tc.expected, result)
				}
			}
		})
	}
}

// tests the function that exists with invalid yaml file
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
		{
			name: "Invalid YAML with missing URL",
			content: `
method: POST
headers:
  Content-Type: application/json
body:
  graphql:
    query: |
      query Test {
        test
      }
`,
			expectErr: true,
		},
		{
			name: "Invalid HTTP Method",
			content: `
method: INVALID
url: https://api.example.com/data
body:
  graphql:
    query: |
      query Test {
        test
      }
`,
			expectErr: true,
		},
		{
			name: "Missing body",
			content: `
method: POST
url: https://api.example.com/data
headers:
  Content-Type: application/json
`,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filepath, err := createTempYamlFile(tc.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(filepath)

			if tc.expectErr {
				if os.Getenv("EXPECT_EXIT") == "1" {
					ReadYamlForHttpRequest(filepath)
					return
				}

				cmd := exec.Command(os.Args[0], "-test.run="+t.Name())
				cmd.Env = append(os.Environ(), "EXPECT_EXIT=1")
				err := cmd.Run()

				if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
					return // Expected exit, test passes
				}
				t.Fatalf("Expected process to exit with code 1, but got %v", err)
			} else {
				result := ReadYamlForHttpRequest(filepath)
				if result == "" {
					t.Errorf("Expected result but got empty string for test %s", tc.name)
				}
			}
		})
	}
}
