package checks

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type policy struct {
	SchemaVersion        int               `json:"schema_version"`
	Root                 string            `json:"root"`
	AllowedTopLevelRoots topLevelRoots     `json:"allowed_top_level_roots"`
	InternalPackages     []string          `json:"internal_packages"`
	ForbiddenImports     []importRule      `json:"forbidden_imports"`
	ForbiddenPackages    []packageRule     `json:"forbidden_packages"`
	ForbiddenPaths       []pathRule        `json:"forbidden_paths"`
	MigrationFiles       migrationRule     `json:"migration_files"`
	ContractDocuments    contractRule      `json:"contract_documents"`
	Locations            map[string]string `json:"locations"`
	Transport            transportRule     `json:"transport"`
}

type topLevelRoots struct {
	Rule    string   `json:"rule"`
	Names   []string `json:"names"`
	Message string   `json:"message"`
}

type packageRule struct {
	Under   string   `json:"under"`
	Rule    string   `json:"rule"`
	Names   []string `json:"names"`
	Message string   `json:"message"`
}

type importRule struct {
	Rule    string   `json:"rule"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Message string   `json:"message"`
}

type pathRule struct {
	Rule    string `json:"rule"`
	Pattern string `json:"pattern"`
	Message string `json:"message"`
}

type migrationRule struct {
	Rule               string   `json:"rule"`
	AllowedRoot        string   `json:"allowed_root"`
	FilePatterns       []string `json:"file_patterns"`
	ForbiddenElsewhere []string `json:"forbidden_elsewhere"`
	Message            string   `json:"message"`
}

type contractRule struct {
	Rule           string   `json:"rule"`
	AllowedRoot    string   `json:"allowed_root"`
	FilePatterns   []string `json:"file_patterns"`
	ForbiddenRoots []string `json:"forbidden_roots"`
	Message        string   `json:"message"`
}

type transportRule struct {
	Root                          string   `json:"root"`
	SubdirectoriesMustBeProtocols bool     `json:"subdirectories_must_be_protocols"`
	AllowedProtocols              []string `json:"allowed_protocols"`
}

type violation struct {
	Rule    string
	Path    string
	Message string
}

func (v violation) String() string {
	return fmt.Sprintf("topology violation: rule=%s path=%s message=%s", v.Rule, v.Path, v.Message)
}

func assertViolation(t *testing.T, files map[string]string, rule string, path string) {
	t.Helper()

	root := fixture(t, files)
	policy := mustLoadPolicy(t)
	violations, err := scanTree(root, policy)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range violations {
		if item.Rule == rule && item.Path == path {
			return
		}
	}
	t.Fatalf("expected violation rule=%s path=%s, got %v", rule, path, violations)
}

func assertClean(t *testing.T, files map[string]string) {
	t.Helper()

	root := fixture(t, files)
	policy := mustLoadPolicy(t)
	violations, err := scanTree(root, policy)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("expected clean fixture, got %v", violations)
	}
}

func mustLoadPolicy(t *testing.T) policy {
	t.Helper()

	item, err := loadPolicy(findRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func loadPolicy(root string) (policy, error) {
	path := filepath.Join(root, "quality", "boundary-manifest.json")
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return policy{}, errors.New("boundary manifest missing: quality/boundary-manifest.json")
		}
		return policy{}, err
	}
	defer file.Close()

	var item policy
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return policy{}, fmt.Errorf("boundary manifest invalid: %w", err)
	}
	if item.SchemaVersion != 1 {
		return policy{}, fmt.Errorf("boundary manifest invalid: unsupported schema_version %d", item.SchemaVersion)
	}
	if err := validatePolicyData(item); err != nil {
		return policy{}, err
	}
	if len(item.ForbiddenImports) == 0 {
		return item, nil
	}
	graph, err := loadPackageGraph(root)
	if err != nil {
		return policy{}, err
	}
	if err := validateImportRules(item.ForbiddenImports, graph); err != nil {
		return policy{}, err
	}
	return item, nil
}

func validatePolicyData(item policy) error {
	if item.Root == "" || item.AllowedTopLevelRoots.Rule == "" || len(item.AllowedTopLevelRoots.Names) == 0 {
		return errors.New("boundary manifest invalid: required rule data is missing")
	}
	return nil
}

func validateImportRules(rules []importRule, graph packageGraph) error {
	for _, rule := range rules {
		if importRuleDataMissing(rule) {
			return errors.New("boundary manifest invalid: forbidden import rule data is missing")
		}
		if containsBlank(rule.From) || containsBlank(rule.To) {
			return errors.New("boundary manifest invalid: forbidden import paths include blank value")
		}
		if err := validateImportSources(rule.From, graph); err != nil {
			return err
		}
		if err := validateImportTargets(rule.To, graph); err != nil {
			return err
		}
	}
	return nil
}

func importRuleDataMissing(rule importRule) bool {
	return strings.TrimSpace(rule.Rule) == "" ||
		strings.TrimSpace(rule.Message) == "" ||
		len(rule.From) == 0 ||
		len(rule.To) == 0
}

func containsBlank(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func validateImportSources(sources []string, graph packageGraph) error {
	for _, source := range sources {
		if _, ok := graph.sourcePackage(source); !ok {
			return fmt.Errorf("boundary manifest invalid: forbidden import source %q does not resolve", source)
		}
	}
	return nil
}

func validateImportTargets(targets []string, graph packageGraph) error {
	for _, target := range targets {
		if _, ok := graph.importPackage(target); !ok {
			return fmt.Errorf("boundary manifest invalid: forbidden import target %q does not resolve", target)
		}
	}
	return nil
}

func scanTree(root string, item policy) ([]violation, error) {
	var violations []violation
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if shouldIgnore(relative, entry) {
			return nil
		}

		violations = append(violations, violationsForPath(relative, entry, item)...)
		return nil
	})
	return violations, err
}

func violationsForPath(path string, entry os.DirEntry, item policy) []violation {
	var violations []violation
	violations = append(violations, topLevelViolation(path, item)...)
	violations = append(violations, packageViolations(path, entry, item)...)
	violations = append(violations, pathViolations(path, item)...)
	violations = append(violations, migrationViolations(path, item)...)
	violations = append(violations, contractViolations(path, item)...)
	violations = append(violations, transportViolations(path, item)...)
	return violations
}

func topLevelViolation(path string, item policy) []violation {
	root := firstSegment(path)
	if slices.Contains(item.AllowedTopLevelRoots.Names, root) {
		return nil
	}
	return []violation{{
		Rule:    item.AllowedTopLevelRoots.Rule,
		Path:    root,
		Message: item.AllowedTopLevelRoots.Message,
	}}
}

func packageViolations(path string, entry os.DirEntry, item policy) []violation {
	if entry.IsDir() || filepath.Ext(path) != ".go" {
		return nil
	}

	var violations []violation
	for _, rule := range item.ForbiddenPackages {
		violations = append(violations, forbiddenPackageViolations(path, rule)...)
	}
	if unlistedInternalPackage(path, item.InternalPackages) {
		violations = append(violations, violation{
			Rule:    "package-without-manifest",
			Path:    packagePath(path),
			Message: "internal packages require manifest entries",
		})
	}
	return violations
}

func forbiddenPackageViolations(path string, rule packageRule) []violation {
	if !strings.HasPrefix(path, rule.Under+"/") {
		return nil
	}

	var violations []violation
	for _, segment := range strings.Split(strings.TrimPrefix(path, rule.Under+"/"), "/") {
		if slices.Contains(rule.Names, segment) {
			violations = append(violations, violation{Rule: rule.Rule, Path: packagePath(path), Message: rule.Message})
		}
	}
	return violations
}

func unlistedInternalPackage(path string, allowed []string) bool {
	if !strings.HasPrefix(path, "internal/") {
		return false
	}
	name := packagePath(path)
	return !slices.Contains(allowed, name)
}

func pathViolations(path string, item policy) []violation {
	var violations []violation
	for _, rule := range item.ForbiddenPaths {
		for _, pattern := range expandPattern(rule.Pattern) {
			if matchPattern(pattern, path) {
				violations = append(violations, violation{Rule: rule.Rule, Path: path, Message: rule.Message})
			}
		}
	}
	return violations
}

func migrationViolations(path string, item policy) []violation {
	rule := item.MigrationFiles
	if strings.HasPrefix(path, strings.TrimSuffix(rule.AllowedRoot, "/")+"/") {
		return nil
	}
	for _, pattern := range rule.ForbiddenElsewhere {
		if matchPattern(pattern, path) {
			return []violation{{Rule: rule.Rule, Path: path, Message: rule.Message}}
		}
	}
	return nil
}

func contractViolations(path string, item policy) []violation {
	rule := item.ContractDocuments
	for _, root := range rule.ForbiddenRoots {
		if strings.HasPrefix(path, root+"/") && matchesAnyBase(rule.FilePatterns, path) {
			return []violation{{Rule: rule.Rule, Path: path, Message: rule.Message}}
		}
	}
	return nil
}

func transportViolations(path string, item policy) []violation {
	rule := item.Transport
	if !rule.SubdirectoriesMustBeProtocols || !strings.HasPrefix(path, rule.Root+"/") {
		return nil
	}

	remainder := strings.TrimPrefix(path, rule.Root+"/")
	protocol := firstSegment(remainder)
	if protocol == "" || slices.Contains(rule.AllowedProtocols, protocol) {
		return nil
	}
	return []violation{{
		Rule:    "transport-non-protocol-subdir",
		Path:    strings.Join([]string{rule.Root, protocol}, "/"),
		Message: "transport directories require accepted protocol entries",
	}}
}

func shouldIgnore(path string, entry os.DirEntry) bool {
	if entry.IsDir() && (path == ".git" || strings.HasPrefix(path, ".git/")) {
		return true
	}
	return strings.HasPrefix(path, ".git/")
}

func packagePath(path string) string {
	return path[:strings.LastIndex(path, "/")]
}

func firstSegment(path string) string {
	before, _, _ := strings.Cut(path, "/")
	return before
}

func matchesAnyBase(patterns []string, path string) bool {
	name := filepath.Base(path)
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func expandPattern(pattern string) []string {
	open := strings.Index(pattern, "{")
	close := strings.Index(pattern, "}")
	if open < 0 || close < open {
		return []string{pattern}
	}

	prefix := pattern[:open]
	suffix := pattern[close+1:]
	parts := strings.Split(pattern[open+1:close], ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		items = append(items, prefix+part+suffix)
	}
	return items
}

func splitPath(path string) []string {
	return strings.Split(path, "/")
}

func joinPath(segments []string) string {
	return strings.Join(segments, "/")
}

func containsPatternSyntax(segment string) bool {
	return strings.ContainsAny(segment, "*?[]")
}

func matchPattern(pattern string, path string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchSegments(pattern []string, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		return matchDoubleStar(pattern, path)
	}
	if len(path) == 0 {
		return false
	}
	matched, _ := filepath.Match(pattern[0], path[0])
	return matched && matchSegments(pattern[1:], path[1:])
}

func matchDoubleStar(pattern []string, path []string) bool {
	if matchSegments(pattern[1:], path) {
		return true
	}
	if len(path) == 0 {
		return false
	}
	return matchDoubleStar(pattern, path[1:])
}

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for path, content := range files {
		writeFile(t, root, path, content)
	}
	return root
}

func writeFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func findRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if exists(filepath.Join(dir, "go.mod")) && exists(filepath.Join(dir, "quality", "boundary-manifest.json")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("source root not found")
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func requireErrorContains(t *testing.T, err error, text string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q", text)
	}
	if !strings.Contains(err.Error(), text) {
		t.Fatalf("expected error containing %q, got %v", text, err)
	}
}
