// Package curl handles importing cURL commands into Hulak YAML files
package curl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// ImportCurl handles the import curl subcommand
// Supports multiple input methods:
// 1. Command line argument: hulak import curl 'curl command'
// 2. Stdin/pipe: echo 'curl command' | hulak import curl
// 3. Heredoc: hulak import curl <<'EOF' ... EOF
// outputPath is from -o flag (empty string if not provided)
func ImportCurl(args []string, outputPath string) error {
	var curlString string
	var err error

	// Show usage help if no arguments are provided after "curl"
	if len(args) < 1 {
		utils.PrintCurlImportUsage()
		return nil
	}

	// Check if first arg is "curl" keyword
	if args[0] != "curl" {
		return utils.ColorError("expected 'curl' keyword after 'import'")
	}

	// Decide between stdin and command-line argument
	if len(args) < 2 || args[1] == "" || args[1] == "-" {
		// Read from stdin (pipe or heredoc)
		curlString, err = readCurlFromStdin()
		if err != nil {
			return err
		}
	} else {
		// Use command-line argument
		curlString = args[1]
	}

	// Validate we have input
	if strings.TrimSpace(curlString) == "" {
		return utils.ColorError("no cURL command provided")
	}

	// Parse the cURL command (using yamlparser's function)
	apiCallFile, err := yamlparser.ParseCurlCommand(curlString)
	if err != nil {
		return utils.ColorError("failed to parse curl command: %w", err)
	}

	// Convert to YAML (this replaces ConvertToYAML function)
	yamlContent, err := yaml.Marshal(apiCallFile)
	if err != nil {
		return utils.ColorError("failed to convert to YAML: %w", err)
	}

	// Add document separator
	fullYamlContent := "---\n" + string(yamlContent)

	// Determine output file path (method signature updated to use ApiCallFile)
	filePath, err := determineOutputPath(outputPath, apiCallFile)
	if err != nil {
		return err
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(fullYamlContent), utils.FilePer); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	// Success message with usage hint
	utils.PrintGreen(fmt.Sprintf("Created '%s' %s", filePath, utils.CheckMark))
	utils.PrintInfo(fmt.Sprintf("Run with: hulak -env <n> -fp %s", filePath))

	return nil
}

// readCurlFromStdin reads cURL command from stdin and handles piped input and heredoc
func readCurlFromStdin() (string, error) {
	// Check if stdin has data
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat stdin: %w", err)
	}

	// Check if stdin is from pipe/redirect or terminal
	isPipe := (stat.Mode() & os.ModeCharDevice) == 0

	if !isPipe {
		// If not piped input, show usage and exit
		utils.PrintCurlImportUsage()
		os.Exit(0)
	}

	// Read all input from stdin
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Add the last line if it doesn't end with newline
				if line != "" {
					lines = append(lines, line)
				}
				break
			}
			return "", fmt.Errorf("error reading from stdin: %w", err)
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return "", utils.ColorError("no input provided")
	}

	// Join lines and clean up
	curlString := strings.Join(lines, "\n")
	curlString = cleanStdinInput(curlString)

	return curlString, nil
}

// cleanStdinInput cleans up the input from stdin
// Handles multi-line with backslashes, extra whitespace, etc.
func cleanStdinInput(input string) string {
	// Split into lines
	lines := strings.Split(input, "\n")

	// Process each line
	var cleanedLines []string
	for _, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Remove trailing backslash (line continuation)
		line = strings.TrimSuffix(line, "\\")
		line = strings.TrimSpace(line)

		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join with spaces
	result := strings.Join(cleanedLines, " ")

	// Normalize multiple spaces to single space
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

// determineOutputPath decides where to save the file
func determineOutputPath(outputPath string, apiCallFile *yamlparser.ApiCallFile) (string, error) {
	if outputPath == "" {
		// Auto-generate name in imported/ directory
		return generateAutoFilePath(apiCallFile)
	}

	// User provided path - ensure it has correct extension
	if !strings.HasSuffix(outputPath, utils.HulakFileSuffix) &&
		!strings.HasSuffix(outputPath, utils.HulakFileSuffix2) {
		outputPath = outputPath + utils.HulakFileSuffix
	}

	// Handle collision - append incremental number if file exists
	outputPath = handleFileCollision(outputPath)

	// Create directory if needed
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return outputPath, nil
}

// handleFileCollision appends incremental number if file exists
func handleFileCollision(filePath string) string {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, no collision, return filePath
		return filePath
	}

	// If File exists, append number
	// Handle .hk.yaml extension properly
	var ext, base string
	if strings.HasSuffix(filePath, utils.HulakFileSuffix) {
		ext = utils.HulakFileSuffix
		base = strings.TrimSuffix(filePath, utils.HulakFileSuffix)
	} else if strings.HasSuffix(filePath, utils.HulakFileSuffix2) {
		ext = utils.HulakFileSuffix2
		base = strings.TrimSuffix(filePath, utils.HulakFileSuffix2)
	} else {
		ext = filepath.Ext(filePath)
		base = strings.TrimSuffix(filePath, ext)
	}

	counter := 1
	for {
		newPath := fmt.Sprintf("%s_%d%s", base, counter, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
		counter++
	}
}

// generateAutoFilePath creates auto-generated filename
func generateAutoFilePath(apiCallFile *yamlparser.ApiCallFile) (string, error) {
	// Create imported/ directory if it doesn't exist
	if err := os.MkdirAll(utils.ImportDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create imported directory: %w", err)
	}

	// Generate filename: METHOD_urlpart_timestamp.hk.yaml
	method := string(apiCallFile.Method)
	if method == "" {
		method = "GET"
	}

	// Extract useful part from URL
	urlPart := extractURLPart(string(apiCallFile.URL))
	timestamp := time.Now().Unix()

	filename := fmt.Sprintf("%s_%s_%d.hk.yaml", method, urlPart, timestamp)
	fullPath := filepath.Join(utils.ImportDir, filename)

	// Handle collision even for auto-generated files
	return handleFileCollision(fullPath), nil
}

// extractURLPart extracts meaningful part from URL for filename
func extractURLPart(urlStr string) string {
	// Remove protocol
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// Split by / and take meaningful parts
	parts := strings.Split(urlStr, "/")
	if len(parts) == 0 {
		return "request"
	}

	// Take domain and first path segment
	var result string
	if len(parts) > 0 {
		// Use domain or first path segment
		if parts[0] != "" {
			// Extract just the main domain part (not full domain)
			domain := parts[0]
			domainParts := strings.Split(domain, ".")
			if len(domainParts) > 1 {
				// Use the part before the TLD (e.g., "example" from "example.com")
				result = domainParts[len(domainParts)-2]
			} else {
				result = domain
			}
		}
	}

	// If we have a path segment, prefer that over domain
	if len(parts) > 1 && parts[1] != "" {
		result = parts[1]
	}

	// If still empty, use default
	if result == "" {
		result = "request"
	}

	// Sanitize for filename
	result = sanitizeForFilename(result)

	if len(result) > 30 {
		result = result[:30]
	}

	return result
}

// sanitizeForFilename removes/replaces invalid filename characters
func sanitizeForFilename(s string) string {
	// Replace common separators with underscore
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")

	// Remove any other invalid characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	s = reg.ReplaceAllString(s, "")

	// Ensure it doesn't start with a number (optional, but good practice)
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = "r_" + s
	}

	return strings.ToLower(s)
}
