package userflags

import (
	"fmt"
)

var version = "dev"

// requestSchema holds the hulak request JSON Schema, injected from main via
// SetRequestSchema (main owns the //go:embed of assets/). Passed to the mcp
// command so write_request can validate content.
var requestSchema []byte

// SetRequestSchema injects the embedded request schema. Call before building
// the command tree.
func SetRequestSchema(b []byte) { requestSchema = b }

func getVersion() {
	fmt.Printf("%s\n", version)
}
