package apicalls

import (
	"reflect"
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
				// Body: strings.NewReader(
				// 	`{"query":"query Hello {\n  hello(person: { name: Jane Doe, age: 22 })\n}\n","variables":null}`,
				// ),
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			apiInfo := CombineAndCall(testCase.json)
			if !reflect.DeepEqual(apiInfo, testCase.expectedResponse) {
				t.Errorf(
					"Expected the ApiInfo struct to be \n %v, but got \n%v",
					testCase.expectedResponse,
					apiInfo,
				)
			}
		})
	}
}
