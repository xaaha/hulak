package yamlParser

import (
	"os"
	"os/exec"
	"testing"
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
	secretsMap := map[string]interface{}{}
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
		// note: since yaml is essentially json under the hood, we need to wrap {{}} with ""
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
			expectErr: true,
		},
	}

	secretsMap := map[string]interface{}{
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

			if tc.expectErr {
				if os.Getenv("EXPECT_EXIT") == "1" {
					FinalStructForApi(filepath, secretsMap)
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
				result := FinalStructForApi(filepath, secretsMap)
				if result == "" {
					t.Errorf("Expected result but got empty string for test %s", tc.name)
				}
			}
		})
	}
}
