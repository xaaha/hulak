package yamlparser

import (
	"os"
	"os/exec"
	"testing"

	"github.com/goccy/go-yaml"
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

func TestHandleYamlFile(t *testing.T) {
	secretsMap := map[string]any{}
	tests := []struct {
		name      string
		content   string
		expectErr bool
	}{
		{
			name: "Valid YAML",
			content: `
KeyOne: value1
KeyTwo: value2
`,
			expectErr: false,
		},
		{
			name:      "Empty file",
			content:   "",
			expectErr: true,
		},
		{
			name:      "Non-existent file",
			content:   "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var filepath string
			var err error

			if !tc.expectErr || tc.name != "Non-existent file" {
				filepath, err = createTempYamlFile(tc.content)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(filepath)
			} else {
				filepath = "/non/existent/file.yaml"
			}

			if tc.expectErr {
				// Simulate child process to test os.Exit behavior
				if os.Getenv("EXPECT_EXIT") == "1" {
					_, _ = checkYamlFile(
						filepath,
						secretsMap,
					) // Call function that triggers os.Exit
					return
				}
				// handle the current subprocess
				cmd := exec.Command(os.Args[0], "-test.run="+t.Name())
				cmd.Env = append(os.Environ(), "EXPECT_EXIT=1")
				err := cmd.Run()

				// Verify exit code from the subprocess
				if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 1 {
					return // Test passes
				}
				t.Fatalf("Expected process to exit with code 1, but got %v", err)
			} else {
				buf, err := checkYamlFile(filepath, secretsMap)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if buf.Len() == 0 {
					t.Errorf("Expected non-empty buffer, got empty buffer for test %s", tc.name)
				}
			}
		})
	}
}

// tests the function that exists with invalid yaml file
func TestReadYamlForHttpRequest(t *testing.T) {
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
			expectErr: false,
		},
		{
			name: "Valid YAML with valid template",
			content: `
Method: GET
url: https://api.example.com/data
urlparams:
  key1: value1
headers:
  Accept: application/json
body:
  formdata:
    field1: "this is {{.sponsor}} body"
    field2: data2
`,
			expectErr: false,
		},
		{
			name: "Valid YAML: template without and without double quote",
			content: `
Method: GET
url: https://api.example.com/data
urlparams:
  key1: value1
headers:
  Accept: application/json
body:
  formdata:
    field1: this is {{.sponsor}} body
    field2: "{{.field2}}"
`,
			expectErr: false,
		},
		//		note: since yaml is essentially json under the hood, we need to wrap {{}} with ""
		// 	{
		// 		name: "Invalid YAML should pancic: Unexpected mapping key",
		// 		content: `
		// Method: GET
		// url: https://api.example.com/data
		// urlparams:
		//   key1: value1
		// headers:
		//   Accept: application/json
		// body:
		//   formdata:
		//     field1: this is {{.sponsor}} body
		//     field2: {{.field2}}
		// 	`,
		// 		expectErr: true,
		// 	},
	}

	secretsMap := map[string]any{
		"sponsor": "mastercard, visa, google",
		"field2":  "myRandomStringWith19382",
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filepath, err := createTempYamlFile(tc.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(filepath)

			result, _, err := FinalStructForAPI(filepath, secretsMap)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected an error for test %s, but got none", tc.name)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for test %s, but got: %v", tc.name, err)
				}

				if result.Method == "" || result.URL == "" {
					t.Errorf("Expected valid ApiCallFile struct but got empty fields for test %s", tc.name)
				}
			}
		})
	}
}

