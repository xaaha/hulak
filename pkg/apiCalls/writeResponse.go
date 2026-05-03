// Package apicalls has all things related to api call
package apicalls

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"golang.org/x/net/html"
)

// IsJSON reports whether s is a syntactically valid JSON document.
// Uses encoding/json's state-machine validator so truncated payloads like
// `{"key":"unclosed` are rejected (regression for #208).
func IsJSON(s string) bool {
	return json.Valid([]byte(s))
}

// IsXML reports whether s parses as XML.
func IsXML(str string) bool {
	var v any
	return xml.Unmarshal([]byte(str), &v) == nil
}

// IsHTML reports whether s starts with a recognizable HTML structural marker.
//
// We require either `<!DOCTYPE` or a `<html` opening tag (case-insensitive,
// after leading whitespace) instead of substring-matching `</html>` anywhere
// in the body — the old check classified JSON error messages mentioning HTML
// (e.g. `{"error":"... </html> ..."}`) as HTML and saved them with the wrong
// extension (#208).
func IsHTML(s string) bool {
	trimmed := strings.TrimLeft(s, " \t\r\n")
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype html") {
		return true
	}
	if !strings.HasPrefix(lower, "<html") {
		return false
	}
	// Final cheap sanity check: parse succeeds. html.Parse is permissive
	// but it does reject truly broken inputs.
	_, err := html.Parse(strings.NewReader(s))
	return err == nil
}

// extensionFor picks the response-file extension. Content-Type header is the
// primary signal; body sniffing is the fallback for servers that send no
// header or a generic one (application/octet-stream, */*).
//
// Header parsing uses mime.ParseMediaType so parameters like "; charset=utf-8"
// are stripped, and structured-suffix subtypes like "application/vnd.api+json"
// resolve to the right extension via the "+suffix" convention (RFC 6839).
func extensionFor(contentType, body string) string {
	media, _, err := mime.ParseMediaType(contentType)
	if err == nil && media != "" && !isGenericMediaType(media) {
		if ext, ok := extensionForMediaType(media); ok {
			return ext
		}
	}

	// Fall back to body sniffing — order matters: HTML check is now strict
	// enough that JSON containing markup-like substrings won't trip it.
	switch {
	case IsJSON(body):
		return ".json"
	case IsHTML(body):
		return ".html"
	case IsXML(body):
		return ".xml"
	default:
		return ".txt"
	}
}

// isGenericMediaType identifies media types that carry no useful classification
// signal, so callers should fall through to body sniffing instead.
func isGenericMediaType(media string) bool {
	switch media {
	case "application/octet-stream", "*/*":
		return true
	}
	return false
}

// extensionForMediaType maps a parsed media type (no parameters) to a file
// extension. Returns ok=false for unknown types so the caller can fall back.
func extensionForMediaType(media string) (string, bool) {
	// Structured-suffix subtypes per RFC 6839: "application/vnd.api+json"
	// → ".json", "image/svg+xml" → ".xml", etc. Check before exact-match
	// lookup so vendored types resolve correctly.
	if idx := strings.LastIndex(media, "+"); idx > 0 {
		switch media[idx+1:] {
		case "json":
			return ".json", true
		case "xml":
			return ".xml", true
		}
	}

	switch media {
	case "application/json", "text/json":
		return ".json", true
	case "text/html":
		return ".html", true
	case "application/xml", "text/xml":
		return ".xml", true
	case "text/plain":
		return ".txt", true
	case "text/csv":
		return ".csv", true
	case "application/pdf":
		return ".pdf", true
	}
	return "", false
}

// Write the content to the specified path with the appropriate file extension.
// Returns an error if the disk write fails so the caller can fail the task
// instead of silently losing the response file.
func writeFile(path, suffixType, contentBody string) error {
	fileName := utils.FileNameWithoutExtension(path) + utils.ResponseBase
	dir := filepath.Dir(path)
	fullFilePath := filepath.Join(dir, fileName+suffixType)
	if err := os.WriteFile(fullFilePath, []byte(contentBody), 0o600); err != nil {
		return fmt.Errorf("saving response %s: %w", fullFilePath, err)
	}
	return nil
}

// evalAndWriteRes picks the file extension via Content-Type (with body-sniff
// fallback) and writes resBody next to path.
func evalAndWriteRes(resBody, contentType, path string) error {
	if resBody == "" || path == "" {
		return utils.ColorError("Invalid input: file path and resBody cannot be empty")
	}
	return writeFile(path, extensionFor(contentType, resBody), resBody)
}
