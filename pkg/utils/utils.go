package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type Utilities struct{}

// Creates and returns file path by joining the project root with provided filePath
func CreateFilePath(filePath string) (string, error) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", err
	}
	finalFilePath := filepath.Join(projectRoot, filePath)

	return finalFilePath, nil
}

// Get a list of environment file names from the env folder
func (u *Utilities) GetEnvFiles() ([]string, error) {
	var environmentFiles []string
	dir, err := os.Getwd()
	if err != nil {
		return environmentFiles, err
	}
	// get a list of envFileName
	contents, err := os.ReadDir(dir + "/env")
	if err != nil {
		panic(err)
	}

	// discard any folder in the env directory
	for _, fileOrDir := range contents {
		if !fileOrDir.IsDir() {
			lowerCasedEnvFromFile := strings.ToLower(fileOrDir.Name())
			environmentFiles = append(environmentFiles, lowerCasedEnvFromFile)
		}
	}
	return environmentFiles, nil
}

// converts all keys in a map to lowercase recursively
func ConvertKeysToLowerCase(dict map[string]interface{}) map[string]interface{} {
	loweredMap := make(map[string]interface{})
	for key, val := range dict {
		lowerKey := strings.ToLower(key)
		switch almostFinalValue := val.(type) {
		case map[string]interface{}:
			loweredMap[lowerKey] = ConvertKeysToLowerCase(almostFinalValue)
		default:
			loweredMap[lowerKey] = almostFinalValue
		}
	}
	return loweredMap
}

// Copies the Environment map[string]string and returns a CopyEnvMap
// EnvMap is a simple json without any nested properties.
// Mostly used for go routines
func CopyEnvMap(original map[string]string) map[string]string {
	result := map[string]string{}
	for key, val := range original {
		result[key] = val
	}
	return result
}

// Searches for files matching the "matchFile" name (case-insensitive, .yaml/.yml only)
// in the specified directory and its subdirectories. If no directory is specified, it starts from the project root.
// Skips all hidden folders like `.git`, `.vscode` or `.random` folder during traversal.
// Returns slice of matched file path and an error if no matching files are found or if there are file system errors.
func ListMatchingFiles(matchFile string, initialPath ...string) ([]string, error) {
	matchFile = strings.ToLower(matchFile)
	var result []string

	initAbsFp, err := CreateFilePath("")
	if err != nil {
		return nil, fmt.Errorf("error getting initial file path: %w", err)
	}

	var startPath string
	if len(initialPath) == 0 {
		startPath = initAbsFp
	} else {
		startPath = initialPath[0]
	}

	dirContents, err := os.ReadDir(startPath)
	if err != nil {
		return nil, ColorError("error reading directory "+startPath, err)
	}

	filePattern := [2]string{YAML, YML}

	for _, val := range dirContents {
		// Skip hidden directories
		if val.IsDir() && strings.HasPrefix(val.Name(), ".") {
			continue
		}

		// Process files
		if !val.IsDir() {
			lowerName := strings.ToLower(val.Name())
			for _, ext := range filePattern {
				if strings.HasSuffix(lowerName, ext) {
					yamlFile := strings.TrimSuffix(lowerName, ext)
					if matchFile == yamlFile {
						matchingFp := filepath.Join(startPath, val.Name())
						result = append(result, matchingFp)
					}
				}
			}
		}

		// Process subdirectories
		if val.IsDir() {
			subDirPath := filepath.Join(startPath, val.Name())
			matches, err := ListMatchingFiles(matchFile, subDirPath)
			if err != nil && !isNoMatchingFileError(err) {
				PrintRed("Skipping subdirectory" + val.Name() + "due to error: \n" + err.Error())
				continue
			}
			result = append(result, matches...)
		}
	}

	if len(result) == 0 {
		return []string{}, ColorError(
			"no files with matching name " + matchFile + " found in " + initAbsFp,
		)
	}
	return result, nil
}

// isNoMatchingFileError determines if the error is related to no matching files found.
func isNoMatchingFileError(err error) bool {
	return strings.Contains(err.Error(), "no files with matching name")
}

