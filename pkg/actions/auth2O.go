package actions

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	apicalls "github.com/xaaha/hulak/pkg/apiCalls"
	"github.com/xaaha/hulak/pkg/yamlParser"
)

/*
 Define types for yaml file ✅
 Use checkYamlFile in reader.go and get the  to the defined type above. ✅
 Use Auth section to determine if we should follow this flow. ✅
 Use Method, Url and parameters for open ✅
 After user authorization, we'll receive a code
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

// copied from Githubhttps://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8?permalink_comment_id=5084817#gistcomment-5084817
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

// CallbackServer represents our local OAuth callback server
type CallbackServer struct {
	server   *http.Server
	token    string
	tokenErr error
	wg       sync.WaitGroup
}

// handleCallback processes the OAuth callback
func (cs *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Extract the token from the query parameters
	token := r.URL.Query().Get("code")
	if token == "" {
		cs.tokenErr = fmt.Errorf("no access token found in callback")
		fmt.Fprintln(w, "No access token found. You can close this window.")
		cs.shutdown()
		return
	}

	fmt.Println("This is my code: ", token)
	// Store the token and shutdown the server
	cs.token = token
	fmt.Fprintln(w, "Authentication successful! You can close this window.")
	cs.shutdown()
}

// NewCallbackServer creates a new callback server instance
func NewCallbackServer(port int) *CallbackServer {
	cs := &CallbackServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: http.NewServeMux(),
		},
	}

	// Setup the callback handler with ServeMux
	mux := cs.server.Handler.(*http.ServeMux)
	mux.HandleFunc("/callback", cs.handleCallback)
	return cs
}

// shutdown gracefully shuts down the server
func (cs *CallbackServer) shutdown() {
	go func() {
		if err := cs.server.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()
	cs.wg.Done()
}

// Start starts the callback server
func (cs *CallbackServer) Start() {
	cs.wg.Add(1)
	go func() {
		if err := cs.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
			cs.tokenErr = err
			cs.wg.Done()
		}
	}()
}

// WaitForToken waits for the callback to complete and returns the token or error
func (cs *CallbackServer) WaitForToken() (string, error) {
	cs.wg.Wait()
	return cs.token, cs.tokenErr
}

// GrabCode starts a minimal HTTP server,
// grabs the 'code' query parameter, then immediately
// shuts down the server and the running process.
// func GrabCode(filePath string, secretsMap map[string]interface{}) {
// 	var wg sync.WaitGroup
// 	wg.Add(1)
//
// 	// Channel to receive the captured code
// 	codeCh := make(chan string, 1)
//
// 	// simple server
// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
// 		// Extract the 'code' parameter
// 		code := r.URL.Query().Get("code")
// 		if code == "" {
// 			fmt.Fprintln(w, "No code found. You can close this window.")
// 		} else {
// 			fmt.Fprintln(w, "Authentication successful! You can close this window.")
// 			fmt.Println("Captured code:", code)
// 			codeCh <- code
// 		}
// 		wg.Done()
// 	})
//
// 	server := &http.Server{
// 		Addr:    fmt.Sprintf(":%d", 2882), // port
// 		Handler: mux,
// 	}
//
// 	// Listen for manual shutdown signals (Ctrl+C, etc.)
// 	stopCh := make(chan os.Signal, 1)
// 	signal.Notify(stopCh, os.Interrupt)
//
// 	// Start the server in a new goroutine
// 	go func() {
// 		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 			log.Fatalf("Server error: %v", err)
// 		}
// 	}()
// 	log.Println("Callback server is running on http://localhost:2882/callback")
//
// 	// OpenBrowser here
// 	if err := OpenBrowser(filePath, secretsMap); err != nil {
// 		log.Fatalf("OpenBrowser error: %v", err)
// 	}
//
// 	// Wait for the callback to complete or a shutdown signal
// 	go func() {
// 		wg.Wait()
// 		// Shut down the server when the callback is done
// 		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 		defer cancel()
// 		_ = server.Shutdown(ctx)
// 		close(codeCh)
// 	}()
//
// 	select {
// 	case <-stopCh:
// 		// If someone manually sends an interrupt (Ctrl+C) before the callback
// 		fmt.Println("Received interrupt signal. Shutting down.")
// 	case c := <-codeCh:
// 		// If the code was received successfully
// 		if c != "" {
// 			fmt.Println("Code received:", c)
// 		}
// 		// Exit the whole process after shutting down the server
// 		os.Exit(0)
// 	}
// }

// OpenBrowser starts the callback server and opens the browser for OAuth flow
func OpenBrowser(filePath string, secretsMap map[string]interface{}) error {
	// Create and start the callback server
	callbackServer := NewCallbackServer(2882)
	callbackServer.Start()

	// Prepare the OAuth URL
	authReqBody := yamlParser.FinalStructForOAuth2(filePath, secretsMap)
	urlStr := apicalls.PrepareUrl(string(authReqBody.Url), authReqBody.UrlParams)

	// Open the browser
	log.Println("Opening browser for authentication...")
	if err := OpenURL(urlStr); err != nil {
		return fmt.Errorf("error opening browser: %w", err)
	}

	// Wait for the callback to complete
	token, err := callbackServer.WaitForToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	log.Printf("Successfully received access token %v", token)
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

// func main() {
// 	// Initialize OAuth2 configuration
// 	config := OAuth2Config{
// 		AuthURL:      "YOUR_AUTH_URL",
// 		TokenURL:     "YOUR_TOKEN_URL",
// 		ClientID:     "YOUR_CLIENT_ID",
// 		ClientSecret: "YOUR_CLIENT_SECRET",
// 		Scope:        "YOUR_SCOPE",
// 		State:        "random_state_string", // Generate a random string for security
// 		RedirectURI:  "YOUR_REDIRECT_URI",
// 	}
//
// 	// Step 1: Generate the authorization URL
// 	authURL := GenerateAuthURL(config)
// 	fmt.Printf("Visit this URL to authorize: %s\n", authURL)
//
// 	// Step 2: After user authorization, you'll receive a code
// 	// For demonstration, let's assume you have the code
// 	var authCode string
// 	fmt.Print("Enter the authorization code: ")
// 	fmt.Scan(&authCode)
//
// 	// Step 3: Exchange the code for an access token
// 	token, err := GetAccessToken(config, authCode)
// 	if err != nil {
// 		fmt.Printf("Error getting access token: %v\n", err)
// 		return
// 	}
//
// 	fmt.Printf("Response: %s\n", token)
// }

// to capture the token from the browser, you need to spin up the local server
/*
package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
)

const redirectURI = "http://localhost:8080/callback"

func main() {

	// Open the authorization URL in the user's browser
	err := OpenURL(authURL)
	if err != nil {
		log.Fatalf("Failed to open URL: %v", err)
	}

	// Start a simple web server to handle the callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Extract the token from the query parameters
		token := r.URL.Query().Get("access_token")
		if token == "" {
			fmt.Fprintln(w, "No access token found.")
			log.Println("Access token not found in the request.")
			return
		}

		fmt.Fprintf(w, "Access Token: %s", token)
		log.Printf("Access Token received: %s\n", token)

		// Close the server after capturing the token
		go func() {
			_ = http.DefaultServer.Close()
		}()
	})

	log.Println("Listening for the callback...")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

*/
