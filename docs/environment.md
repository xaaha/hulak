# Environment Secrets

- Environment secrets files, `global.env` (default), or `prod.env` (example) must exist in a folder which is in the root directory.
- `global.env` is created automatically when running an API (yaml file) or you can create the global.env manually
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

- `global.env` supports the following type

  - `bool`, `float64`, `int`, and `string`
  - Wrap these values in double or single quote if you want to treat them as string

```env
graphqlUrl="https://example.com/graphql" # this is a string
baseUrl=https://example.com/ # this is also a string
userName=xaaha
product=hulak
userAge={{.myAwesomeNumber}} # referenced from the value below
myAwesomeNumber=22 # treated as int
myAwesomeNumberString="100.0" # treated as string
myAwesomeNumberbool=false # this is bool
myAwesomeNumberboolString="false" # this is a string
```

- All the values of the environment is treated as a string. If you need to use an environment variable as `int`, `bool`, `float` or `null` use yaml's explicit typing. See the [yaml documentation](https://yaml.org/spec/1.2.2/#chapter-10-recommended-schemas) for more details.

```yaml
body:
  graphql:
    query: |
      query Hello($name: String!, $age: Int) {
        hello(person: { name: $name, age: $age })
      }
    variables:
      name: "{{.userName}}"
      age: !!int "{{.userAge}}" # userAge is treated as an int with explicit typing
```

## Flags

| Flag   | Description                                                                                                     | Usage       |
| ------ | --------------------------------------------------------------------------------------------------------------- | ----------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`. | `-env prod` |
