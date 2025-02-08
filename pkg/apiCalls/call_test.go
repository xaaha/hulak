package apicalls

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/yamlParser"
)

func TestCombineAndCall(t *testing.T) {
	testCases := []struct {
		name             string
		apiInfo          yamlParser.ApiInfo
		json             string
		expectedResponse yamlParser.ApiInfo
		expectedError    string
	}{
		{
			name:             "empty json string should result in nil yamlParser.ApiInfo",
			apiInfo:          yamlParser.ApiInfo{},
			json:             "",
			expectedResponse: yamlParser.ApiInfo{},
			expectedError:    "jsonString constructed from yamlFile is empty",
		},
		// TODO-2: Add enoded body using EncodeBody
		{
			name: "proper json string: with GraphQL body",
			apiInfo: yamlParser.ApiInfo{
				Url: "https://example.com/graphql",
				UrlParams: map[string]string{
					"baz": "bin",
					"foo": "bar",
				},
				Headers: map[string]string{
					"content-type": "applicaton/json",
				},
			},
			json: `
{
  "urlparams": {
    "baz": "bin",
    "foo": "bar"
  },
  "headers": {
    "content-type": "application/json"
  },
  "body": {
    "graphql": {
      "query": "query Hello {\n  hello(person: { name: Jane Doe, age: 22 })\n}\n"
    }
  },
  "method": "POST",
  "url": "https://example.com/graphql"
}
      `,
			expectedResponse: yamlParser.ApiInfo{
				Method: "POST",
				Url:    "https://example.com/graphql",
				UrlParams: map[string]string{
					"baz": "bin",
					"foo": "bar",
				},
				Headers: map[string]string{"content-type": "application/json"},
				Body: strings.NewReader(
					`{"query":"query Hello {\n  hello(person: { name: Jane Doe, age: 22 })\n}\n","variables":null}`,
				),
			},
			expectedError: "",
		},
		{
			name: "proper json string: without body",
			json: `
{
  "urlparams": {
    "baz": "bin",
    "foo": "bar"
  },
  "headers": {
    "content-type": "application/json"
  },
  "method": "POST",
  "url": "https://example.com/graphql"
}
      `,
			expectedResponse: yamlParser.ApiInfo{
				Method: "POST",
				Url:    "https://example.com/graphql",
				UrlParams: map[string]string{
					"baz": "bin",
					"foo": "bar",
				},
				Headers: map[string]string{"content-type": "application/json"},
				Body:    nil,
			},
			expectedError: "",
		},
		{
			name: "proper json string: without body and headers",
			json: `
{
  "urlparams": {
    "baz": "bin",
    "foo": "bar"
  },
  "method": "POST",
  "url": "https://example.com/graphql"
}
      `,
			expectedResponse: yamlParser.ApiInfo{
				Method: "POST",
				Url:    "https://example.com/graphql",
				UrlParams: map[string]string{
					"baz": "bin",
					"foo": "bar",
				},
				Headers: nil,
				Body:    nil,
			},
			expectedError: "",
		},
		{
			name: "proper json string: with url and method only",
			json: `
{
  "method": "POST",
  "url": "https://example.com/graphql"
}
      `,
			expectedResponse: yamlParser.ApiInfo{
				Method:    "POST",
				Url:       "https://example.com/graphql",
				UrlParams: nil,
				Headers:   nil,
				Body:      nil,
			},
			expectedError: "",
		},
		{
			name: "form data body with headers set",
			json: `
{
  "method": "POST",
  "url": "https://example.com/graphql",
  "body": {
    "formdata": {
      "baz": "bin",
      "foo": "bar"
    }
  }
}
      `,
			expectedResponse: yamlParser.ApiInfo{
				Method:    "POST",
				Url:       "https://example.com/graphql",
				UrlParams: nil,
				Headers:   map[string]string{"content-type": "multipart/form-data"},
				Body:      nil,
			},
			expectedError: "",
		},
	}
	// TODO: 1 - Remove PrepareStruct of ApiCalls. Use body instead of json and use the one from yamlParser
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiInfo, err := PrepareStruct(testCase.json)
			if testCase.expectedError != "" {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if !strings.Contains(err.Error(), testCase.expectedError) {
					t.Errorf(
						"Expected error to contain %q, but got %q",
						testCase.expectedError,
						err.Error(),
					)
				}
				return
			}

			var formData bool
			actualHeaders := apiInfo.Headers
			if contentType, ok := actualHeaders["content-type"]; ok &&
				strings.Contains(contentType, "multipart/form-data") {
				actualHeaders["content-type"] = "multipart/form-data"
				formData = true
			}

			// Compare all fields except Body
			expected := testCase.expectedResponse
			expected.Body = nil
			actual := apiInfo
			actual.Body = nil
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf(
					"Expected yamlParser.ApiInfo (except Body) to be \n%v, but got \n%v",
					expected, actual,
				)
			}

			// Compare Body content separately
			expectedBody, err1 := readBodyContent(testCase.expectedResponse.Body)
			actualBody, err2 := readBodyContent(apiInfo.Body)
			if formData {
				actualBody = "" // body has dynamic value for formData. So, setting it to empty string
			}
			if err1 != nil || err2 != nil {
				t.Fatalf(
					"Failed to read body content: expected error=%v, actual error=%v",
					err1,
					err2,
				)
			}
			if expectedBody != actualBody {
				t.Errorf(
					"Expected Body content to be \n%q, but got \n%q",
					expectedBody, actualBody,
				)
			}
		})
	}
}

// Helper function to read and return the content of an io.Reader
func readBodyContent(body io.Reader) (string, error) {
	if body == nil {
		return "", nil
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
