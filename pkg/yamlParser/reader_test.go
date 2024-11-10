package yamlParser

import (
	"fmt"
	"testing"
)

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
  # variables:
  #   run: true
  `
	// pos is invalid
	// post and POST is valid
	// error when url, body, and method is invalid
	// may be check if the url, method is also missing
	fmt.Println(content)
}