// takes in filepath and returns the name of the file
func FileNameWithoutExtension(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func LookUpValuePath(key string, data map[string]interface{}) (string, error) {
	if value, exists := data[key]; exists {
		return MarshalToJSON(value)
	}

	pathSeparator := "."
	segments := parseKeySegments(key, pathSeparator)
	current := interface{}(data)

	for i, segment := range segments {
		// Check if the segment includes an array index
		isArrayKey, keyPart, index := parseArrayKey(segment)

		if isArrayKey {
			// Ensure current is an array
			arr, ok := current.([]interface{})
			if !ok {
				return "", errors.New("key not found or not an array: " + segment)
			}

			// Check index validity
			if index < 0 || index >= len(arr) {
				return "", errors.New("array index out of bounds: " + segment)
			}

			current = arr[index]
		} else {
			// Treat as map key
			currMap, ok := current.(map[string]interface{})
			if !ok {
				// Handle struct conversion
				currMap, ok = structToMap(current)
				if !ok {
					return "", errors.New("invalid path, segment is not a map: " + strings.Join(segments[:i+1], pathSeparator))
				}
			}

			value, exists := currMap[keyPart]
			if !exists {
				return "", errors.New("key not found: " + strings.Join(segments[:i+1], pathSeparator))
			}
			current = value
		}

		// If this is the last segment, marshal and return the value
		if i == len(segments)-1 {
			return MarshalToJSON(current)
		}
	}

	return "", errors.New("unexpected error")
}

func structToMap(value interface{}) (map[string]interface{}, bool) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Struct {
		mapData := make(map[string]interface{})
		valType := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := valType.Field(i)
			mapData[field.Name] = val.Field(i).Interface()
		}
		return mapData, true
	}
	return nil, false
}

func parseKeySegments(key, pathSeparator string) []string {
	var segments []string
	current := strings.Builder{}
	inBracket := false

	for _, char := range key {
		switch {
		case char == '{':
			inBracket = true
		case char == '}':
			inBracket = false
			segments = append(segments, current.String())
			current.Reset()
		case char == rune(pathSeparator[0]) && !inBracket:
			segments = append(segments, current.String())
			current.Reset()
		default:
			current.WriteRune(char)
		}
	}
	// Append the last segment, if any
	if current.Len() > 0 {
		segments = append(segments, current.String())
	}
	return segments
}

func parseArrayKey(segment string) (bool, string, int) {
	if strings.HasSuffix(segment, "]") && strings.Contains(segment, "[") {
		openBracket := strings.LastIndex(segment, "[")
		closeBracket := strings.LastIndex(segment, "]")
		indexStr := segment[openBracket+1 : closeBracket]
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			return false, segment, -1 // Invalid index
		}
		return true, segment[:openBracket], index
	}
	return false, segment, -1
}

func MarshalToJSON(value interface{}) (string, error) {
	if arr, ok := value.([]interface{}); ok {
		var jsonArray []string
		for _, item := range arr {
			jsonStr, err := json.Marshal(item)
			if err != nil {
				return "", err
			}
			jsonArray = append(jsonArray, string(jsonStr))
		}
		return fmt.Sprintf("[%s]", strings.Join(jsonArray, ",")), nil
	}
	jsonStr, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}

/*
type Person struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Years int    `json:"years"`
}

func main() {
	myDict := map[string]interface{}{
		"company.inc": "Test Company",
		"name":        "Pratik",
		"age":         32,
		"years":       111,
		"marathon":    false,
		"profession": map[string]interface{}{
			"company.info": "Earth Based Human Led",
			"title":        "Senior Test SE",
			"years":        5,
		},
		"myArr": []Person{
			{Name: "xaaha", Age: 22, Years: 11},
			{Name: "pt", Age: 35, Years: 88},
		}, "myArr2": []interface{}{
			Person{Name: "xaaha", Age: 22, Years: 11},
			Person{Name: "pt", Age: 35, Years: 88},
		},
	}

	result, err := LookUpValuePath("age", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'age':", result)
	}

	result, err = LookUpValuePath("marathon", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'marathon':", result)
	}

	result, err = LookUpValuePath("profession", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'profession':", result)
	}

	result, err = LookUpValuePath("myArr2", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'myArr2':", result)
	}

	result, err = LookUpValuePath("{company.inc}", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for '{company.inc}':", result)
	}

	result, err = LookUpValuePath("profession.{company.info}", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'company.info':", result)
	}

	result, err = LookUpValuePath("myArr[0].name", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'myArr':", result)
	}
}
*/
