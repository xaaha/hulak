package apicalls

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestCombineAndCall(t *testing.T) {
	testCases := []struct {
		name             string
		json             string
		expectedResponse ApiInfo
	}{
		{
			name:             "empty json string should result in nil ApiInfo",
			json:             "",
			expectedResponse: ApiInfo{},
		},
		{
			name: "proper json string",
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
			expectedResponse: ApiInfo{
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
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiInfo := CombineAndCall(testCase.json)

			// Compare fields except `Body` using a shallow copy of `expectedResponse`
			expected := testCase.expectedResponse
			expected.Body = nil
			actual := apiInfo
			actual.Body = nil

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf(
					"Expected ApiInfo (except Body) to be \n%v, but got \n%v",
					expected, actual,
				)
			}

			// Compare the `Body` content separately
			expectedBody, err1 := readBodyContent(testCase.expectedResponse.Body)
			actualBody, err2 := readBodyContent(apiInfo.Body)
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
