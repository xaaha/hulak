// Package apicalls has all things related to api call
package apicalls

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"golang.org/x/net/html"
)

func IsJSON(str string) bool {
	var jsBfr json.RawMessage
	return json.Unmarshal([]byte(str), &jsBfr) == nil
}

func IsXML(str string) bool {
	var v any
	return xml.Unmarshal([]byte(str), &v) == nil
}

func IsHTML(str string) bool {
	doc, err := html.Parse(strings.NewReader(str))
	return err == nil && strings.Contains(str, "</html>") && doc != nil
}

// Write the content to the specified path with the appropriate file extension
func writeFile(path, suffixType, contentBody string) {
	fileName := utils.FileNameWithoutExtension(path) + utils.ResponseBase
	dir := filepath.Dir(path)
	fullFilePath := filepath.Join(dir, fileName+suffixType)
	if err := os.WriteFile(fullFilePath, []byte(contentBody), 0600); err != nil {
		utils.PrintRed("Error while saving file: %v\n" + err.Error())
		return
	}
}

// checks the content type of resBody and writes to the corresponding file format
func evalAndWriteRes(resBody, path string) error {
	if resBody == "" || path == "" {
		return utils.ColorError("Invalid input: file path and resBody cannot be empty")
	}

	switch {
	case IsJSON(resBody):
		writeFile(path, ".json", resBody)
	case IsXML(resBody):
		writeFile(path, ".xml", resBody)
	case IsHTML(resBody):
		writeFile(path, ".html", resBody)
	default:
		writeFile(path, ".txt", resBody)
	}
	return nil
}
