package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func golden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func root(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunConflictAndSafeExitCodes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"check", "--root", root(t, "conflict-basic")}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("conflict exit = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GCC001") || !strings.Contains(stdout.String(), "release-Release Gateway") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"check", "--format", "json", "--root", root(t, "safe-caller-only")}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("safe exit = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"diagnostics": []`) {
		t.Fatalf("unexpected JSON: %q", stdout.String())
	}
}

func TestRunBoundaryAndFailureExitCodes(t *testing.T) {
	for _, name := range []string{"safe-distinct-literal", "unknown-dynamic-input", "graph-cycle"} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"check", "--format", "json", "--root", root(t, name)}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit = %d, stderr=%q", code, stderr.String())
			}
		})
	}

	for _, name := range []string{"invalid-yaml", "path-escape"} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"check", "--root", root(t, name)}, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("exit = %d, stdout=%q, stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunDirectorySymlinkBoundaryExitCodes(t *testing.T) {
	realRoot := root(t, "conflict-basic")
	alias := filepath.Join(t.TempDir(), "repository")
	if err := os.Symlink(realRoot, alias); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "--format", "json", "--root", alias}, &stdout, &stderr); code != 1 {
		t.Fatalf("root alias exit = %d, stderr=%q", code, stderr.String())
	}
	var aliasReport struct {
		Root string `json:"root"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &aliasReport); err != nil {
		t.Fatal(err)
	}
	if aliasReport.Root != filepath.ToSlash(realRoot) {
		t.Fatalf("root = %q, want canonical %q", aliasReport.Root, filepath.ToSlash(realRoot))
	}

	missing := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"check", "--root", missing}, &stdout, &stderr); code != 0 {
		t.Fatalf("missing workflows exit = %d, stderr=%q", code, stderr.String())
	}

	for _, component := range []string{".github", "workflows"} {
		t.Run(component, func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "outside")
			outsideWorkflows := filepath.Join(outside, ".github", "workflows")
			if err := os.MkdirAll(outsideWorkflows, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outsideWorkflows, "private.yml"), []byte("private: ["), 0o600); err != nil {
				t.Fatal(err)
			}

			testRoot := t.TempDir()
			if component == ".github" {
				if err := os.Symlink(filepath.Join(outside, ".github"), filepath.Join(testRoot, ".github")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.Mkdir(filepath.Join(testRoot, ".github"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outsideWorkflows, filepath.Join(testRoot, ".github", "workflows")); err != nil {
					t.Fatal(err)
				}
			}

			stdout.Reset()
			stderr.Reset()
			if code := run([]string{"check", "--root", testRoot}, &stdout, &stderr); code != 2 {
				t.Fatalf("%s symlink exit = %d, stdout=%q, stderr=%q", component, code, stdout.String(), stderr.String())
			}
			if strings.Contains(stderr.String(), "private: [") {
				t.Fatalf("stderr disclosed outside content: %q", stderr.String())
			}
		})
	}
}

func TestFanOutTextAndJSONGolden(t *testing.T) {
	fixtureRoot := root(t, "fan-out-multi-level")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"check", "--root", fixtureRoot}, &stdout, &stderr); code != 1 {
		t.Fatalf("text exit = %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != golden(t, "fan-out.txt") {
		t.Fatalf("text output:\n%s\nwant:\n%s", stdout.String(), golden(t, "fan-out.txt"))
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"check", "--format", "json", "--root", fixtureRoot}, &stdout, &stderr); code != 1 {
		t.Fatalf("JSON exit = %d, stderr=%q", code, stderr.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	decoded["root"] = "<ROOT>"
	normalized, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	normalized = append(normalized, '\n')
	if string(normalized) != golden(t, "fan-out.json") {
		t.Fatalf("JSON output:\n%s\nwant:\n%s", normalized, golden(t, "fan-out.json"))
	}
}
