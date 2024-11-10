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

func TestReadingYamlWithStruc(t *testing.T) {
	content := `
  ---
  method: post
  url: https://graphql.postman-echo.com/graphql
  urlparams:
    foo: bar
    baz: bin
  headers:
    Content-type: application/json
  body:
    # formdata:
    #   foo: bar
    #   baz: bin
    randomThing:
      foo: this is being ignored correctly
    # this should be invalid or just ignored
    graphql:
      query: |
        query Hello {
          hello(person: { name: "pratik", age: 11 })
        }
      variable:
        run: true
  `

	filepath, err := createTempYamlFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(filepath)
	// pos is invalid
	// post and POST is valid. Same with other methods like get patch...
	// missing url key in content is invalid
	// url is checked if it passes the url.ParseRequestURI(), if not then it's invalie
	// so, url or URL should be valid
	// but Url: this is my name is invalid. As the value is not a valid url
	// error when url, body, and method is invalid
	// may be check if the url, method is also missing
	// Body is only valid  if the key exists, and
	//  it includes one of these in the body type
	// variable in graphql is optional. If key and value is not provided, variable is only initialied with length of 0. not nill
	// graphql's query is required.
	// randomThing in body is safely ignored
	/*
	   type Body struct {
	   	FormData           map[string]string `json:"formdata,omitempty"           yaml:"formdata"`
	   	UrlEncodedFormData map[string]string `json:"urlencodedformdata,omitempty" yaml:"urlencodedformdata"`
	   	Graphql            *GraphQl          `json:"graphql,omitempty"            yaml:"graphql"`
	   	Raw                string            `json:"raw,omitempty"                yaml:"raw"`
	   }
	*/
}
