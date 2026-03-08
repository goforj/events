package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListExampleNames(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"newsync", "newnull"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatalf("Mkdir returned error: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	got, err := listExampleNames(dir)
	if err != nil {
		t.Fatalf("listExampleNames returned error: %v", err)
	}
	want := []string{"newnull", "newsync"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("listExampleNames = %v, want %v", got, want)
	}
}
