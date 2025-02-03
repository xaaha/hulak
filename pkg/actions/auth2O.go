package actions

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

/*
 Define types for yaml file ✅
 Use checkYamlFile in reader.go and get the  to the defined type above. ✅
 Use Auth section to determine if we should follow this flow. ✅
 Use Method, Url and parameters for open ✅
 After user authorization, we'll receive a code  ✅
 Exchange the code for an access token
 API call with POST request
 	token, err := GetAccessToken(config, authCode)
 	if err != nil {
 		fmt.Printf("Error getting access token: %v\n", err)
 		return
 	}

 	fmt.Printf("Response: %s\n", token)
 }
*/

const (
	portNum = ":2982"
	timeout = 60 * time.Second
)

// OAuth2Config holds the configuration for OAuth2 flow
type OAuth2Config struct {
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
	State        string
	RedirectURI  string
}

func GenerateAuthURL(config OAuth2Config) string {
	// Build authorization URL with query parameters
	params := url.Values{}
	params.Add("client_id", config.ClientID)
	params.Add("response_type", "code")
	params.Add("scope", config.Scope)
	params.Add("state", config.State)
	params.Add("redirect_uri", config.RedirectURI)
	params.Add("connection", "connection value") // Your specific connection value
	params.Add("audience", "audience value")     // Your specific audience value

	return fmt.Sprintf("%s?%s", config.AuthURL, params.Encode())
}

func GetAccessToken(config OAuth2Config, code string) (string, error) {
	// Prepare token request payload
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)

	// Create HTTP client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("POST", config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	return string(body), nil
}

// copied from Github https://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8?permalink_comment_id=5084817#gistcomment-5084817
// Opens the url in the brwoser based on the user's OS
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
		fmt.Printf("Received code: %s\n", code)
		fmt.Fprintf(w, "Code received: %s\nYou can now safely close this window.\n", code)

		// Send the code to the channel and close it
		codeChan <- code
		close(codeChan)
	} else {
		fmt.Fprint(w, "No 'code' query parameter found.")
	}
}

func Server() {
	log.Println("Starting server on port", portNum)
	http.HandleFunc("/callback", callback)
	err := http.ListenAndServe(portNum, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// OpenBrowser starts the callback server and opens the browser for OAuth flow
func OpenBrowser(filePath string, secretsMap map[string]interface{}) error {
	// Create and start the callback server
	go Server()
	// Prepare the OAuth URL
	authReqBody := yamlParser.FinalStructForOAuth2(filePath, secretsMap)
	urlStr := apicalls.PrepareUrl(string(authReqBody.Url), authReqBody.UrlParams)

	// Open the browser
	log.Println("Opening browser for authentication...")
	if err := OpenURL(urlStr); err != nil {
		return fmt.Errorf("error opening browser: %w", err)
	}
	// Wait for the code or a timeout
	select {
	case code := <-codeChan:
		log.Printf("Authentication code received: %s\n", code)
	case <-time.After(timeout):
		log.Println("Timeout waiting for the code.")
		return fmt.Errorf("timeout waiting for the code")
	}
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
