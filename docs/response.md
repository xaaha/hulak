# Response

- Std output is always a json response. If the response body is a format other than json, it would be converted to string.
- But, appropriate response format is saved in the same path as the file making the call.
- By default rsponse is saved with the same name with "\_response.json" added.
  So, `test.yaml` would have it's response saved as`test_response.json`
- If the reponse is plain text it would be `test_response.txt`
- If the response is xml, the response file would have the proper suffix `test_response.xml`
- If the reponse is not successful, the entire std output is saved as a json file

```shell
getUserData.yaml # calling file
getuserData_response.json # automated saved response
```
