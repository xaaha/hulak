package userflags

import (
	"github.com/xaaha/hulak/pkg/utils"
)

// Returns a slice of file paths based on the flags -f and -fp.
func GenerateFilePathList(fileName string, fp string) ([]string, error) {
	standardErrMsg := "to send api request(s), please provide a valid file name with \n'-f fileName' flag or  \n'-fp file/path/' "

	// Both inputs are empty, return an error
	if fileName == "" && fp == "" {
		return nil, utils.ColorError(standardErrMsg)
	}

	var filePathList []string

	// Add file path from -fp flag if provided
	if fp != "" {
		filePathList = append(filePathList, fp)
	}

	// Add matching paths for -f flag if provided
	if fileName != "" {
		if matchingPaths, err := utils.ListMatchingFiles(fileName); err != nil {
			utils.PrintRed("helper.go: error occurred while collecting file paths " + err.Error())
		} else {
			filePathList = append(filePathList, matchingPaths...)
		}
	}

	if len(filePathList) == 0 {
		return nil, utils.ColorError(standardErrMsg)
	}
	return filePathList, nil
}
