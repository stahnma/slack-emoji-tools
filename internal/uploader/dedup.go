package uploader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// EmojiLister abstracts fetching the emoji list from Slack (for testability).
type EmojiLister interface {
	GetEmoji() (map[string]string, error)
}

// IsDuplicate checks if localData has the same SHA256 as the remote emoji at
// remoteURL. Returns true if they match (same image), false if they differ or
// the remote image cannot be fetched.
func IsDuplicate(ctx context.Context, httpClient *http.Client, remoteURL string, localData []byte) (bool, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return false, fmt.Errorf("creating request for remote emoji: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("downloading remote emoji: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("remote emoji returned status %d", resp.StatusCode)
	}

	remoteBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("reading remote emoji body: %w", err)
	}

	localHash := sha256.Sum256(localData)
	remoteHash := sha256.Sum256(remoteBody)
	return localHash == remoteHash, nil
}

// LookupEmojiURL finds the URL for a given emoji name in the emoji map.
// It returns the URL and true if found, or empty string and false if not found
// or if the entry is an alias (starts with "alias:").
func LookupEmojiURL(emojiMap map[string]string, name string) (string, bool) {
	url, ok := emojiMap[name]
	if !ok {
		return "", false
	}
	if strings.HasPrefix(url, "alias:") {
		return "", false
	}
	return url, true
}
