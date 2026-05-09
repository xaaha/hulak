package vault

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/httpclient"
)

// HTTP client and URL construction for fetching SSH public keys
// from GitHub or compatible keyservers.

// GitHubKeysBase is the default keyserver base URL for GitHub.
// Pass to KeyserverKeysURL for GitHub users.
const GitHubKeysBase = "https://github.com"

const (
	keysSuffix       = ".keys"
	maxBodyBytes     = 1 << 20 // 1 MiB
	fetchKeysTimeout = 5 * time.Second
)

// KeyserverKeysURL returns the URL for a user's published SSH keys on a
// keyserver. Use GitHubKeysBase for GitHub, or pass a custom base for
// GitLab/Forgejo/etc. Trailing slashes on baseURL are trimmed.
func KeyserverKeysURL(baseURL, username string) string {
	return strings.TrimRight(baseURL, "/") + "/" + username + keysSuffix
}

// FetchKeysFromURL fetches SSH public keys from the given HTTPS URL.
//
// Pass a non-nil client for testing; nil uses httpclient.New().
// Timeout is 5 seconds via context, not client-level — same pattern as the runner.
//
// Returns the non-empty lines from the response body. Errors on non-HTTPS URLs,
// 404 responses, and empty bodies.
func FetchKeysFromURL(url string, client *httpclient.Client) ([]string, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf(
			"refusing to fetch keys over insecure transport — HTTPS required: %s",
			url,
		)
	}

	if client == nil {
		client = httpclient.New()
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchKeysTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching keys from %s: %w", url, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("no keys published at %s — check the username", url)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	body, err := httpclient.ReadBody(resp, maxBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("no keys found at %s", url)
	}

	var keys []string
	// github returns one public key per line
	for line := range strings.SplitSeq(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// FilterKeysByType categorizes SSH public key strings by type using
// ClassifyKeyType. Keys are sorted into ed25519, RSA, and skipped buckets.
func FilterKeysByType(keys []string) (ed25519Keys, rsaKeys, skippedKeys []string) {
	for _, key := range keys {
		switch ClassifyKeyType(key) {
		case sshEd25519:
			ed25519Keys = append(ed25519Keys, key)
		case sshRSA:
			rsaKeys = append(rsaKeys, key)
		default:
			skippedKeys = append(skippedKeys, key)
		}
	}
	return
}
