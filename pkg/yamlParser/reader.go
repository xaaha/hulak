package yamlParser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// From the yaml file, create a json file. But the json could have {{}} on it
// So, we need to, read the file, make sure those values are handled, then return the proper map

// Parses the user's input yaml file to a json interface.
// Then, this function recursively replaces all variables {{.value}} specified in user's yaml values, with values from environment map
// This is necessary, as the some variables, like URL needs correct string
func replaceVarsWithValues(
	dict map[string]interface{},
	secretsMap map[string]interface{},
) map[string]interface{} {
	changedMap := make(map[string]interface{})

	for key, val := range dict {
		switch valTyped := val.(type) {
		case map[string]interface{}:
			changedMap[key] = replaceVarsWithValues(valTyped, secretsMap)
		case string:
			finalChangedValue, err := envparser.SubstituteVariables(valTyped, secretsMap)
			if err != nil {
				utils.PrintRed(err.Error())
			}
			if replacedValue, ok := secretsMap[valTyped]; ok {
				changedMap[key] = replacedValue
			} else {
				changedMap[key] = finalChangedValue
			}
		case map[string]string:
			innerMap := make(map[string]interface{})
			for k, v := range valTyped {
				finalChangedValue, err := envparser.SubstituteVariables(v, secretsMap)
				if err != nil {
					utils.PrintRed(err.Error())
				}
				if replacedValue, ok := secretsMap[v]; ok {
					innerMap[k] = replacedValue
				} else {
					innerMap[k] = finalChangedValue
				}
			}
			changedMap[key] = innerMap
		default:
			changedMap[key] = val
		}
	}
	return changedMap
}

// checks whether string matches exactly "{{value}}"
// and retuns whether the string matches the delimiter criteria and the associated content
// So, the "{{ .value }}" returns "true, .value". Space is trimmed around the return string
func stringHasDelimiter(value string) (bool, string) {
	if len(value) < 4 || !strings.HasPrefix(value, "{{") || !strings.HasSuffix(value, "}}") {
		return false, ""
	}
	if strings.Count(value[:3], "{") > 2 || strings.Count(value[len(value)-3:], "}") > 2 {
		return false, ""
	}
	content := value[2 : len(value)-2]
	re := regexp.MustCompile(`^\s+$`)
	onlyHasEmptySpace := re.Match([]byte(value))
	if len(content) == 0 || onlyHasEmptySpace {
		return false, ""
	}
	content = strings.TrimSpace(content)
	return len(content) > 0, content
}

// for actions, evaluate the type that's coming from the getValueOf, and convert it to the original form
// return the type, and use the type convert the value again to the original type
// func stringHasTypeAssertion(value string) bool {
// 	return false
// }
// what type is the value coming from secretsMap, is the final dataAfter value, (key's value the same as before)?
// For example, for `key: "{{.value}}"` if this gets replaced to `key: "22"`, what type was the 22 before it became string?

func CompareAndConvert(
	dataBefore, dataAfter, secretsMap map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})
	beforeMap := make(map[string]interface{})
	// range over on dataBefore, (key, value) and find all the values, with valid delimiters "{{}}" (recursion)
	// -- keep track of the key and value we are concerned with `"myAwesomeNumber": "{{.myAwesomeNumber}}"`

	for _, bvalue := range dataBefore {
		switch bValType := bvalue.(type) {
		case string:
			strHasDelimeter, innerStr := stringHasDelimiter(bValType)
			if strHasDelimeter {
				// first separate them into array [getValueOf, `"foo"`, `"bar"`] or [".value"]
				innerStrChunks := strings.Split(innerStr, " ")
				// evaluate if the string has .value
				if len(innerStrChunks) == 1 { // when using dot there should only be 1 in the array
					dotStr := innerStrChunks[0] // get the first chunk with dot (.)
					if strings.Contains(dotStr, ".") {
						dotStr = strings.Replace(dotStr, ".", "", 1) // remove the first dot
						beforeMap[dotStr] = secretsMap[dotStr]
					}
				}
				// getValueOf "key" "path"
				if len(innerStrChunks) == 3 && innerStrChunks[0] == "getValueOf" {
					gvoKey := innerStrChunks[1]  // key
					gvoPath := innerStrChunks[2] // path
					// replace the characters " and `. Single quote ' is not allowed in go template
					gvoKey = strings.ReplaceAll(gvoKey, `"`, "")
					gvoPath = strings.ReplaceAll(gvoPath, `"`, "")
					gvoKey = strings.ReplaceAll(gvoKey, "`", "")
					gvoPath = strings.ReplaceAll(gvoPath, "`", "")
					beforeMap[gvoKey] = envparser.GetValueOf(gvoKey, gvoPath)
				}
			}
		case map[string]interface{}:
			return CompareAndConvert(bValType, dataAfter, secretsMap)
		default:
			// Handle Array of objects
			// handle simple arrays
			// Handle error and other cases
		}
	}

	// From here, we get a flatmap like this {myAwesomeNumber : 22, foo: false}
	// but we will get our values replaced in flatMap if there are items with the same key in nested objects
	// So, we need to return path as well. Somehow, we need to create path and attach the path to the key and it's value

	// {"myAwesomeNumber": 22, "foo": false, "body.foo": "345", "body.{company.inc}" : "xaaha.inc", "body.{company.inc}.assets[0]: 2344239" ]

	// finally, loop over the dataAfter and look for the key from beforeMap
	// the path of the key returned by the dataBefore map should make this easier to find exaclty
	// for aKey, aValue := range dataAfter {
	// }

	// Then if the type of the value in dataAfter != type of the value we got from type evaluation, then convert and relace in dataAfter

	return result
}

