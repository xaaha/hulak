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

// DryRun builds the request at opts.Path (resolving templates with
// opts.Secrets) and returns its formatted wire representation as a string,
// without sending anything. opts.Show controls sensitive-header masking. Used
// by non-terminal callers such as the MCP dry_run tool.
func DryRun(opts RequestOptions) (string, error) {
	apiConfig, _, err := yamlparser.FinalStructForAPI(opts.Path, opts.Secrets)
	if err != nil {
		return "", err
	}
	apiInfo, err := apiConfig.PrepareStruct()
	if err != nil {
		return "", err
	}
	return FormatDryRun(&apiInfo, opts.Show)
}

// PrintDryRun writes the fully-built request to stdout and returns. It
// performs no I/O — no transport, no response file, no follow-up. Use to
// verify the wire shape of a request before sending it.
//
// Body is read from apiInfo.Body, which consumes the reader. Callers must
// not rely on apiInfo.Body after this call.
func PrintDryRun(apiInfo *yamlparser.APIInfo, show bool) error {
	out, err := FormatDryRun(apiInfo, show)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// FormatDryRun builds the fully-resolved request into a printable string and
// returns it. It performs no transport and writes no files — use it to
// inspect the wire shape of a request before sending it, whether that's for
// stdout (PrintDryRun) or a non-terminal caller like the MCP dry_run tool.
//
// Sensitive headers (Authorization, Cookie, etc.) are masked unless show
// is true. Body is pretty-printed when JSON, otherwise written verbatim.
//
// Body is read from apiInfo.Body, which consumes the reader. Callers must
// not rely on apiInfo.Body after this call.
func FormatDryRun(apiInfo *yamlparser.APIInfo, show bool) (string, error) {
	var b strings.Builder

	reqURL := PrepareURL(apiInfo.URL, apiInfo.URLParams)
	fmt.Fprintf(&b, "%s %s\n", apiInfo.Method, reqURL)

	headers := utils.RedactHeaders(apiInfo.Headers, show)
	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Fprintf(&b, "%s: %s\n", k, headers[k])
	}

	ct := contentTypeOf(apiInfo.Headers)

	// A streamed body carries an attached file. Never read it into memory just
	// to print it: describe it by size instead, so a dry run of a large upload
	// stays cheap and never dumps binary to the output.
	if streamed, ok := apiInfo.Body.(*yamlparser.StreamedBody); ok {
		b.WriteByte('\n')
		writeStreamedDryRun(&b, streamed, ct)
		return b.String(), nil
	}

	body, err := readBody(apiInfo.Body)
	if err != nil {
		return "", fmt.Errorf("reading request body: %w", err)
	}
	if len(body) == 0 {
		return b.String(), nil
	}

	b.WriteByte('\n')
	if pretty, ok := prettyFormBody(body, ct); ok {
		b.WriteString(pretty)
		return b.String(), nil
	}
	if isJSONContentType(ct) || IsJSON(string(body)) {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			b.WriteString(pretty.String())
			b.WriteByte('\n')
			return b.String(), nil
		}
	}
	b.WriteString(string(body))
	b.WriteByte('\n')
	return b.String(), nil
}

// isJSONContentType reports whether contentType is a JSON media type
// (application/json or any +json suffix like application/vnd.api+json).
func isJSONContentType(contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return media == "application/json" || strings.HasSuffix(media, "+json")
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

// writeStreamedDryRun renders an attached-file body without sending it. A
// whole-file body is summarized by size. A multipart body is expanded into its
// field and file summaries only when it fits the display cap, so a large
// attachment is never pulled into memory to be printed.
func writeStreamedDryRun(b *strings.Builder, sb *yamlparser.StreamedBody, ct string) {
	defer func() { _ = sb.Close() }()

	if sb.SuggestedContentType != "" {
		fmt.Fprintf(b, "<file body: %d bytes>\n", sb.Length)
		return
	}

	if sb.Length > debugBodyLimit {
		fmt.Fprintf(b, "<multipart/form-data body: %d bytes>\n", sb.Length)
		return
	}
	data, err := io.ReadAll(sb)
	if err != nil {
		fmt.Fprintf(b, "<multipart/form-data body: %d bytes>\n", sb.Length)
		return
	}
	if pretty, ok := prettyFormBody(data, ct); ok {
		b.WriteString(pretty)
		return
	}
	fmt.Fprintf(b, "<multipart/form-data body: %d bytes>\n", sb.Length)
}

// readBody consumes an io.Reader and returns its bytes. Returns an empty
// slice when r is nil.
func readBody(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return io.ReadAll(r)
}
