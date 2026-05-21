package apicalls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

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

// readBody consumes an io.Reader and returns its bytes. Returns an empty
// slice when r is nil.
func readBody(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return io.ReadAll(r)
}
