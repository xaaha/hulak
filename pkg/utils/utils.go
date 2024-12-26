package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// searches the myKey in dict, and returns the interface[].
// If the interface is map[string]interface{}, a json string is returned
// LookupValue collects all matches for a given key, including nested maps and arrays.
// TODO: accept [0] for the array otherwise return error && skip . and [] with \ and make them string

func LookUpValuePath(key string, data map[string]interface{}) (string, error) {
	if value, exists := data[key]; exists {
		return marshalToJSON(value)
	}

	// Path separator to traverse the key path
	pathSeparator := "."
	segments := strings.Split(key, pathSeparator)
	current := data

	// Traverse the path
	for i, segment := range segments {
		value, exists := current[segment]
		if !exists {
			return "", ColorError("key not found: " + strings.Join(segments[:i+1], pathSeparator))
		}

		// If we're at the last segment, return the JSON representation of the value
		if i == len(segments)-1 {
			return marshalToJSON(value)
		}

		// continue inside map
		if nestedMap, ok := value.(map[string]interface{}); ok {
			current = nestedMap
		} else {
			return "", ColorError("invalid path, segment is not a map: " + strings.Join(segments[:i+1], pathSeparator))
		}
	}

	return "", ColorError("unexpected error")
}

func marshalToJSON(value interface{}) (string, error) {
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
		"name":  "Pratik",
		"age":   32,
		"years": 111,
		"profession": map[string]interface{}{
			"title": "Senior Test SE",
			"years": 5,
		},
		"myArr": []Person{
			{Name: "xaaha", Age: 22, Years: 11},
			{Name: "pt", Age: 35, Years: 88},
		},
		"myArr2": []interface{}{
			Person{Name: "xaaha", Age: 22, Years: 11},
			Person{Name: "pt", Age: 35, Years: 88},
		},
	}

	result, err := LookUpValuePath("profession", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'profession':", result)
	}

	result, err = LookUpValuePath("myArr", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'myArr':", result)
	}
	result, err = LookUpValuePath("years", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'years':", result)
	}
	result, err = LookUpValuePath("myArr[0].name", myDict)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("Result for 'myArr':", result)
	}
}
*/
