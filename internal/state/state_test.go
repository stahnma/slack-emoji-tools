package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()

	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load new: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Fatal("expected empty state")
	}

	s.RecordSuccess("emoji/partyparrot.gif", "partyparrot")
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	s2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load existing: %v", err)
	}
	entry, ok := s2.Entries["emoji/partyparrot.gif"]
	if !ok {
		t.Fatal("expected entry for partyparrot")
	}
	if entry.Name != "partyparrot" {
		t.Errorf("name = %q, want partyparrot", entry.Name)
	}
}

func TestStateIsUploaded(t *testing.T) {
	dir := t.TempDir()

	s, _ := Load(dir)
	if s.IsUploaded("emoji/partyparrot.gif") {
		t.Fatal("should not be uploaded yet")
	}
	s.RecordSuccess("emoji/partyparrot.gif", "partyparrot")
	if !s.IsUploaded("emoji/partyparrot.gif") {
		t.Fatal("should be uploaded")
	}
}

func TestConflictsRoundTrip(t *testing.T) {
	dir := t.TempDir()

	c, err := LoadConflicts(dir)
	if err != nil {
		t.Fatalf("Load new: %v", err)
	}

	c.RecordConflict("emoji/partyparrot.gif", "partyparrot", []string{"partyparrot"})
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	c2, err := LoadConflicts(dir)
	if err != nil {
		t.Fatalf("Load existing: %v", err)
	}
	entry, ok := c2.Entries["emoji/partyparrot.gif"]
	if !ok {
		t.Fatal("expected conflict entry")
	}
	if entry.Name != "partyparrot" {
		t.Errorf("name = %q, want partyparrot", entry.Name)
	}
}

func TestLoadCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")

	s, err := Load(dir)
	if err != nil {
		t.Fatalf("Load with nested dir: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Fatal("expected empty state")
	}
	// Verify directory was created
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
}

func TestLoadConflictsCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")

	c, err := LoadConflicts(dir)
	if err != nil {
		t.Fatalf("LoadConflicts with nested dir: %v", err)
	}
	if len(c.Entries) != 0 {
		t.Fatal("expected empty conflicts")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
}

func TestStateDir(t *testing.T) {
	dir, err := StateDir("T12345")
	if err != nil {
		t.Fatalf("StateDir: %v", err)
	}
	cfg, _ := os.UserConfigDir()
	expected := filepath.Join(cfg, "slack-emoji-tools", "state", "T12345")
	if dir != expected {
		t.Errorf("StateDir = %q, want %q", dir, expected)
	}
}
