package yamlparser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/utils"
)

type HTTPMethodType string

// All supported methos
const (
	GET     HTTPMethodType = http.MethodGet
	POST    HTTPMethodType = http.MethodPost
	PUT     HTTPMethodType = http.MethodPut
	PATCH   HTTPMethodType = http.MethodPatch
	DELETE  HTTPMethodType = http.MethodDelete
	HEAD    HTTPMethodType = http.MethodHead
	OPTIONS HTTPMethodType = http.MethodOptions
	TRACE   HTTPMethodType = http.MethodTrace
	CONNECT HTTPMethodType = http.MethodConnect
)

// ToUpperCase convert the method to uppercase
func (h *HTTPMethodType) ToUpperCase() {
	*h = HTTPMethodType(strings.ToUpper(string(*h)))
}

// IsValid enforce HTTPMethodType
func (h HTTPMethodType) IsValid() bool {
	upperCasedMethod := HTTPMethodType(strings.ToUpper(string(h)))
	switch upperCasedMethod {
	case GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, TRACE, CONNECT:
		return true
	}
	return false
}

// Struct we need to call request api
type APIInfo struct {
	Body      io.Reader
	Headers   map[string]string
	URLParams map[string]string
	Method    string
	URL       string
}

// StreamedBody is a body whose size is known but which net/http cannot measure,
// because it is produced as it is read rather than held in memory.
//
// The size and the regenerator travel with the body rather than on APIInfo, so
// there is no field a constructor can leave at its zero value. That matters:
// net/http reads ContentLength == 0 on a non-nil body as "unknown" and quietly
// downgrades the request to chunked encoding.
type StreamedBody struct {
	io.ReadCloser
	// Length is the exact byte count the reader will produce.
	Length int64
	// GetBody rebuilds the body so net/http can replay it across a 307/308
	// redirect. Without it the client stops following and returns the redirect
	// as though it were the answer.
	GetBody func() (io.ReadCloser, error)
}

type URL string

// IsValidURL URL should not be missing
func (u *URL) IsValidURL() bool {
	userProvidedURL := string(*u)
	_, err := url.ParseRequestURI(userProvidedURL)
	return err == nil
}

// APICallFile represents user's yaml file for api request
type APICallFile struct {
	URLParams map[string]string `json:"urlparams,omitempty" yaml:"urlparams"`
	Headers   map[string]string `json:"headers,omitempty"   yaml:"headers"`
	Body      *Body             `json:"body,omitempty"      yaml:"body"`
	Method    HTTPMethodType    `json:"method,omitempty"    yaml:"method"`
	URL       URL               `json:"url,omitempty"       yaml:"url"`
}

// IsValid checks whether the user has valid file
func (user *APICallFile) IsValid(filePath string) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("requested api file is not valid")
	}

	user.Method.ToUpperCase()

	// method is required for any http request
	if !user.Method.IsValid() {
		if user.Method == "" {
			return false, fmt.Errorf("missing or empty HTTP method in '%s'", filePath)
		}
		return false, fmt.Errorf("invalid HTTP method '%s' in '%s'", user.Method, filePath)
	}

	// url is required for any http request
	if !user.URL.IsValidURL() {
		return false, fmt.Errorf("missing or invalid URL: %s in file %s", user.URL, filePath)
	}

	if !user.Body.IsValid() {
		return false, fmt.Errorf(
			"invalid Body in '%s': make sure body contains only one valid argument.\n %v",
			filePath,
			user.Body,
		)
	}
	return true, nil
}

// Returns APIInfo object for the User's API request yaml file
func (user *APICallFile) PrepareStruct() (APIInfo, error) {
	body, contentType, err := user.Body.EncodeBody()
	if err != nil {
		return APIInfo{}, fmt.Errorf("%s: %w", utils.ErrBodyEncoding, err)
	}

	if contentType != "" {
		if user.Headers == nil {
			user.Headers = make(map[string]string)
		}
		user.Headers["content-type"] = contentType
	}

	return APIInfo{
		Method:    string(user.Method),
		URL:       string(user.URL),
		URLParams: user.URLParams,
		Headers:   user.Headers,
		Body:      body,
	}, nil
}

