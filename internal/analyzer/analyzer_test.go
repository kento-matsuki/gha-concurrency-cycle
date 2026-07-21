package analyzer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestConflictBasic(t *testing.T) {
	report, err := Analyze(fixture(t, "conflict-basic"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %d, want 1: %#v", len(report.Diagnostics), report.Diagnostics)
	}
	diagnostic := report.Diagnostics[0]
	if diagnostic.RuleID != RuleConcurrencyCollision {
		t.Fatalf("rule = %q, want %q", diagnostic.RuleID, RuleConcurrencyCollision)
	}
	if diagnostic.EffectiveGroup != "release-Release Gateway" {
		t.Fatalf("effective group = %q", diagnostic.EffectiveGroup)
	}
	if diagnostic.Caller.Path != ".github/workflows/gateway.yml" || diagnostic.Caller.Line != 6 {
		t.Fatalf("caller = %#v", diagnostic.Caller)
	}
	if diagnostic.Callee.Path != ".github/workflows/worker.yml" || diagnostic.Callee.Line != 7 {
		t.Fatalf("callee = %#v", diagnostic.Callee)
	}
	if diagnostic.CallSite.Path != ".github/workflows/gateway.yml" || diagnostic.CallSite.Line != 11 {
		t.Fatalf("call site = %#v", diagnostic.CallSite)
	}
}

func TestSafeCallerOnly(t *testing.T) {
	report, err := Analyze(fixture(t, "safe-caller-only"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", report.Diagnostics)
	}
}

func TestSafeDistinctLiteral(t *testing.T) {
	report, err := Analyze(fixture(t, "safe-distinct-literal"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || len(report.Unknowns) != 0 {
		t.Fatalf("report = %#v, want no diagnostics or unknowns", report)
	}
}

func TestUnknownDynamicInput(t *testing.T) {
	report, err := Analyze(fixture(t, "unknown-dynamic-input"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", report.Diagnostics)
	}
	if len(report.Unknowns) != 1 {
		t.Fatalf("unknowns = %#v, want one", report.Unknowns)
	}
	unknown := report.Unknowns[0]
	if unknown.Path != ".github/workflows/worker.yml" || unknown.Line != 7 || !strings.Contains(unknown.Reason, "unsupported expression") {
		t.Fatalf("unknown = %#v", unknown)
	}
}

func TestGraphCycleTerminatesDeterministically(t *testing.T) {
	first, err := Analyze(fixture(t, "graph-cycle"), "test")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Analyze(fixture(t, "graph-cycle"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Diagnostics) != 0 || len(first.Unknowns) != 0 {
		t.Fatalf("first report = %#v", first)
	}
	if len(second.Diagnostics) != 0 || len(second.Unknowns) != 0 {
		t.Fatalf("second report = %#v", second)
	}
}

func TestMalformedYAMLAndPathEscapeFail(t *testing.T) {
	for _, name := range []string{"invalid-yaml", "path-escape"} {
		t.Run(name, func(t *testing.T) {
			if _, err := Analyze(fixture(t, name), "test"); err == nil {
				t.Fatal("Analyze() error = nil, want failure")
			}
		})
	}
}

func TestWorkflowSymlinkIsRejectedBeforeRead(t *testing.T) {
	root := t.TempDir()
	workflowDir := filepath.Join(root, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.yml")
	if err := os.WriteFile(outside, []byte("private: [not valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(workflowDir, "linked.yml")); err != nil {
		t.Fatal(err)
	}
	_, err := Analyze(root, "test")
	if err == nil || !strings.Contains(err.Error(), "symbolic links are not allowed") {
		t.Fatalf("Analyze() error = %v", err)
	}
}

func TestRootAliasesAreCanonicalized(t *testing.T) {
	realRoot := fixture(t, "conflict-basic")
	alias := filepath.Join(t.TempDir(), "repository")
	if err := os.Symlink(realRoot, alias); err != nil {
		t.Fatal(err)
	}

	report, err := Analyze(alias, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one", report.Diagnostics)
	}
	if report.Root != filepath.ToSlash(realRoot) {
		t.Fatalf("root = %q, want canonical %q", report.Root, filepath.ToSlash(realRoot))
	}
}

func TestInternalWorkflowDirectorySymlinksAreRejectedBeforeRead(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside")
	outsideWorkflowDir := filepath.Join(outside, ".github", "workflows")
	if err := os.MkdirAll(outsideWorkflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	privateMarker := "private-content-must-not-be-read"
	if err := os.WriteFile(filepath.Join(outsideWorkflowDir, "outside.yml"), []byte(privateMarker+": ["), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		component string
		setup     func(string) error
	}{
		{
			name:      "github",
			component: ".github",
			setup: func(root string) error {
				return os.Symlink(filepath.Join(outside, ".github"), filepath.Join(root, ".github"))
			},
		},
		{
			name:      "workflows",
			component: ".github/workflows",
			setup: func(root string) error {
				if err := os.Mkdir(filepath.Join(root, ".github"), 0o755); err != nil {
					return err
				}
				return os.Symlink(outsideWorkflowDir, filepath.Join(root, ".github", "workflows"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := test.setup(root); err != nil {
				t.Fatal(err)
			}
			_, err := Analyze(root, "test")
			if err == nil || !strings.Contains(err.Error(), test.component) {
				t.Fatalf("Analyze() error = %v, want component %q", err, test.component)
			}
			if strings.Contains(err.Error(), privateMarker) {
				t.Fatalf("error disclosed outside content: %v", err)
			}
		})
	}
}

func TestMissingWorkflowsIsEmpty(t *testing.T) {
	report, err := Analyze(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || len(report.Unknowns) != 0 {
		t.Fatalf("report = %#v, want empty", report)
	}
}

func TestFanOutAndMultiLevelOrdering(t *testing.T) {
	report, err := Analyze(fixture(t, "fan-out-multi-level"), "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %#v, want two", report.Diagnostics)
	}
	if report.Diagnostics[0].Callee.Path != ".github/workflows/worker-a.yaml" || report.Diagnostics[0].CallSite.Line != 12 {
		t.Fatalf("first diagnostic = %#v", report.Diagnostics[0])
	}
	if report.Diagnostics[1].Callee.Path != ".github/workflows/worker-z.yml" || report.Diagnostics[1].CallSite.Line != 10 {
		t.Fatalf("second diagnostic = %#v", report.Diagnostics[1])
	}
}

func TestWorkflowResourceLimits(t *testing.T) {
	t.Run("file-size", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, ".github", "workflows")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := bytes.Repeat([]byte("x"), MaxWorkflowFileBytes+1)
		if err := os.WriteFile(filepath.Join(dir, "large.yml"), content, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Analyze(root, "test")
		if err == nil || !strings.Contains(err.Error(), "workflow file exceeds 1048576 bytes") {
			t.Fatalf("Analyze() error = %v", err)
		}
	})

	t.Run("file-count", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, ".github", "workflows")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for i := 0; i <= MaxWorkflowFiles; i++ {
			name := filepath.Join(dir, fmt.Sprintf("workflow-%03d.yml", i))
			if err := os.WriteFile(name, []byte("name: bounded\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		_, err := Analyze(root, "test")
		if err == nil || !strings.Contains(err.Error(), "workflow file count exceeds 256") {
			t.Fatalf("Analyze() error = %v", err)
		}
	})
}
