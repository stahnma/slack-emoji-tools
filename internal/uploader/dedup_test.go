package uploader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsDuplicate_Match(t *testing.T) {
	content := []byte("identical-image-bytes")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	match, err := IsDuplicate(context.Background(), srv.Client(), srv.URL, content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !match {
		t.Fatal("expected match but got mismatch")
	}
}

func TestIsDuplicate_Mismatch(t *testing.T) {
	localContent := []byte("local-image-bytes")
	remoteContent := []byte("different-remote-image")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(remoteContent)
	}))
	defer srv.Close()

	match, err := IsDuplicate(context.Background(), srv.Client(), srv.URL, localContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match {
		t.Fatal("expected mismatch but got match")
	}
}

func TestIsDuplicate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := IsDuplicate(context.Background(), srv.Client(), srv.URL, []byte("data"))
	if err == nil {
		t.Fatal("expected error for server 500 response")
	}
}

func TestIsDuplicate_InvalidURL(t *testing.T) {
	_, err := IsDuplicate(context.Background(), nil, "http://127.0.0.1:0/nonexistent", []byte("data"))
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}

func TestLookupEmojiURL_Found(t *testing.T) {
	emojiMap := map[string]string{
		"parrot": "https://emoji.slack-edge.com/T123/parrot/abc123.gif",
	}
	url, ok := LookupEmojiURL(emojiMap, "parrot")
	if !ok {
		t.Fatal("expected to find emoji URL")
	}
	if url != "https://emoji.slack-edge.com/T123/parrot/abc123.gif" {
		t.Fatalf("unexpected URL: %s", url)
	}
}

func TestLookupEmojiURL_NotFound(t *testing.T) {
	emojiMap := map[string]string{}
	_, ok := LookupEmojiURL(emojiMap, "missing")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestLookupEmojiURL_Alias(t *testing.T) {
	emojiMap := map[string]string{
		"parrot": "alias:original_parrot",
	}
	_, ok := LookupEmojiURL(emojiMap, "parrot")
	if ok {
		t.Fatal("expected alias to be skipped")
	}
}
