// Package features have all the additional features hulak supports
package features

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

const (
	portNum      = ":2982"
	timeout      = 60 * time.Second
	redirectURI  = "http://localhost" + portNum + "/callback"
	responseType = utils.ResponseType // for consistency
)

// OpenURL Opens the url in the brwoser based on the user's OS
// copied from Github https://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8?permalink_comment_id=5084817#gistcomment-5084817
func OpenURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		// Check if running under WSL
		if isWSL() {
			// Use 'cmd.exe /c start' to open the URL in the default Windows browser
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			// Use xdg-open on native Linux environments
			cmd = "xdg-open"
			args = []string{url}
		}
	}
	if len(args) > 1 {
		// args[0] is used for 'start' command argument, to prevent issues with URLs starting with a quote
		args = append(args[:1], append([]string{""}, args[1:]...)...)
	}
	return exec.Command(cmd, args...).Start()
}

// Channel to communicate the received code
var codeChan = make(chan string)

// handle '/callback' to processes the OAuth server
func callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code != "" {
		authHtml := filepath.Join("assets", "auth.html")
		http.ServeFile(w, r, authHtml)
		// Send the code to the channel and close it
		codeChan <- code
		close(codeChan)
	} else {
		fmt.Fprint(w, "No 'code' query parameter found.")
	}
}

func server() {
	// log.Println("Starting server on port", portNum)
	http.HandleFunc("/callback", callback)
	err := http.ListenAndServe(portNum, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// openBrowserAndGetCode starts the callback server and opens the browser for OAuth flow
// Returns the code coming from the ur
func openBrowserAndGetCode(filePath string, secretsMap map[string]any) (string, error) {
	// Create and start the callback server
	go server()
	// Prepare the OAuth URL
	authReqBody, err := yamlparser.FinalStructForOAuth2(filePath, secretsMap)
	if err != nil {
		return "", err
	}

	// required fields for oAuth web flow. This is true github and Okta.
	// from my testing, extra field does not do any harm, if this is not the case, I'll revisit
	reqField := make(map[string]string)
	reqField["response_type"] = responseType
	reqField["redirect_uri"] = redirectURI
	authReqBody.UrlParams = utils.MergeMaps(authReqBody.UrlParams, reqField)
	urlStr := apicalls.PrepareURL(string(authReqBody.Url), authReqBody.UrlParams)

	// Open the browser
	log.Println("Opening browser for authentication...")
	if err := OpenURL(urlStr); err != nil {
		return "", utils.ColorError("error opening browser: %w", err)
	}
	// Wait for the code or a timeout
	select {
	case code := <-codeChan:
		return code, nil
	case <-time.After(timeout):
		return "", utils.ColorError("timeout waiting for the code")
	}
}

// SendAPIRequestForAuth2  calls the PrepareStruct using the provided envMap
// and makes the Api Call with StandardCall and prints the response in console
func SendAPIRequestForAuth2(secretsMap map[string]any, filePath string, debug bool) error {
	code, err := openBrowserAndGetCode(filePath, secretsMap)
	if err != nil {
		return err
	}

	authReqConfig, err := yamlparser.FinalStructForOAuth2(filePath, secretsMap)
	if err != nil {
		return err
	}

	apiInfo, err := authReqConfig.PrepareStruct(code)
	if err != nil {
		return err
	}
	resp, err := apicalls.StandardCall(apiInfo, debug)
	if err != nil {
		return err
	}
	apicalls.PrintAndSaveFinalResp(resp, filePath)
	return nil
}

// isWSL checks if the Go program is running inside Windows Subsystem for Linux
func isWSL() bool {
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}
