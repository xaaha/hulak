# How to access variables?

## 1. Acccessing secrets from `.env` file

Hulak uses go's templating to access variables from `.env` files. For example, to access a `Url` value use `"{{.Url}}"`

```yaml
#test.yaml
Method: post
url: "{{.Url}}"
# other body items
```

So, if `Url` is present in `prod.env` file as

```env
Url = https://example.com
```

and is present in `staging.env` file as

```env
Url = https://test.example.com
```

When running the `test.yaml` file, hulak picks up the `Url` value depending on the value of `-env` flag. For example,

```bash
hulak -env prod -f test # picks up the value from `prod.env` file,
hulak -env staging -f test # picks up the value from `staging.env` file
```

> [!Note]
>
> - If the call is made with `-env prod`, then `Key` should be defined in either `global` or `prod` file. Similarly, if the call is made with `-env staging`, then the `Key` should be present in either `global` or `staging` file. If the `Key` is defined in both files, `global` is ignored.
> - `Key` is case sensitive. So, `{{.Url}}` and `{{.url}}` are two different values.

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
