# Import cURL Commands

Hulak can import cURL commands and convert them into `.hk.yaml` files, making it easy to share, version control, and organize your API requests.

## Table of Contents

- [Overview](#overview)
- [Usage](#usage)
- [Supported cURL Options](#supported-curl-options)
- [Output File Naming](#output-file-naming)
- [Examples](#examples)
- [Limitations](#limitations)
- [Tips](#tips)

## Overview

The `import curl` subcommand parses cURL command strings and generates Hulak-compatible YAML files. This is particularly useful when:

- Sharing API requests with team members
- Converting browser DevTools network requests to Hulak format
- Migrating from cURL-based workflows to Hulak
- Documenting API calls in a structured, version-controllable format

## Usage

### Basic Syntax

```bash
hulak import curl '<curl_command>' [-o output_path]
```

### Parameters

- `curl_command` (required): The cURL command string (must be quoted)
- `-o output_path` (optional): Custom output path for the generated `.hk.yaml` file

**Note**: The `-o` flag must come BEFORE the `curl` keyword.

### Output Behavior

**With `-o` flag:**
```bash
hulak import -o ./my-request.hk.yaml curl 'curl https://example.com'
```
- Creates file at specified path
- Automatically adds `.hk.yaml` extension if not provided
- Creates parent directories automatically
- Appends incremental number if file already exists (e.g., `file_1.hk.yaml`, `file_2.hk.yaml`)

**Without `-o` flag:**
```bash
hulak import curl 'curl https://example.com/users'
```
- Auto-generates filename in `imported/` directory
- Format: `METHOD_urlpart_timestamp.hk.yaml`
- Example: `GET_users_1767672792.hk.yaml`

## Supported cURL Options

Hulak supports the following cURL options:

### HTTP Methods
- `-X METHOD` or `--request METHOD`
- Supports: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT
- Defaults to GET if not specified
- Case-insensitive

### URL
- Required parameter
- Supports both quoted and unquoted URLs
- Query parameters are automatically extracted

### Headers
- `-H "Header: Value"` or `--header "Header: Value"`
- Multiple headers supported
- Example: `-H "Content-Type: application/json" -H "Authorization: Bearer token"`

### Request Body

**Raw Data:**
- `-d 'data'`, `--data 'data'`, `--data-raw 'data'`, `--data-binary 'data'`
- JSON bodies are automatically pretty-printed
- GraphQL queries are automatically detected and formatted

**Form Data (multipart/form-data):**
- `-F "key=value"` or `--form "key=value"`
- Multiple fields supported
- File uploads (`@filename`) are noted as TODO in output

**URL-encoded Form Data:**
- `--data-urlencode "key=value"`
- Also auto-detected from `-d` with `key=value` format

### Authentication

**Basic Auth:**
- `-u username:password` or `--user username:password"`
- Automatically converts to Base64-encoded Authorization header

**Cookies:**
- `--cookie "name=value"` or `-b "name=value"`
- Added as Cookie header

### Unsupported Flags (with warnings)

The following flags are not supported and will show warnings:
- `-k`, `--insecure`: Skip certificate verification
- `-L`, `--location`: Follow redirects
- `--compressed`: Request compressed response
- `-v`, `--verbose`: Verbose mode
- `-s`, `--silent`: Silent mode
- `-i`, `--include`: Include headers in output
- `-I`, `--head`: HEAD request method
- `--max-time`: Maximum time for request
- `--connect-timeout`: Connection timeout

## Output File Naming

### Auto-generated Names

Format: `METHOD_urlpart_timestamp.hk.yaml`

**Examples:**
- `curl https://api.example.com/users` → `GET_users_1767672792.hk.yaml`
- `curl -X POST https://api.example.com/posts` → `POST_posts_1767672815.hk.yaml`
- `curl https://jsonplaceholder.typicode.com/todos/1` → `GET_todos_1767672820.hk.yaml`

### Custom Names

```bash
# Specify full path with extension
hulak import -o ./requests/get-users.hk.yaml curl 'curl https://api.example.com/users'

# Extension added automatically
hulak import -o ./requests/get-users curl 'curl https://api.example.com/users'
# Creates: ./requests/get-users.hk.yaml

# Nested directories created automatically
hulak import -o ./api/v1/users/get.hk.yaml curl 'curl https://api.example.com/users'
```

## Examples

### 1. Simple GET Request

```bash
hulak import curl 'curl https://jsonplaceholder.typicode.com/todos/1'
```

**Output** (`imported/GET_todos_*.hk.yaml`):
```yaml
---
method: GET
url: "https://jsonplaceholder.typicode.com/todos/1"
```

### 2. GET with Query Parameters

```bash
hulak import curl 'curl "https://api.example.com/search?q=test&page=1&limit=10"'
```

**Output**:
```yaml
---
method: GET
url: "https://api.example.com/search"
urlparams:
  limit: "10"
  page: "1"
  q: test
```

### 3. POST with JSON Body

```bash
hulak import curl 'curl -X POST https://jsonplaceholder.typicode.com/posts \
  -H "Content-Type: application/json" \
  -d '"'"'{"title":"foo","body":"bar","userId":1}'"'"''
```

**Output**:
```yaml
---
method: POST
url: "https://jsonplaceholder.typicode.com/posts"
headers:
  Content-Type: application/json
body:
  raw: |
    {
      "body": "bar",
      "title": "foo",
      "userId": 1
    }
```

### 4. POST with Form Data

```bash
hulak import curl 'curl -X POST https://api.example.com/login \
  -F "username=john" \
  -F "password=secret123"'
```

**Output**:
```yaml
---
method: POST
url: "https://api.example.com/login"
body:
  formdata:
    password: secret123
    username: john
```

### 5. POST with URL-encoded Form Data

```bash
hulak import curl 'curl -X POST https://api.example.com/login \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=john&password=secret"'
```

**Output**:
```yaml
---
method: POST
url: "https://api.example.com/login"
headers:
  Content-Type: application/x-www-form-urlencoded
body:
  urlencodedformdata:
    password: secret
    username: john
```

### 6. GraphQL Query

```bash
hulak import curl 'curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '"'"'{"query":"query Hello($name: String!) { hello(person: { name: $name }) }","variables":{"name":"John"}}'"'"''
```

**Output**:
```yaml
---
method: POST
kind: GraphQL
url: "https://api.example.com/graphql"
headers:
  Content-Type: application/json
body:
  graphql:
    query: |
      query Hello($name: String!) { hello(person: { name: $name }) }
    variables:
      name: John
```

### 7. With Basic Authentication

```bash
hulak import curl 'curl -u username:password https://api.example.com/secure'
```

**Output**:
```yaml
---
method: GET
url: "https://api.example.com/secure"
headers:
  Authorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=
```

### 8. Multi-line cURL (from DevTools)

```bash
hulak import curl 'curl "https://api.example.com/data" \
  -H "accept: application/json" \
  -H "authorization: Bearer eyJhbGc..." \
  -H "user-agent: Mozilla/5.0" \
  --compressed'
```

**Output**:
```yaml
---
method: GET
url: "https://api.example.com/data"
headers:
  accept: application/json
  authorization: Bearer eyJhbGc...
  user-agent: Mozilla/5.0
```

*Note: `--compressed` flag shows a warning but is ignored.*

### 9. Custom Output Path

```bash
hulak import -o ./api/users/get-all.hk.yaml curl 'curl https://api.example.com/users'
```

Creates file at `./api/users/get-all.hk.yaml`

## Limitations

### Not Supported

1. **File Uploads**: Form fields with `@filename` are noted as TODO in the output
2. **Binary Data**: `--data-binary` with binary files
3. **Complex Authentication**: OAuth flows, client certificates
4. **Advanced Options**: Proxies, custom DNS, SSL options, connection options
5. **Redirect Following**: `-L` flag behavior
6. **Cookie Jars**: `--cookie-jar` for saving cookies

### Known Issues

1. **Nested JSON in Form Data**: Complex nested structures in form data may not parse correctly
2. **Escaped Characters**: Heavily escaped strings in cURL may need manual adjustment
3. **Environment Variables**: cURL commands with shell variables (`$VAR`) are imported as-is; you'll need to replace them with Hulak template syntax (`{{.VAR}}`) manually

## Tips

### From Browser DevTools

1. Open DevTools (F12)
2. Go to Network tab
3. Right-click on a request
4. Select "Copy" → "Copy as cURL"
5. Paste into Hulak import command

### Working with Multiline cURL

For readability, you can use backslashes:

```bash
hulak import curl 'curl -X POST https://api.example.com/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token" \
  -d '"'"'{"name":"John","age":30}'"'"''
```

### Quote Handling

Use single quotes for the outer string and escape inner quotes:

```bash
# Good
hulak import curl 'curl -d '"'"'{"key":"value"}'"'"' https://example.com'

# Also good (using double quotes and escaping)
hulak import curl "curl -d '{\"key\":\"value\"}' https://example.com"
```

### Converting to Environment Variables

After importing, you may want to replace sensitive data with environment variables:

**Imported:**
```yaml
headers:
  Authorization: Bearer eyJhbGc...
```

**After manual edit:**
```yaml
headers:
  Authorization: Bearer {{.apiToken}}
```

Then add `apiToken` to your `env/global.env` file.

### Testing Imported Files

After importing, test the file immediately:

```bash
hulak import curl 'curl https://api.example.com/users'
# Output: Created 'imported/GET_users_1767672792.hk.yaml' ✓

# Test it
hulak -fp imported/GET_users_1767672792.hk.yaml
```

## See Also

- [Body Documentation](./body.md) - Details on all body types
- [Actions Documentation](./actions.md) - Using template actions
- [Environment Documentation](./environment.md) - Managing secrets
