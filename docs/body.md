# Body

This section includes all the key-value structures that can be passed in api (yaml) file.

For the dedicated GraphQL explorer flow, see [graphql-explorer.md](./graphql-explorer.md).

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

```yaml
method: GET # case insensitive
```

### URL

Represents the URL for the API endpoint. The url must match go's `net/url` `ParseRequestURI`.
Dynamic query parameters can be included as key-value pairs under urlparams. For, example,

```yaml
method: GET
url: "https://jsonplaceholder.typicode.com/todos/1"
urlparams:
  foo: "bar"
  check: true
```

### Timeout

Optional per-request timeout. When unset, Hulak falls back to the `--timeout` flag, then `$HULAK_TIMEOUT`, then a 60-second default.

```yaml
method: GET
url: "https://slow.example.com/big-export"
timeout: 5m
```

Accepts any [Go duration string](https://pkg.go.dev/time#ParseDuration): `30s`, `90s`, `5m`, `2m30s`, `1h`. Bare numbers (`60`) are rejected. Always include the unit.

#### Precedence

When more than one source could time-out a request, Hulak picks the highest-precedence source that is set:

1. **YAML `timeout:` field**. Wins over everything else for this file.
2. **`--timeout <duration>` flag**. Fallback for files with no YAML override. Works on `hulak run`, `hulak -fp`, and `hulak -dir`.
3. **`HULAK_TIMEOUT` env var**. Same scope as the flag, scoped to the shell session. Useful for slow VPNs or one-off runs without retyping.
4. **60-second default**. When nothing above is set.

A non-zero value at any layer overrides every layer below it; an unset / zero layer falls through to the next.

#### Single-file runs

```bash
hulak run requests/big-export.hk.yaml                    # uses YAML timeout if set, else 60s
hulak run requests/big-export.hk.yaml --timeout 10m      # 10m unless the YAML overrides it
HULAK_TIMEOUT=2m hulak run requests/big-export.hk.yaml   # 2m unless YAML or --timeout overrides
hulak -fp requests/big-export.hk.yaml --timeout 10m      # same precedence on the root-flag form
```

If the YAML has `timeout: 30s`, the file takes 30s no matter what `--timeout` or `HULAK_TIMEOUT` say.

#### Directory / bulk runs

Each file's YAML wins for that file. `--timeout` and `HULAK_TIMEOUT` act as the per-file fallback for files with no `timeout:` field. Resolution is per-file: in a directory of ten files where two set `timeout: 5m` in YAML, those two get 5 minutes and the other eight get the resolved fallback.

```bash
hulak run requests/ --timeout 90s          # 90s for files without YAML timeout, YAML still wins per-file
hulak run requests/ --sequential --timeout 90s
hulak -dir ./requests --timeout 90s        # root-flag form, concurrent
hulak -dirseq ./requests --timeout 90s     # root-flag form, sequential
```

Files in concurrent mode (`-dir` / `hulak run <dir>`) each get their own resolved timeout independently. One slow file does not steal the budget from the next. Sequential mode (`-dirseq` / `--sequential`) applies the same per-file resolution one file at a time.

#### Errors

Hulak validates timeouts up front so a typo cannot silently fall back to the default:

- A YAML `timeout:` value that is not a valid positive Go duration fails that file with `in <path>: invalid timeout "<value>": ...` before any request goes out. Other files in the run are unaffected.
- An invalid `HULAK_TIMEOUT` aborts the whole run before any request work begins.
- An invalid `--timeout` is rejected by Go's `flag` package the same way as any other duration flag.

> [!Note]
>
> 1. `timeout: 0s` and negative durations are rejected. There is no "no timeout" mode. Set a value high enough for the slowest request you expect.
> 2. The timeout covers the whole request (connect + send + read). It is not a connect-only or read-only timeout.

### Body

Represents the body of an HTTP request. Only one body type is allowed per request.

- FormData `map[string]string` Form data fields sent as multipart/form-data.
- UrlEncodedFormData `map[string]string` Data sent as application/x-www-form-urlencoded.
- Graphql: With GraphQL queries and variables.
- Raw string Raw body content as a string.

> [!Note]
>
> 1. The body must not be empty or nil.
> 2. Only one body type must be present and non-empty.
> 3. If using Graphql, the query field must be provided.
> 4. `kind: GraphQL` is recommended for GraphQL files, and required for GraphQL directory discovery in `hulak gql <directory>`.

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
	Authorization: "Bearer {{.token}}"
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
    field1: this is {{.secret}} body
    field2: data2
```

```yaml
Method: GET
url: "{{.baseUrl}}" # Remember to wrap the secret decleration with double if that's the only string
urlparams:
  key1: this is {{.secret}} also valid
headers:
  Accept: application/json
body:
  formdata:
    field1: this is {{.secret}} body
    field2: data2
```

- Hulak uses `Go's` template under the hood to replace your secrets. As seen above,
  if you want to replace the string with secrets, entire secret with double quote `" "` in your yaml file.
  - For secrets, use dot/period `.` to reference a secret
  - Graphql variables that needs `Int!`, `Boolean` or other types are automtically converted based on their original type

```yaml
url: "{{.baseUrl}}" # Mandatory "" when we want to reference secret
method: POST
kind: GraphQL
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
      id: "{{.userId}}" # if userId is an Int in secets map, then this id will also be automtically converted to an int
```

> [!Note]
>
> 1.  URL: Must be a valid, well-formed URI.
> 2.  Method: Must be one of the supported HTTP methods.
> 3.  Headers: Key-value pairs for HTTP headers. Most of the headers are listed in [headers documentation](./headers.yaml)
> 4.  Body: Only one body type is allowed, and it must be valid.
> 5.  Secrets are allowed with `{{.secretName}}` but make sure formatting is right

## GraphQL Explorer Source Files

The GraphQL explorer can also start from lightweight schema source files.

These files are usually smaller than normal request files. They are used to discover schema operations and then generate reusable `.gql` and `.hk.yaml` files later.

Example:

```yaml
---
kind: GraphQL
url: "{{.graphqlUrl}}"
headers:
  Content-Type: application/json
```

Read the full explorer flow in [graphql-explorer.md](./graphql-explorer.md).
