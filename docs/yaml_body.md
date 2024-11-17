# Yaml Body

## Data Structures

### HTTPMethodType

Defines the HTTP methods supported by Hulak.

Supported Methods:

• GET
• POST
• PUT
• PATCH
• DELETE
• HEAD
• OPTIONS
• TRACE
• CONNECT

### URL

Represents the URL for the API endpoint. The url must match go's `net/url` `ParseRequestURI`.
Dynamic query parameters can be included as key-value pairs under urlparams. For, example,

```yaml
url: "https://api.example.com/search"
urlparams:
  foo: "bar"
  check: true
method: GET
```

### Body

Represents the body of an HTTP request. Only one body type is allowed per request.

- FormData map[string]string Form data fields sent as multipart/form-data.
- UrlEncodedFormData map[string]string Data sent as application/x-www-form-urlencoded.
- Graphql \*GraphQl GraphQL queries and variables.
- Raw string Raw body content as a string.

Rules: 1. The body must not be empty or nil. 2. Only one body type must be present and non-empty. 3. If using Graphql, the query field must be provided.

#### GraphQl

Graphql must have query. Specifies the structure for GraphQL requests.

## Examples

Sample YAML Configuration

```yaml
url: "https://api.example.com/resource"
method: POST
urlparams:
	param1: "value1"
	param2: "value2"
headers:
	Content-Type: "application/json"
	Authorization: "Bearer token"
body:
	formdata:
		field1: "value1"
		field2: "value2"
```

```yaml
Method: GET
url: https://api.example.com/data
urlparams:
  key1: value1
headers:
  Accept: application/json
body:
  formdata:
    field1: "this is {{.secret}} body"
    field2: data2
```

- Hulak uses go template under the hood to replace your secrets. As seen above,
  if you want to replace the string with secrets, wrap the entire value with double quote `" "` in your yaml file.
  - For secrets, use dot/period `.` to reference a secret

```yaml
url: "{{.baseUrl}}" # Mandatory "" when we want to reference secret
method: POST
headers:
  Content-Type: application/json. # doouble quote is optional
body:
  graphql:
    query: |
      query getUser($id: ID!) {
        user(id: $id) {
          name
          email
        }
      }
    variables:
      id: "12345"
```

Key Rules:

1.  URL: Must be a valid, well-formed URI.
2.  Method: Must be one of the supported HTTP methods.
3.  Headers: Key-value pairs for HTTP headers.
4.  Body: Only one body type is allowed, and it must be valid.

Error Handling and Validation

```txt
  1.	HTTP Method Validation:
    •	Invalid methods will be rejected during validation.
  2.	URL Validation:
    •	If the URL is invalid, the request cannot proceed.
  3.	Body Validation:
    •	If multiple body types are present or none are valid, an error is thrown.
```