func TestCheckYamlFile(t *testing.T) {
	secretsMap := map[string]any{
		"sponsor": "mastercard, visa, google",
		"field2":  "myRandomStringWith19382",
		"number":  42,
		"boolean": true,
	}

	tests := []struct {
		name       string
		content    string
		secretMap  map[string]any
		expectErr  bool
		fileExists bool
	}{
		{
			name: "Valid YAML with simple data",
			content: `
method: post
url: https://api.example.com
headers:
  Content-Type: application/json
`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
		{
			name: "Valid YAML with template variables",
			content: `
method: post
url: https://api.example.com
headers:
  Content-Type: application/json
body:
  json:
    field1: "{{.sponsor}}"
    field2: "{{.field2}}"
`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
		{
			name: "Valid YAML with mixed case keys",
			content: `
Method: POST
URL: https://api.example.com
Headers:
  Content-Type: application/json
`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
		{
			name: "Valid YAML with nested structures",
			content: `
method: post
url: https://api.example.com
headers:
  Content-Type: application/json
body:
  json:
    nested:
      field1: value1
      field2: value2
    array:
      - item1
      - item2
`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
		{
			name: "Valid YAML with numeric and boolean values",
			content: `
method: post
url: https://api.example.com
headers:
  Content-Type: application/json
body:
  json:
    number: "{{.number}}"
    boolean: '{{.boolean}}'
`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
		{
			name:       "File does not exist",
			content:    "",
			secretMap:  secretsMap,
			expectErr:  true,
			fileExists: false,
		},
		{
			name:       "Empty file",
			content:    "",
			secretMap:  secretsMap,
			expectErr:  true,
			fileExists: true,
		},
		{
			name: "Invalid YAML: Unexpected mapping key",
			content: `
 Method: GET
 url: https://api.example.com/data
 urlparams:
   key1: value1
 headers:
   Accept: application/json
 body:
   formdata:
     field1: this is {{.sponsor}} body
     field2: {{.field2}}
		`,
			secretMap:  secretsMap,
			expectErr:  true,
			fileExists: true,
		},
		{
			name: "Missing required template variable",
			content: `
 method: post
 url: https://api.example.com
 headers:
   Content-Type: application/json
 body:
   json:
     field1: "{{.missing_variable}}"
		`,
			secretMap:  secretsMap,
			expectErr:  false,
			fileExists: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var filepath string
			var err error

			// Create the file or use a non-existent path
			if tc.fileExists {
				filepath, err = createTempYamlFile(tc.content)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(filepath)
			} else {
				filepath = "/non/existent/path/file.yaml"
			}

			if tc.expectErr {
				if os.Getenv("EXPECT_EXIT") == "1" {
					_, _ = checkYamlFile(filepath, tc.secretMap)
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
				// For cases where we don't expect an error/panic
				buf, err := checkYamlFile(filepath, tc.secretMap)
				if err != nil {
					t.Errorf("Expected no error for test %s, but got: %v", tc.name, err)
					return
				}

				if buf == nil {
					t.Errorf("Expected non-nil buffer for test %s", tc.name)
					return
				}

				// Verify that the buffer contains valid YAML
				var result map[string]any
				decoder := yaml.NewDecoder(buf)
				if err := decoder.Decode(&result); err != nil {
					t.Errorf("Failed to decode result buffer for test %s: %v", tc.name, err)
					return
				}

				// Additional validation for specific test cases
				if tc.name == "Valid YAML with mixed case keys" {
					// Check that keys were converted to lowercase
					if _, ok := result["method"]; !ok {
						t.Errorf("Expected lowercase 'method' key to exist in result")
					}
				}

				if tc.name == "Valid YAML with template variables" {
					// Verify template variables were replaced
					if body, ok := result["body"].(map[string]any); ok {
						if jsonData, ok := body["json"].(map[string]any); ok {
							if field1, ok := jsonData["field1"].(string); ok {
								if field1 != tc.secretMap["sponsor"] {
									t.Errorf("Expected field1 to be '%v', got '%v'", tc.secretMap["sponsor"], field1)
								}
							} else {
								t.Errorf("field1 is not a string or doesn't exist")
							}
						} else {
							t.Errorf("json data is not a map or doesn't exist")
						}
					} else {
						t.Errorf("body is not a map or doesn't exist")
					}
				}
			}
		})
	}
}