// Body represents Body in a yaml file
// binary type is not yet configured
// Only one is possible that could be passed
type Body struct {
	FormData           map[string]string `json:"formdata,omitempty"           yaml:"formdata"`
	URLEncodedFormData map[string]string `json:"urlencodedformdata,omitempty" yaml:"urlencodedformdata"`
	Graphql            *GraphQl          `json:"graphql,omitempty"            yaml:"graphql"`
	Raw                string            `json:"raw,omitempty"                yaml:"raw"`
}

// IsValid checks whether body is valid when,
// if body is present, it's not nil
// has only one expected Body type,
// those body type is not empty,
// is not nil, and
// if the body has graphql key, it has at least query on it
func (b *Body) IsValid() bool {
	// it's allowed for yaml files to not have any body
	if b == nil {
		return true
	}
	validFieldCount := 0
	ln := reflect.ValueOf(*b)
	for i := range ln.NumField() {
		field := ln.Field(i)
		switch field.Kind() {
		case reflect.Pointer:
			// If the pointer is non-nil, it's valid
			if !field.IsNil() {
				validFieldCount++
			}
		case reflect.Map:
			// If the map has at least one element, it's valid
			if field.Len() > 0 {
				validFieldCount++
			}
		case reflect.String:
			// If the string is non-empty, it's valid
			if field.Len() > 0 {
				validFieldCount++
			}
		default:
			// If there's an unexpected kind, consider it invalid
			return false
		}
	}

	// Return true only if there's only 1 correct body type
	return validFieldCount == 1
}

// GraphQl inside body
type GraphQl struct {
	Query     string `json:"query"     yaml:"query"`
	Variables any    `json:"variables" yaml:"variables"`
}

// EncodeBody returns body for apiCall, content type header string and error if any
func (b *Body) EncodeBody() (io.Reader, string, error) {
	var body io.Reader
	var contentType string

	if b == nil {
		return nil, "", nil
	}

	switch {
	case b.Graphql != nil && b.Graphql.Query != "":
		encodedBody, err := EncodeGraphQlBody(b.Graphql.Query, b.Graphql.Variables)
		if err != nil {
			return nil, "", fmt.Errorf("error encoding GraphQL body: %w", err)
		}
		body = encodedBody

	case len(b.FormData) > 0:
		encodedBody, ct, err := EncodeFormData(b.FormData)
		if err != nil {
			return nil, "", fmt.Errorf("error encoding multipart form data: %w", err)
		}
		body, contentType = encodedBody, ct

	case len(b.URLEncodedFormData) > 0:
		encodedBody, err := EncodeXwwwFormURLBody(b.URLEncodedFormData)
		if err != nil {
			return nil, "", fmt.Errorf("error encoding URL-encoded form data: %w", err)
		}
		body, contentType = encodedBody, "application/x-www-form-urlencoded"

	case b.Raw != "":
		body = strings.NewReader(b.Raw)

	default:
		return nil, "", errors.New("no valid body type provided")
	}

	return body, contentType, nil
}

// EncodeXwwwFormURLBody encodes key-value pairs as "application/x-www-form-urlencoded" data.
// Returns an io.Reader containing the encoded data, or an error if the input is empty.
func EncodeXwwwFormURLBody(keyValue map[string]string) (io.Reader, error) {
	// Initialize form data
	formData := url.Values{}

	// Populate form data, using Set to overwrite duplicate keys if any
	for key, val := range keyValue {
		if key != "" && val != "" {
			formData.Set(key, val)
		}
	}

	// Return an error if no valid key-value pairs were found
	if len(formData) == 0 {
		return nil, errors.New("no valid key-value pairs to encode")
	}

	// Encode form data to "x-www-form-urlencoded" format
	return strings.NewReader(formData.Encode()), nil
}

// quoteEscaper mirrors the unexported escaper in mime/multipart, which applies
// it to names written into Content-Disposition.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, `\"`)

