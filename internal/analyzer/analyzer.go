package analyzer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const RuleConcurrencyCollision = "GCC001"

const (
	MaxWorkflowFiles     = 256
	MaxWorkflowFileBytes = 1 << 20
)

var workflowExpression = regexp.MustCompile(`\$\{\{\s*github\.workflow\s*\}\}`)

type Location struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

type Diagnostic struct {
	RuleID         string   `json:"ruleId"`
	Severity       string   `json:"severity"`
	EffectiveGroup string   `json:"effectiveGroup"`
	Caller         Location `json:"caller"`
	Callee         Location `json:"callee"`
	CallSite       Location `json:"callSite"`
	Message        string   `json:"message"`
	Remediation    string   `json:"remediation"`
}

type Unknown struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

type Report struct {
	SchemaVersion int          `json:"schemaVersion"`
	ToolVersion   string       `json:"toolVersion"`
	Root          string       `json:"root"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
	Unknowns      []Unknown    `json:"unknowns"`
}

type workflow struct {
	path         string
	name         string
	group        string
	groupLine    int
	localCallees []callEdge
}

type callEdge struct {
	path string
	line int
}

func Analyze(root, version string) (Report, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(canonicalRoot)
	if err != nil {
		return Report{}, fmt.Errorf("read root: %w", err)
	}
	if !info.IsDir() {
		return Report{}, fmt.Errorf("root is not a directory: %s", root)
	}

	report := Report{
		SchemaVersion: 1,
		ToolVersion:   version,
		Root:          filepath.ToSlash(canonicalRoot),
		Diagnostics:   []Diagnostic{},
		Unknowns:      []Unknown{},
	}

	workflows, err := loadWorkflows(canonicalRoot)
	if err != nil {
		return Report{}, err
	}

	for _, caller := range workflows {
		if caller.name == "" || caller.group == "" || len(caller.localCallees) == 0 {
			continue
		}
		callerGroup, callerKnown := resolveGroup(caller.group, caller.name)
		if !callerKnown {
			report.Unknowns = append(report.Unknowns, Unknown{Path: caller.path, Line: caller.groupLine, Reason: "caller concurrency group contains an unsupported expression"})
			continue
		}
		for _, edge := range caller.localCallees {
			callee, ok := workflows[edge.path]
			if !ok || callee.group == "" {
				continue
			}
			calleeGroup, calleeKnown := resolveGroup(callee.group, caller.name)
			if !calleeKnown {
				report.Unknowns = append(report.Unknowns, Unknown{Path: callee.path, Line: callee.groupLine, Reason: "called workflow concurrency group contains an unsupported expression"})
				continue
			}
			if !strings.EqualFold(callerGroup, calleeGroup) {
				continue
			}
			report.Diagnostics = append(report.Diagnostics, Diagnostic{
				RuleID:         RuleConcurrencyCollision,
				Severity:       "error",
				EffectiveGroup: callerGroup,
				Caller:         Location{Path: caller.path, Line: caller.groupLine},
				Callee:         Location{Path: callee.path, Line: callee.groupLine},
				CallSite:       Location{Path: caller.path, Line: edge.line},
				Message:        "caller and called workflow resolve to the same workflow-level concurrency group",
				Remediation:    "keep concurrency ownership in the caller and remove it from the called workflow",
			})
		}
	}

	sort.Slice(report.Diagnostics, func(i, j int) bool {
		a, b := report.Diagnostics[i], report.Diagnostics[j]
		if a.Caller.Path != b.Caller.Path {
			return a.Caller.Path < b.Caller.Path
		}
		if a.Callee.Path != b.Callee.Path {
			return a.Callee.Path < b.Callee.Path
		}
		return a.CallSite.Line < b.CallSite.Line
	})
	sort.Slice(report.Unknowns, func(i, j int) bool {
		if report.Unknowns[i].Path != report.Unknowns[j].Path {
			return report.Unknowns[i].Path < report.Unknowns[j].Path
		}
		return report.Unknowns[i].Line < report.Unknowns[j].Line
	})

	return report, nil
}

func loadWorkflows(root string) (map[string]workflow, error) {
	githubDir := filepath.Join(root, ".github")
	exists, err := validateWorkflowDirectory(githubDir, ".github")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]workflow{}, nil
	}

	dir := filepath.Join(githubDir, "workflows")
	exists, err = validateWorkflowDirectory(dir, ".github/workflows")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]workflow{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read workflows: %w", err)
	}

	candidates := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("read workflows: symbolic links are not allowed: %s", entry.Name())
		}
		candidates = append(candidates, entry)
		if len(candidates) > MaxWorkflowFiles {
			return nil, fmt.Errorf("read workflows: workflow file count exceeds %d", MaxWorkflowFiles)
		}
	}

	result := make(map[string]workflow, len(candidates))
	for _, entry := range candidates {
		absolutePath := filepath.Join(dir, entry.Name())
		data, err := readBoundedWorkflowFile(root, absolutePath, MaxWorkflowFileBytes)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		rel, err := filepath.Rel(root, absolutePath)
		if err != nil {
			return nil, fmt.Errorf("relativize %s: %w", entry.Name(), err)
		}
		parsed, err := parseWorkflow(filepath.ToSlash(rel), &document, root)
		if err != nil {
			return nil, err
		}
		result[parsed.path] = parsed
	}
	return result, nil
}

func validateWorkflowDirectory(path, label string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read workflows: inspect %s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("read workflows: symbolic directory links are not allowed: %s", label)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("read workflows: not a directory: %s", label)
	}
	return true, nil
}

func readBoundedWorkflowFile(root, path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symbolic links are not allowed")
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("workflow is not a regular file")
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	withinRoot, err := filepath.Rel(root, resolved)
	if err != nil || withinRoot == ".." || strings.HasPrefix(withinRoot, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("workflow resolves outside repository root")
	}
	if filepath.Clean(resolved) != filepath.Clean(path) {
		return nil, fmt.Errorf("symbolic links are not allowed")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("workflow file exceeds %d bytes", limit)
	}
	return data, nil
}

func parseWorkflow(path string, document *yaml.Node, root string) (workflow, error) {
	parsed := workflow{path: path}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return parsed, fmt.Errorf("parse %s: workflow root must be a mapping", path)
	}
	rootNode := document.Content[0]
	if name := mapValue(rootNode, "name"); name != nil && name.Kind == yaml.ScalarNode {
		parsed.name = name.Value
	}
	if concurrency := mapValue(rootNode, "concurrency"); concurrency != nil {
		group := concurrency
		if concurrency.Kind == yaml.MappingNode {
			group = mapValue(concurrency, "group")
		}
		if group != nil && group.Kind == yaml.ScalarNode {
			parsed.group = group.Value
			parsed.groupLine = group.Line
		}
	}
	jobs := mapValue(rootNode, "jobs")
	if jobs == nil || jobs.Kind != yaml.MappingNode {
		return parsed, nil
	}
	for i := 0; i+1 < len(jobs.Content); i += 2 {
		job := jobs.Content[i+1]
		if job.Kind != yaml.MappingNode {
			continue
		}
		uses := mapValue(job, "uses")
		if uses == nil || uses.Kind != yaml.ScalarNode || !strings.HasPrefix(uses.Value, "./.github/workflows/") {
			continue
		}
		target, err := safeWorkflowPath(root, uses.Value)
		if err != nil {
			return parsed, fmt.Errorf("parse %s:%d: %w", path, uses.Line, err)
		}
		parsed.localCallees = append(parsed.localCallees, callEdge{path: target, line: uses.Line})
	}
	return parsed, nil
}

func safeWorkflowPath(root, raw string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(raw, "./")))
	absolute := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("local workflow path escapes repository root")
	}
	return filepath.ToSlash(rel), nil
}

func mapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func resolveGroup(group, workflowName string) (string, bool) {
	resolved := workflowExpression.ReplaceAllString(group, workflowName)
	if strings.Contains(resolved, "${{") {
		return "", false
	}
	return resolved, true
}
