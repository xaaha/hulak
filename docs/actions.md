# How to access variables?

## 1. Accessing secrets from the vault

Hulak resolves `{{.key}}` template values from the encrypted vault (`.hulak/store.age`). Set a secret once with `hulak secrets keys set`. Then reference it in your request files.

```bash
# Set once, per environment
hulak secrets keys set Url https://example.com         --env prod
hulak secrets keys set Url https://test.example.com    --env staging
```

Reference it in a request file:

```yaml
# test.hk.yaml
method: post
url: "{{.Url}}"
# other body items
```

Pick the environment at run time:

```bash
hulak run test.hk.yaml --env prod      # uses prod Url
hulak run test.hk.yaml --env staging   # uses staging Url
```

> [!Note]
>
> - If the call is made with `--env prod`, then the key should be defined in either the `global` or the `prod` environment. The same rule applies to other environments. If the key is in both, `global` is ignored.
> - Keys are case sensitive. So `{{.Url}}` and `{{.url}}` are two different values.

> Using the classic plaintext `env/` backend? See [environment.md](./environment.md).

## 2. Using `getValueOf`

```yaml
# example
url: `{{getValueOf "key" "file_name" }}`
```

`getValueOf` looks for the value of the `key` inside the `file_name.json` file. Since responses of the api requests are saved in `file_name_response.json` file in the same directory, you don't need to provide `_response.json` suffix when using `getValueOf`. If multiple `file_name.json` is found, hulak recurces through the directory and uses the first file match. So, it is recommended that you use a unique name for each file.

- `"key"` and `"file_name"`: Should be surrounded by double quotes (Go template).

```yaml
# where name is the key in the file user.json
name: '{{getValueOf "name" "user.json"}}'
# name is inside the user object in the user file
name: '{{getValueOf "user.name" "user"}}' # same as user.json
```

### Nested Object handling

If the key is in a nested object, `getValueof` can still retrieve the value using dot notation or array indexing. Here are some more examples

```json
{
  "company.info": "Earth Company",
  "name": "xaaha",
  "age": 100,
  "years": 111,
  "marathon": false,
  "profession": {
    "company.info": "Earth Based Human Led",
    "title": "Senior Human",
    "years": 5
  },
  "myArr": [
    { "Name": "xaaha", "Age": 222, "Years": 3989 },
    { "Name": "pt", "Age": 352, "Years": 8889 }
  ]
}
```

- To access a simple key

```yaml
years: '{{getValueOf "years" "example"}}' # gets the 111
name: '{{getValueOf "{company.info}" "example"}}' # gets Earth Company
age: '{{getValueOf "age" "example"}}' # gets 100
```

- To access company info inside the profession object

```yaml
# get the company.info value inside the profession object of example.json file
profession: '{{getValueOf "profession.{company.info}" "example.json"}}' # gets Earth Based Human Led
```

- To get value inside an array

```yaml
# get's the Name value "pt" from example.json above
employee: '{{getValueOf "myArr[1].Name" "example.json"}}' # gets "pt"
# if the example.josn file is an array start with indexig [0]
employee: '{{getValueOf "[0].company.Name" "example.json"}}' # gets "pt"
```

### Using `path` or `file_name`

While providing the `file_name` is easier, sometimes it is preferred to provide full path to the file you want to use. Especially if there are multiple files with the same name.
In such case use the path to the file from the root of the project.

```yaml
name: '{{getValueOf "user.name" "./users/profiles.json"}}'
```

## 2. Using `getFile`

Gets the file content as string and dumps it in context. It takes one argument, file path. So, provide either the file path from the root of the project

```yaml
# example
body:
  graphql:
    query: '{{getFile "e2etests/test_collection/test.graphql"}}'
```

OR provide the full path

```yaml
body:
  graphql:
    query: '{{getFile "/Users/yourname/Documents/Projects/hulak/e2etests/test_collection/graphql.yml"}}'
```

`getFile` gets the entire file content and dumps it in context. For example, in the above example, it dumps the content in the query section of grapqhl

## 3. Using `basicAuth`

Generates a `Basic` authentication header value. It takes a username and password, joins them with a colon, base64-encodes the result, and returns the full header value `Basic <encoded>`.

```yaml
headers:
  Authorization: '{{basicAuth "admin" "secret123"}}'
```

This produces `Basic YWRtaW46c2VjcmV0MTIz`. No manual base64 encoding needed.

### Using with environment variables

Store credentials in the vault and reference them with template vars:

```bash
hulak secrets keys set apiUser admin       --env prod
hulak secrets keys set apiPassword secret123 --env prod
```

```yaml
# request.hk.yaml
method: GET
url: https://api.example.com/protected
headers:
  Authorization: "{{basicAuth .apiUser .apiPassword}}"
```

```bash
hulak run request.hk.yaml --env prod
```

This works the same as `curl -u admin:secret123 https://api.example.com/protected`.

> Classic env/ users: put the keys in `env/prod.env` as `apiUser = admin` and `apiPassword = secret123` instead. See [environment.md](./environment.md).

## 4. Using `os`

Reads an OS environment variable at template execution time. Takes a single argument, the variable name, and returns its value. Returns an empty string if unset.

```yaml
# request.hk.yaml
method: GET
url: https://api.example.com
headers:
  Authorization: 'Bearer {{os "GITHUB_TOKEN"}}'
```

```bash
export GITHUB_TOKEN=ghp_abc123
hulak -f request
```

This is useful for secrets that live in your shell environment. CI tokens or credentials injected by your platform are common examples. The vault store and `env/` files are the other two places hulak looks for values.

### Combining with store variables

`os` can be used alongside `{{.Var}}` references in the same template:

```yaml
url: '{{.BASE_URL}}/callback?token={{os "SESSION_TOKEN"}}'
```

### Behaviour notes

- Variable names are **case sensitive**. `{{os "path"}}` and `{{os "PATH"}}` are different
- Returns an **empty string** if the variable is not set (no error)
- Does **not** trigger env file loading. It reads directly from the OS environment.