// fileContentType guesses a part's type from its extension, the same rule curl
// uses for -F. An unrecognized extension falls back to the generic type rather
// than guessing from content.
func fileContentType(path string) string {
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

// byteCounter tallies what is written to it and discards the bytes.
type byteCounter int64

func (c *byteCounter) Write(p []byte) (int, error) {
	*c += byteCounter(len(p))
	return len(p), nil
}

// formPart is one resolved multipart field. A file part holds an already-open
// handle so its size and its bytes come from the same descriptor: stat-ing a
// path and opening it later leaves a window where the file changes and the
// declared Content-Length becomes a lie.
type formPart struct {
	field    string
	value    string
	file     *os.File
	size     int64
	filename string
}

func (p formPart) isFile() bool { return p.file != nil }

// resolveFormParts walks keyValue once, opening every attachFile reference.
// One walk means the length and the streamed bytes can never disagree, even if
// the caller mutates the map afterwards.
func resolveFormParts(keyValue map[string]string) ([]formPart, error) {
	parts := make([]formPart, 0, len(keyValue))
	closeAll := func() {
		for _, p := range parts {
			if p.isFile() {
				_ = p.file.Close()
			}
		}
	}

	for key, val := range keyValue {
		if key == "" || val == "" {
			continue
		}
		path, ok := utils.FileRefPath(val)
		if !ok {
			if strings.Contains(val, utils.FileRefPrefix) {
				closeAll()
				return nil, fmt.Errorf(
					"field %q mixes attachFile with other text; a file must be the whole value",
					key,
				)
			}
			parts = append(parts, formPart{field: key, value: val})
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			closeAll()
			return nil, fmt.Errorf("attachFile %s: %w", path, err)
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			closeAll()
			return nil, fmt.Errorf("attachFile %s: %w", path, err)
		}
		parts = append(parts, formPart{
			field:    key,
			file:     file,
			size:     info.Size(),
			filename: filepath.Base(path),
		})
	}
	return parts, nil
}

// writePart emits one part. File contents stream from the open handle.
func writePart(writer *multipart.Writer, part formPart) error {
	if !part.isFile() {
		return writer.WriteField(part.field, part.value)
	}

	// Built by concatenation, not %q: quoteEscaper has already applied MIME
	// escaping, and %q would escape that again and mangle non-ASCII filenames.
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		`form-data; name="`+quoteEscaper.Replace(part.field)+
			`"; filename="`+quoteEscaper.Replace(part.filename)+`"`)
	header.Set("Content-Type", fileContentType(part.filename))

	w, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, part.file)
	return err
}

// formDataLength returns the exact byte count the parts will produce.
//
// It runs the real multipart writer into a counter rather than modelling the
// framing, so it cannot drift from what is actually emitted. File contents are
// added from the descriptor's size instead of being copied, so nothing is read.
func formDataLength(parts []formPart, boundary string) (int64, error) {
	var counter byteCounter
	writer := multipart.NewWriter(&counter)
	if err := writer.SetBoundary(boundary); err != nil {
		return 0, err
	}

	var fileBytes int64
	for _, part := range parts {
		if !part.isFile() {
			if err := writer.WriteField(part.field, part.value); err != nil {
				return 0, err
			}
			continue
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition",
			`form-data; name="`+quoteEscaper.Replace(part.field)+
				`"; filename="`+quoteEscaper.Replace(part.filename)+`"`)
		header.Set("Content-Type", fileContentType(part.filename))
		if _, err := writer.CreatePart(header); err != nil {
			return 0, err
		}
		fileBytes += part.size
	}
	// Close unconditionally: it emits the trailing boundary even with zero parts
	// written, and EncodeFormData closes on that same path.
	if err := writer.Close(); err != nil {
		return 0, err
	}
	return int64(counter) + fileBytes, nil
}

