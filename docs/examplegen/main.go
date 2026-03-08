package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	examplesDir := filepath.Join(filepath.Clean(filepath.Join(root, "..")), "examples")
	names, err := listExampleNames(examplesDir)
	if err != nil {
		fail(err)
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func listExampleNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
