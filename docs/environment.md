# Environment Secrets

Environment secrets files, `global.env` (default), or `prod.env` (example) must exist in a folder which is in the root directory.

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

## Flags

| Flag   | Description                                                                                                     | Usage       |
| ------ | --------------------------------------------------------------------------------------------------------------- | ----------- |
| `-env` | Specify the environment file you want to use for Api Call. If the user flag is absent, it defaults to `global`. | `-env prod` |