// EncodeFormData encodes multipart/form-data other than x-www-form-urlencoded.
// Returns the payload, the Content-Type header, the exact body length, and an
// error if any.
//
// The payload streams: parts are written on a goroutine as the consumer reads,
// so an attached file never sits in memory in full.
func EncodeFormData(keyValue map[string]string) (*StreamedBody, string, error) {
	if len(keyValue) == 0 {
		return nil, "", errors.New("no key-value pairs to encode")
	}

	parts, err := resolveFormParts(keyValue)
	if err != nil {
		return nil, "", err
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	// Read the boundary before the goroutine starts so the caller never races it.
	contentType := writer.FormDataContentType()

	boundary := writer.Boundary()
	length, err := formDataLength(parts, boundary)
	if err != nil {
		closeParts(parts)
		return nil, "", err
	}

	streamParts(writer, pw, parts)
	return &StreamedBody{
		ReadCloser: pr,
		Length:     length,
		// Pinned to the boundary already advertised in contentType: a replay
		// with a fresh boundary parses as zero parts on the far side.
		GetBody: func() (io.ReadCloser, error) {
			return encodeFormDataWith(keyValue, boundary)
		},
	}, contentType, nil
}

// encodeFormDataWith re-encodes under a boundary that is already advertised in
// a Content-Type header, which a replayed body must match.
func encodeFormDataWith(keyValue map[string]string, boundary string) (io.ReadCloser, error) {
	parts, err := resolveFormParts(keyValue)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	if err := writer.SetBoundary(boundary); err != nil {
		closeParts(parts)
		return nil, err
	}

	streamParts(writer, pw, parts)
	return pr, nil
}

func closeParts(parts []formPart) {
	for _, p := range parts {
		if p.isFile() {
			_ = p.file.Close()
		}
	}
}

func streamParts(writer *multipart.Writer, pw *io.PipeWriter, parts []formPart) {
	go func() {
		defer closeParts(parts)
		for _, part := range parts {
			// CloseWithError, not Close: a plain close would surface as a clean
			// EOF and the consumer would send a truncated body believing it whole.
			if err := writePart(writer, part); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
		_ = pw.CloseWithError(writer.Close())
	}()
}

// EncodeGraphQlBody accepts a query string and variables of any type,
// and returns an encoded GraphQL payload as an io.Reader
func EncodeGraphQlBody(query string, variables any) (io.Reader, error) {
	// Validate query
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("graphql query cannot be empty")
	}

	// Create the payload
	payload := GraphQl{
		Query: query,
	}

	// Handle variables if present
	if variables != nil {
		processed, err := processVariable(variables)
		if err != nil {
			return nil, fmt.Errorf("error processing variables: %w", err)
		}
		payload.Variables = processed
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL payload: %w", err)
	}

	return bytes.NewReader(jsonData), nil
}

// AddKeyValueToFormData is a helper function to dynamically add Key Value pair to FormData
func (b *Body) AddKeyValueToFormData(key, value string) {
	if b.FormData == nil {
		b.FormData = make(map[string]string)
	}
	b.FormData[key] = value
}

// AddKeyValueToURLEncodedFormData helper function to dynamically add Key Value pair to UrlEncodedFormData
func (b *Body) AddKeyValueToURLEncodedFormData(key, value string) {
	if b.URLEncodedFormData == nil {
		b.URLEncodedFormData = make(map[string]string)
	}
	b.URLEncodedFormData[key] = value
}

// processVariable handles different types of variables and ensures they're properly encoded
func processVariable(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		// Basic types can be used as-is
		return val, nil

	case time.Time:
		// Convert time to ISO 8601 format
		return val.Format(time.RFC3339), nil

	case []any:
		// Process array elements
		processed := make([]any, len(val))
		for i, item := range val {
			p, err := processVariable(item)
			if err != nil {
				return nil, err
			}
			processed[i] = p
		}
		return processed, nil

	case map[string]any:
		// Process nested maps
		processed := make(map[string]any, len(val))
		for k, item := range val {
			p, err := processVariable(item)
			if err != nil {
				return nil, err
			}
			processed[k] = p
		}
		return processed, nil

	case json.RawMessage:
		// Handle raw JSON
		var parsed any
		if err := json.Unmarshal(val, &parsed); err != nil {
			return nil, fmt.Errorf("invalid JSON in variable: %w", err)
		}
		return processVariable(parsed)

	default:
		// Try to convert other types to JSON and back to ensure they're properly encoded
		jsonData, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("unsupported variable type %T: %w", val, err)
		}

		var processed any
		if err := json.Unmarshal(jsonData, &processed); err != nil {
			return nil, fmt.Errorf("failed to process variable type %T: %w", val, err)
		}
		return processed, nil
	}
}
