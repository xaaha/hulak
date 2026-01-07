// Package curl handles importing cURL commands into Hulak YAML files
package curl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

// ImportCurl handles the import curl subcommand
// args[0] should be "curl"
// args[1] should be the curl command string
// outputPath is from -o flag (empty string if not provided)
func ImportCurl(args []string, outputPath string) error {
	// Validate arguments
	if len(args) < 2 {
		return utils.ColorError("usage: hulak import curl 'curl command' [-o path/to/file.hk.yaml]")
	}

	if args[0] != "curl" {
		return utils.ColorError("expected 'curl' keyword after 'import'")
	}

	curlString := args[1]

	// Parse the cURL command
	parsedCurl, err := ParseCurlCommand(curlString)
	if err != nil {
		return utils.ColorError("failed to parse curl command: %w", err)
	}

	// Convert to YAML structure
	yamlContent, err := ConvertToYAML(parsedCurl)
	if err != nil {
		return utils.ColorError("failed to convert to YAML: %w", err)
	}

	// Determine output file path
	filePath, err := determineOutputPath(outputPath, parsedCurl)
	if err != nil {
		return err
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(yamlContent), utils.FilePer); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	// Success message with usage hint
	utils.PrintGreen(fmt.Sprintf("Created '%s' %s", filePath, utils.CheckMark))
	utils.PrintInfo(fmt.Sprintf("Run with: hulak -env <name> -fp %s", filePath))

	return nil
}

// determineOutputPath decides where to save the file
func determineOutputPath(outputPath string, parsed *CurlCommand) (string, error) {
	if outputPath == "" {
		// Auto-generate name in imported/ directory
		return generateAutoFilePath(parsed)
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
func generateAutoFilePath(parsed *CurlCommand) (string, error) {
	// Create imported/ directory if it doesn't exist
	if err := os.MkdirAll(utils.ImportDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create imported directory: %w", err)
	}

	// Generate filename: METHOD_urlpart_timestamp.hk.yaml
	method := strings.ToUpper(parsed.Method)
	if method == "" {
		method = "GET"
	}

	// Extract useful part from URL
	urlPart := extractURLPart(parsed.URL)
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
