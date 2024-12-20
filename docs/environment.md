# Environment Secrets

- Environment secrets files, `global.env` (default), or `prod.env` (example) must exist in a folder which is in the root directory.

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

If `env/global.env` is missing, it's automatically created.

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
