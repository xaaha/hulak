package yamlParser

import (
	"fmt"
	"os"
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
  `

	fmt.Println(content)
	filepath, err := createTempYamlFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(filepath)
	// pos is invalid
	// post and POST is valid
	// missing url in content is invalid
	// url is checked if it passes the url.ParseRequestURI(), if not then it's invalie
	// so, url or URL should be valid
	// error when url, body, and method is invalid
	// may be check if the url, method is also missing
}
