package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	testCountStart = "<!-- test-count:embed:start -->"
	testCountEnd   = "<!-- test-count:embed:end -->"
)

type Counts struct {
	Unit        int
	Integration int
}

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println("Test badges updated from executed test runs")
}

func run() error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	unitCount, err := countModuleRuns(root, moduleDirs())
	if err != nil {
		return fmt.Errorf("count unit test runs: %w", err)
	}

	readmePath := filepath.Join(root, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}

	existing, _ := existingCountsFromReadme(string(data))

	integrationCount, err := countIntegrationRuns(filepath.Join(root, "integration"))
	if err != nil {
		fmt.Printf("warn: integration executed count unavailable (%v); leaving integration badge unchanged if present\n", err)
		integrationCount = existing.Integration
	}

	out, err := updateTestsSection(string(data), Counts{
		Unit:        unitCount,
		Integration: integrationCount,
	})
	if err != nil {
		return err
	}

	return os.WriteFile(readmePath, []byte(out), 0o644)
}

func moduleDirs() []string {
	return []string{
		".",
		"eventscore",
		"eventstest",
		"eventsfake",
		"examples",
		"docs",
		"driver/gcppubsubevents",
		"driver/kafkaevents",
		"driver/natsevents",
		"driver/redisevents",
	}
}

func countModuleRuns(root string, dirs []string) (int, error) {
	total := 0
	for _, dir := range dirs {
		count, err := countRunEvents(filepath.Join(root, dir), []string{"go", "test", "./...", "-run", "Test", "-count=1", "-json"})
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func countIntegrationRuns(integrationRoot string) (int, error) {
	names, err := integrationTopLevelTests(integrationRoot)
	if err != nil {
		return 0, err
	}
	runPattern := buildTopLevelRunPattern(names)
	if runPattern == "" {
		return 0, nil
	}
	return countRunEvents(integrationRoot, []string{"go", "test", "./...", "-run", runPattern, "-count=1", "-json"})
}

func countRunEvents(dir string, args []string) (int, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("%s (in %s): %w%s", strings.Join(args, " "), dir, err, summarizeOutput(out.String()))
	}

	total := 0
	dec := json.NewDecoder(bytes.NewReader(out.Bytes()))
	for dec.More() {
		var event struct {
			Action string `json:"Action"`
			Test   string `json:"Test"`
		}
		if err := dec.Decode(&event); err != nil {
			return 0, err
		}
		if event.Action == "run" && event.Test != "" {
			total++
		}
	}
	return total, nil
}

func buildTopLevelRunPattern(names map[string]struct{}) string {
	if len(names) == 0 {
		return ""
	}
	parts := make([]string, 0, len(names))
	for name := range names {
		parts = append(parts, regexp.QuoteMeta(name))
	}
	return "^(" + strings.Join(parts, "|") + ")(/.*)?$"
}

func integrationTopLevelTests(root string) (map[string]struct{}, error) {
	names := map[string]struct{}{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "vendor" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			if strings.HasPrefix(fn.Name.Name, "Test") {
				names[fn.Name.Name] = struct{}{}
			}
		}
		return nil
	})
	return names, err
}

func updateTestsSection(body string, counts Counts) (string, error) {
	start := strings.Index(body, testCountStart)
	end := strings.Index(body, testCountEnd)
	if start == -1 || end == -1 || end < start {
		return "", fmt.Errorf("README missing %s/%s markers", testCountStart, testCountEnd)
	}
	end += len(testCountEnd)

	replacement := fmt.Sprintf(`%s
    <img src="https://img.shields.io/badge/unit_tests-%d-brightgreen" alt="Unit tests (executed count)">
    <img src="https://img.shields.io/badge/integration_tests-%d-blue" alt="Integration tests (executed count)">
%s`, testCountStart, counts.Unit, counts.Integration, testCountEnd)

	return body[:start] + replacement + body[end:], nil
}

func existingCountsFromReadme(body string) (Counts, error) {
	unit, err := extractBadgeCount(body, "unit_tests-")
	if err != nil {
		return Counts{}, err
	}
	integration, err := extractBadgeCount(body, "integration_tests-")
	if err != nil {
		return Counts{}, err
	}
	return Counts{Unit: unit, Integration: integration}, nil
}

func extractBadgeCount(body string, prefix string) (int, error) {
	idx := strings.Index(body, prefix)
	if idx == -1 {
		return 0, fmt.Errorf("badge prefix %q not found", prefix)
	}
	start := idx + len(prefix)
	end := start
	for end < len(body) && body[end] >= '0' && body[end] <= '9' {
		end++
	}
	return strconv.Atoi(body[start:end])
}

func summarizeOutput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "{") {
			continue
		}
		lines = append(lines, line)
		if len(lines) == 4 {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n" + strings.Join(lines, "\n")
}

func findRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(root, "..")), nil
}
