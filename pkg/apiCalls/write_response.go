package apicalls

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"golang.org/x/net/html"
)

func isJson(str string) bool {
	var jsBfr json.RawMessage
	return json.Unmarshal([]byte(str), &jsBfr) == nil
}

func isXML(str string) bool {
	var v interface{}
	return xml.Unmarshal([]byte(str), &v) == nil
}

func isHTML(str string) bool {
	_, err := html.Parse(strings.NewReader(str))
	return err == nil
}

func writeFile(fileName, suffixType, contentBody string) {
	if err := os.WriteFile(fileName+suffixType, []byte(contentBody), 0644); err != nil {
		utils.PrintRed("call.go: error while saving file \n" + err.Error())
	}
}

// Checks whether the response is of certain type, json, xml, html or text.
// Based on the evaluation, it writes the response to the provided path in respective file path
func EvalAndWriteRes(resBody, path, fileName string) {
	if fileName == "" || resBody == "" {
		utils.PrintRed("Invalid input: fileName and resBody cannot be empty")
		return
	}

	if isJson(resBody) {
		writeFile(fileName, ".json", resBody)
	} else if isXML(resBody) {
		writeFile(fileName, ".xml", resBody)
	} else if isHTML(resBody) {
		writeFile(fileName, ".html", resBody)
	} else {
		writeFile(fileName, ".txt", resBody)
	}
}
