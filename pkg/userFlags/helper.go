package userflags

import "github.com/xaaha/hulak/pkg/utils"

// returns a slice of file paths to run from the flags -f and -fp
func GenerateFilePathList(fileName string, fp string) ([]string, error) {
	var filePathlist []string
	standardErrMsg := "to send api request(s), please provide a file name with '-f fileName' flag or use  '-fp file/path/' to provide the file path from environment directory"

	if len(fileName) == 0 && len(fp) == 0 {
		return []string{}, utils.ColorError(standardErrMsg)
	}

	if len(fp) > 0 {
		filePathlist = append(filePathlist, fp)
	}

	if len(fileName) > 0 {
		matchingPaths, err := utils.ListMatchingFiles(fileName)
		if err != nil {
			utils.PrintRed("helper.go: error occured while collecting file paths" + err.Error())
		}
		filePathlist = append(filePathlist, matchingPaths...)
	}

	var error error
	if len(filePathlist) == 0 {
		error = utils.ColorError(standardErrMsg)
	}

	return filePathlist, error
}
