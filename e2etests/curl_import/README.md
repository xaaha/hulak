# cURL Import Test Examples

This directory contains expected output files for cURL import testing.

## Test Cases

### 1. Simple GET Request
**File**: `expected_simple_get.hk.yaml`

**Input cURL**:
```bash
curl https://jsonplaceholder.typicode.com/todos/1
```

### 2. POST with JSON Body
**File**: `expected_post_json.hk.yaml`

**Input cURL**:
```bash
curl -X POST https://jsonplaceholder.typicode.com/posts \
  -H "Content-Type: application/json" \
  -d '{"title":"foo","body":"bar","userId":1}'
```

### 3. POST with Form Data
**File**: `expected_form_data.hk.yaml`

**Input cURL**:
```bash
curl -X POST https://api.example.com/form \
  -F "username=john" \
  -F "password=secret123"
```

### 4. POST with GraphQL Query
**File**: `expected_graphql.hk.yaml`

**Input cURL**:
```bash
curl -X POST https://api.example.com/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"query Hello($name: String!) { hello(person: { name: $name }) }","variables":{"name":"John"}}'
```

## Running Tests

To verify these files are correctly formatted and can be executed:

```bash
# Test simple GET
hulak -fp e2etests/curl_import/expected_simple_get.hk.yaml

# Test POST with JSON
hulak -fp e2etests/curl_import/expected_post_json.hk.yaml

# Test form data
hulak -fp e2etests/curl_import/expected_form_data.hk.yaml

# Test GraphQL
hulak -fp e2etests/curl_import/expected_graphql.hk.yaml
```

## Notes

- All files follow the Hulak YAML schema
- JSON bodies are pretty-printed for readability
- GraphQL queries use multiline (`|`) syntax
- Form data and URL-encoded form data use their respective body types
