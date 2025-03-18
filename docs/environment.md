# Environment Secrets

- Environment secrets files, `global.env` (default), must exist in a folder `env/` which is in the root directory.
- Hulak asks user to create `global.env` during runtime, if it's not present already. You can also create the `global.env` manually
- Directory structure

```text
env/
  global.env
  prod.env
collection1/
  file1.yaml
  file2.yaml
  file3.yaml
collection2/
  file4.yml
  file5.yml
  file6.yml
```

- `<name>.env` file supports the following types:

  - `bool`,
  - `float64`,
  - `int`, and
  - `string`

  - To explicitly treat these values as a string, wrap the value in a double or single quote. For example,

```env
# Strings (quoted or unquoted)
graphqlUrl = "https://example.com/graphql"
baseUrl=https://example.com/
userName=xaaha
product = hulak

# Referencing another value
age={{.userAge}} # Referenced from below

# Numeric and boolean values
userAge = 18
userAgeAsString = "100.0"         # string
hasRunMarathon = false            # bool
hasRunMarathonAsString = "false"  # string
```

> [!Important]
> Since Hulak users go's template parsing under the hood, special characters besides underscore is not allowed in key.
> For example
>
> ```env
> "xaaha.userId" = "92n2a-2axaeix-9qnx9285x" ❌ Not allowed
> `user'sId` = "92n2a-2axaeix-9qnx9285x" ❌ Not allowed
> `client-id` = "92n2a-2axaeix-9qnx9285x" ❌ Not allowed
> # only underscores are allowed
> client_id = "92n2a-2axaeix-9qnx9285x" ✅ Allowed
> ```

- Use the secrets above

```yaml
body:
  graphql:
    query: |
      query Hello($name: String!, $age: Int) {
        hello(person: { name: $name, age: $age })
      }
    variables:
      name: "{{.userName}}"
      age: "{{.userAge}}" # userAge is int in this case.
```

> [!Tip]
> Since YAML does not support double curly braces ({{}}) without quotes, wrap values in backticks (`{{.key}}`), single quotes ('{{.key}}'), or double quotes ("{{.key}}"), to avoid issues.

## Flags

| Flag   | Description                                                                                                     | Usage       |
| ------ | --------------------------------------------------------------------------------------------------------------- | ----------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`. | `-env prod` |
