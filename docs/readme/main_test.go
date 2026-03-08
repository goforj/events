package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReadme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	body := "# events\n\n## Installation\n\n## Drivers\n\n## Quick Start\n\nevents.NewSync()\n\n## Delivery Semantics\n\n## events vs queue\n\n## API Index\n\n<!-- test-count:embed:start -->\n<!-- test-count:embed:end -->\n\n<!-- api:embed:start -->\n<!-- api:embed:end -->\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := validateReadme(path, defaultRequiredContent()); err != nil {
		t.Fatalf("validateReadme returned error: %v", err)
	}
}

func TestValidateReadmeFailsOnMissingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("# events\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := validateReadme(path, defaultRequiredContent()); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestReplaceAPISection(t *testing.T) {
	body := "before\n<!-- api:embed:start -->\nold\n<!-- api:embed:end -->\nafter\n"

	got, err := replaceAPISection(body, renderAPISection())
	if err != nil {
		t.Fatalf("replaceAPISection returned error: %v", err)
	}
	if !strings.Contains(got, "### Core") {
		t.Fatal("expected generated API section content")
	}
	if !strings.Contains(got, "driver/natsevents") {
		t.Fatal("expected generated driver section content")
	}
}
