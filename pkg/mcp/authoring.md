# Writing a hulak request file

A request file is one YAML document named `<name>.hk.yaml`. Its name is how every tool refers to it.

Top-level keys:

- `kind`: `API` (default), `GraphQL`, or `Auth` (OAuth 2.0 browser flow)
- `method`: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, `TRACE`, `CONNECT`
- `url`: required. Query parameters go in `urlparams`, not the url
- `urlparams`, `headers`: string maps
- `body`: exactly one of `raw`, `formdata`, `urlencodedformdata`, `graphql`
- `timeout`: Go duration, e.g. `30s`
- `auth`: only for `kind: Auth`

## Template actions

Values are Go templates. Quote any value that starts with `{{`.

- `{{.name}}`: the env variable `name` from the chosen environment. Case sensitive. Use `list_envs` for environment names and `list_requests` with `detail` for the variables a request uses.
- `{{getValueOf "key" "request_name"}}`: a value from the saved response of another request. The first argument is the JSON key (dot path, e.g. `data.token` or `items[0].id`). The second is the request name without extension, or a path to a `.json` file. The response must be saved first: over MCP, call that request with `call_request` and `save: true`.
- `{{getFile "path"}}`: the contents of a file, relative to the project root. `{{getFile "*.gql"}}` reads the sibling `.gql` with the same basename as the request.
- `{{attachFile "path"}}`: upload a file as a `formdata` part or as the whole `raw` body.
- `{{basicAuth .user .pass}}`: a complete `Basic <base64>` header value.
- `{{os "VAR"}}`: an operating system environment variable.

A bearer token is not added for you. Write the `Bearer ` prefix yourself.

## A REST request with env variables

```yaml
method: GET
url: "{{.base_url}}/users"
urlparams:
  page: "1"
headers:
  Accept: application/json
  Authorization: "Bearer {{.api_token}}"
```

## Chaining a token into another request

First a request named `get_token`:

```yaml
method: POST
url: "{{.auth_url}}/oauth/token"
body:
  urlencodedformdata:
    grant_type: client_credentials
    client_id: "{{.client_id}}"
    client_secret: "{{.client_secret}}"
```

Call it with `call_request` and `save: true`, then use its `access_token`:

```yaml
method: GET
url: "{{.base_url}}/me"
headers:
  Authorization: 'Bearer {{getValueOf "access_token" "get_token"}}'
```

## GraphQL

```yaml
kind: GraphQL
method: POST
url: "{{.graphql_url}}"
headers:
  Authorization: "Bearer {{.api_token}}"
body:
  graphql:
    query: '{{getFile "*.gql"}}'
    variables:
      id: "{{.user_id}}"
      first: 10
```

GraphQL `variables` keep their case. Every other key is case-insensitive.

## Raw and form bodies

```yaml
method: POST
url: "{{.base_url}}/posts"
headers:
  Content-Type: application/json
body:
  raw: '{"title": "hello", "userId": "{{.user_id}}"}'
```

```yaml
method: POST
url: "{{.base_url}}/upload"
body:
  formdata:
    description: profile photo
    file: '{{attachFile "assets/photo.png"}}'
```

## Before you call

Check a new request with `dry_run`: it resolves every variable without sending anything. Then send it with `call_request`.
