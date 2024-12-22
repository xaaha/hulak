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

func IsJson(str string) bool {
	var jsBfr json.RawMessage
	return json.Unmarshal([]byte(str), &jsBfr) == nil
}

func isXML(str string) bool {
	var v interface{}
	return xml.Unmarshal([]byte(str), &v) == nil
}

func isHTML(str string) bool {
	doc, err := html.Parse(strings.NewReader(str))
	return err == nil && strings.Contains(str, "</html>") && doc != nil
}

// Write the content to the specified path with the appropriate file extension
func writeFile(path, suffixType, contentBody string) {
	fileName := utils.FileNameWithoutExtension(path) + "_response"
	dir := filepath.Dir(path)
	fullFilePath := filepath.Join(dir, fileName+suffixType)
	if err := os.WriteFile(fullFilePath, []byte(contentBody), 0644); err != nil {
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
	case IsJson(resBody):
		writeFile(path, ".json", resBody)
	case isXML(resBody):
		writeFile(path, ".xml", resBody)
	case isHTML(resBody):
		writeFile(path, ".html", resBody)
	default:
		writeFile(path, ".txt", resBody)
	}
	return nil
}
