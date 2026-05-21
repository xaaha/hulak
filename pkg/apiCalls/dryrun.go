package apicalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"sort"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// PrintDryRun writes the fully-built request to stdout and returns. It
// performs no I/O — no transport, no response file, no follow-up. Use to
// verify the wire shape of a request before sending it.
//
// Sensitive headers (Authorization, Cookie, etc.) are masked unless show
// is true. Body is pretty-printed when JSON, otherwise written verbatim.
//
// Body is read from apiInfo.Body, which consumes the reader. Callers must
// not rely on apiInfo.Body after this call.
func PrintDryRun(apiInfo *yamlparser.APIInfo, show bool) error {
	url := PrepareURL(apiInfo.URL, apiInfo.URLParams)
	fmt.Printf("%s %s\n", apiInfo.Method, url)

	headers := utils.RedactHeaders(apiInfo.Headers, show)
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Printf("%s: %s\n", k, headers[k])
	}

	body, err := readBody(apiInfo.Body)
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	if len(body) == 0 {
		return nil
	}

	fmt.Println()
	ct := contentTypeOf(apiInfo.Headers)
	if pretty, ok := prettyFormBody(body, ct); ok {
		fmt.Print(pretty)
		return nil
	}
	if IsJSON(string(body)) {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			fmt.Println(pretty.String())
			return nil
		}
	}
	fmt.Println(string(body))
	return nil
}

// contentTypeOf returns the Content-Type header value from headers,
// case-insensitively. Returns "" if absent.
func contentTypeOf(headers map[string]string) string {
	for k, v := range headers {
		if strings.EqualFold(k, "content-type") {
			return v
		}
	}
	return ""
}

// prettyFormBody decodes multipart/form-data and application/x-www-form-urlencoded
// payloads into a readable "field: value" listing. Returns (output, true) when
// the content type matches and decoding succeeds; otherwise (_, false) so the
// caller falls back to the verbatim body print.
//
// File parts in multipart are summarized as "<file: <filename>, N bytes>" so
// binary payloads do not flood stdout.
func prettyFormBody(body []byte, contentType string) (string, bool) {
	media, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", false
	}
	switch media {
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return "", false
		}
		return formatFormFields(values), true
	case "multipart/form-data":
		boundary, ok := params["boundary"]
		if !ok {
			return "", false
		}
		fields, err := readMultipartFields(body, boundary)
		if err != nil {
			return "", false
		}
		return formatFormFields(fields), true
	}
	return "", false
}

// readMultipartFields walks a multipart payload and returns each part as a
// "name -> values" map. File parts are represented as a summary string so
// binary content does not get printed.
func readMultipartFields(body []byte, boundary string) (url.Values, error) {
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	out := url.Values{}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		name := part.FormName()
		if filename := part.FileName(); filename != "" {
			content, _ := io.ReadAll(part)
			out.Add(name, fmt.Sprintf("<file: %s, %d bytes>", filename, len(content)))
			_ = part.Close()
			continue
		}
		content, err := io.ReadAll(part)
		_ = part.Close()
		if err != nil {
			return nil, err
		}
		out.Add(name, string(content))
	}
}

// formatFormFields renders url.Values as deterministic "name: value" lines
// for dry-run output. Repeated keys produce multiple lines.
func formatFormFields(values url.Values) string {
	names := make([]string, 0, len(values))
	for k := range values {
		names = append(names, k)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		for _, v := range values[k] {
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// readBody consumes an io.Reader and returns its bytes. Returns an empty
// slice when r is nil.
func readBody(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return io.ReadAll(r)
}
