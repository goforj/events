package examples

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExamplesBuild(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("cannot read examples directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(".", name)
		if _, err := os.Stat(filepath.Join(path, "main.go")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("cannot stat %q: %v", filepath.Join(path, "main.go"), err)
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := buildExample(path); err != nil {
				t.Fatalf("example %q failed to build:\n%s", name, err)
			}
		})
	}
}

func abs(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return absolute
}

func buildExample(exampleDir string) error {
	orig := filepath.Join(exampleDir, "main.go")

	src, err := os.ReadFile(orig)
	if err != nil {
		return fmt.Errorf("read main.go: %w", err)
	}

	clean := stripBuildTags(src)

	tmpDir, err := os.MkdirTemp("", "events-example-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpMain := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(tmpMain, clean, 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(exampleBuildGoMod()), 0o644); err != nil {
		return err
	}

	overlay := map[string]any{
		"Replace": map[string]string{
			abs(orig): abs(tmpMain),
		},
	}

	overlayJSON, err := json.Marshal(overlay)
	if err != nil {
		return err
	}

	overlayPath := filepath.Join(tmpDir, "overlay.json")
	if err := os.WriteFile(overlayPath, overlayJSON, 0o644); err != nil {
		return err
	}

	cmd := exec.Command(
		"go", "build",
		"-mod=mod",
		"-overlay", overlayPath,
		"-o", os.DevNull,
		".",
	)
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOWORK=off")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}

	return nil
}

func exampleBuildGoMod() string {
	root := abs("..")
	sep := string(filepath.Separator)
	rootSlash := filepath.ToSlash(root)
	if runtime.GOOS == "windows" {
		rootSlash = strings.ReplaceAll(root, sep, "/")
	}

	lines := []string{
		"module examplebuild",
		"",
		"go 1.24.0",
		"",
		"require (",
		"\tgithub.com/goforj/events v0.0.0",
		"\tgithub.com/goforj/events/eventscore v0.0.0",
		"\tgithub.com/goforj/events/driver/gcppubsubevents v0.0.0",
		"\tgithub.com/goforj/events/driver/kafkaevents v0.0.0",
		"\tgithub.com/goforj/events/driver/natsevents v0.0.0",
		"\tgithub.com/goforj/events/driver/redisevents v0.0.0",
		"\tgithub.com/goforj/events/driver/snsevents v0.0.0",
		")",
		"",
		"replace github.com/goforj/events => " + rootSlash,
		"replace github.com/goforj/events/eventscore => " + rootSlash + "/eventscore",
		"replace github.com/goforj/events/driver/gcppubsubevents => " + rootSlash + "/driver/gcppubsubevents",
		"replace github.com/goforj/events/driver/kafkaevents => " + rootSlash + "/driver/kafkaevents",
		"replace github.com/goforj/events/driver/natsevents => " + rootSlash + "/driver/natsevents",
		"replace github.com/goforj/events/driver/redisevents => " + rootSlash + "/driver/redisevents",
		"replace github.com/goforj/events/driver/snsevents => " + rootSlash + "/driver/snsevents",
		"",
	}

	return strings.Join(lines, "\n")
}

func stripBuildTags(src []byte) []byte {
	lines := strings.Split(string(src), "\n")

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if strings.HasPrefix(line, "//go:build") ||
			strings.HasPrefix(line, "// +build") ||
			line == "" {
			i++
			continue
		}

		break
	}

	return []byte(strings.Join(lines[i:], "\n"))
}