// Reads YAML, validates if the file exists, is not empty, and changes keys to lowercase for http request.
// Right now, the yaml file is only meant to hold http request as defined in the body struct in "./yamlTypes.go"
func checkYamlFile(filepath string, secretsMap map[string]interface{}) (*bytes.Buffer, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		utils.PanicRedAndExit("File does not exist, %s", filepath)
	}

	file, err := os.Open(filepath)
	if err != nil {
		utils.PanicRedAndExit("Error opening file: %v", err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	if fileInfo.Size() == 0 {
		utils.PanicRedAndExit("Empty yaml file")
	}

	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("1. error decoding data: %v", err)
	}

	// case sensitivity keys in yaml file is ignored.
	// method or Method or METHOD should all be the same
	data = utils.ConvertKeysToLowerCase(data)

	// TODO:
	// if data has key whose value is a template,
	// && the replacement value's type is either false, float64, int, nil/null
	// convert these to original values again
	// or if the key is the same,
	// and the value are of different type, convert them from string to the one of secretsMap

	// parse all the values to with {{.key}} from .env folder
	parsedMap := replaceVarsWithValues(data, secretsMap)

	// dataFmt, _ := utils.MarshalToJSON(data)
	// fmt.Println("this is data", dataFmt)
	// printPm, _ := utils.MarshalToJSON(parsedMap)
	// fmt.Println("this is parsed map", printPm)

	// TODO:
	// parsedMap is always string

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(parsedMap); err != nil {
		utils.PanicRedAndExit("error encoding data: %v", err)
	}
	enc.Close()

	return &buf, nil
}

// checks the validity of all the fields in the yaml file
// and returns the json string of the yaml file
func ReadYamlForHttpRequest(filePath string, secretsMap map[string]interface{}) string {
	buf, err := checkYamlFile(filePath, secretsMap)
	if err != nil {
		utils.PanicRedAndExit("Error occured after reading yaml file: %v", err)
	}

	var user User
	dec := yaml.NewDecoder(buf)
	if err := dec.Decode(&user); err != nil {
		utils.PanicRedAndExit("2. error decoding data: %v", err)
	}

	// uppercase and type conversion
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(user.Method)))
	user.Method = upperCasedMethod

	// method is required for any http request
	if !user.Method.IsValid() {
		utils.PanicRedAndExit("missing or invalid HTTP method: %s", user.Method)
	}

	// url is required for any http request
	if !user.Url.IsValidURL() {
		utils.PanicRedAndExit("missing or invalid URL: %s", user.Url)
	}

	// check if body is valid
	// if the body is not present in the body, then the body is nil
	if user.Body != nil && !user.Body.IsValid() {
		utils.PanicRedAndExit(
			"Invalid Body. Make sure body contains only one valid argument.\n %v",
			user.Body,
		)
	}
	val, _ := json.MarshalIndent(user, "", "  ")
	jsonString := string(val)
	return jsonString
}

func ReadingYamlWithoutStruct() {
	file, err := os.Open("test_collection/test.yml")
	if err != nil {
		utils.PanicRedAndExit("Error opening file: %v", err)
	}
	defer file.Close()

	var data map[string]interface{}
	dec := yaml.NewDecoder(file)
	if err = dec.Decode(&data); err != nil {
		utils.PanicRedAndExit("3. error decoding data: %v", err)
	}
	val, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(val))
}
