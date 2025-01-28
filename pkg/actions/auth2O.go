package actions

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
