#!/usr/bin/env bash

verify_protected_paths() {
  run_go_program <<'GO'
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type manifest struct {
	SchemaVersion int          `json:"schema_version"`
	PathRoot      string       `json:"path_root"`
	Protections   []protection `json:"protections"`
}

type protection struct {
	Mode     string   `json:"mode"`
	Patterns []string `json:"patterns"`
	Reason   string   `json:"reason"`
}

type change struct {
	status string
	path   string
}

func main() {
	item, err := readManifest()
	if err != nil {
		fail(err)
	}
	changes, err := readChanges()
	if err != nil {
		fail(err)
	}
	for _, item := range item.Protections {
		for _, change := range changes {
			if protected(item, change.path) {
				if err := enforce(item, change); err != nil {
					fail(err)
				}
			}
		}
	}
}

func readManifest() (manifest, error) {
	file, err := os.Open("quality/protected-paths.json")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return manifest{}, errors.New("protected paths missing: quality/protected-paths.json")
		}
		return manifest{}, err
	}
	defer file.Close()

	var item manifest
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return manifest{}, fmt.Errorf("protected paths invalid: %w", err)
	}
	if err := validate(item); err != nil {
		return manifest{}, err
	}
	return item, nil
}

func validate(item manifest) error {
	if item.SchemaVersion != 1 {
		return fmt.Errorf("protected paths invalid: unsupported schema_version %d", item.SchemaVersion)
	}
	if item.PathRoot != "source_root" {
		return fmt.Errorf("protected paths invalid: path_root must be source_root")
	}
	for _, item := range item.Protections {
		if item.Mode != "immutable" && item.Mode != "append_only" {
			return fmt.Errorf("protected paths invalid: unsupported mode %s", item.Mode)
		}
		if len(item.Patterns) == 0 || strings.TrimSpace(item.Reason) == "" {
			return errors.New("protected paths invalid: protections require patterns and reason")
		}
		for _, pattern := range item.Patterns {
			if invalidPattern(pattern) {
				return fmt.Errorf("protected paths invalid: source-root-relative pattern %s", pattern)
			}
		}
	}
	return nil
}

func invalidPattern(pattern string) bool {
	if strings.HasPrefix(pattern, "/") {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(pattern), "/") {
		if part == "" || part == "." || part == ".." {
			return true
		}
	}
	return false
}

func readChanges() ([]change, error) {
	output, err := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(output), "\n")
	changes := make([]change, 0, len(lines))
	for _, line := range lines {
		item, ok := parseChange(line)
		if ok {
			changes = append(changes, item)
		}
	}
	return changes, nil
}

func parseChange(line string) (change, bool) {
	if strings.TrimSpace(line) == "" {
		return change{}, false
	}
	if len(line) < 4 {
		return change{status: "?", path: strings.TrimSpace(line)}, true
	}
	code := line[:2]
	path := strings.TrimSpace(line[3:])
	if strings.Contains(path, " -> ") {
		parts := strings.Split(path, " -> ")
		path = parts[len(parts)-1]
	}
	return change{status: status(code), path: filepath.ToSlash(path)}, true
}

func status(code string) string {
	if code == "??" {
		return "A"
	}
	for _, item := range []string{"U", "R", "C", "D", "M", "A", "T"} {
		if strings.Contains(code, item) {
			return item
		}
	}
	return "?"
}

func protected(item protection, path string) bool {
	for _, pattern := range item.Patterns {
		if match(pattern, path) {
			return true
		}
	}
	return false
}

func enforce(item protection, change change) error {
	switch item.Mode {
	case "immutable":
		return fmt.Errorf("protected path changed: mode=immutable path=%s", change.path)
	case "append_only":
		if change.status == "A" {
			return nil
		}
		return fmt.Errorf("protected path changed: mode=append_only path=%s status=%s", change.path, change.status)
	default:
		return fmt.Errorf("protected paths invalid: unsupported mode %s", item.Mode)
	}
}

func match(pattern string, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if len(patternParts) != len(pathParts) {
		return false
	}
	for index, part := range patternParts {
		if strings.Contains(part, "**") {
			return false
		}
		matched, err := filepath.Match(part, pathParts[index])
		if err != nil || !matched {
			return false
		}
	}
	return !slices.Contains(patternParts, "")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
GO
}

verify_coverage() {
  run_go_program <<'GO'
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const header = "package\tgate\tfloor_percent\treason"

type row struct {
	pkg   string
	gate  string
	floor float64
}

func main() {
	module, err := readModule()
	if err != nil {
		fail(err)
	}
	rows, err := readRows()
	if err != nil {
		fail(err)
	}
	packages, err := productionPackages(module)
	if err != nil {
		fail(err)
	}
	for _, pkg := range packages {
		key := pkg + "\tunit"
		row, ok := rows[key]
		if !ok {
			fail(fmt.Errorf("coverage floor missing: %s gate=unit", pkg))
		}
		actual, err := actualCoverage(pkg)
		if err != nil {
			fail(err)
		}
		if actual < row.floor {
			fail(fmt.Errorf(
				"coverage below floor: %s gate=unit actual=%.1f floor=%s",
				pkg,
				actual,
				formatFloor(row.floor),
			))
		}
	}
}

func readModule() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", errors.New("coverage baseline invalid: module declaration missing")
}

func readRows() (map[string]row, error) {
	file, err := os.Open("quality/coverage-baseline.tsv")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("coverage baseline missing: quality/coverage-baseline.tsv")
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() || scanner.Text() != header {
		return nil, errors.New("coverage baseline invalid: header mismatch")
	}

	rows := map[string]row{}
	for scanner.Scan() {
		item, err := parseRow(scanner.Text())
		if err != nil {
			return nil, err
		}
		key := item.pkg + "\t" + item.gate
		if _, exists := rows[key]; exists {
			return nil, fmt.Errorf("coverage baseline invalid: duplicate row %s gate=%s", item.pkg, item.gate)
		}
		rows[key] = item
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func parseRow(line string) (row, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 4 {
		return row{}, errors.New("coverage baseline invalid: rows require four columns")
	}
	if parts[0] == "" || parts[3] == "" {
		return row{}, errors.New("coverage baseline invalid: package and reason are required")
	}
	if parts[1] != "unit" && parts[1] != "integration" {
		return row{}, fmt.Errorf("coverage baseline invalid: unknown gate %s", parts[1])
	}
	if parts[1] == "integration" {
		return row{}, errors.New("coverage baseline invalid: integration rows are not installed")
	}
	if !regexp.MustCompile(`^(100|[0-9]{1,2})(\.[0-9]{1,2})?$`).MatchString(parts[2]) {
		return row{}, fmt.Errorf("coverage baseline invalid: floor_percent %s", parts[2])
	}
	floor, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return row{}, fmt.Errorf("coverage baseline invalid: floor_percent %s", parts[2])
	}
	return row{pkg: parts[0], gate: parts[1], floor: floor}, nil
}

func productionPackages(module string) ([]string, error) {
	roots := []string{"internal", "cmd"}
	packages := map[string]bool{}
	for _, root := range roots {
		if err := collectPackages(module, root, packages); err != nil {
			return nil, err
		}
	}
	items := make([]string, 0, len(packages))
	for item := range packages {
		items = append(items, item)
	}
	return items, nil
}

func collectPackages(module string, root string, packages map[string]bool) error {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		packages[module+"/"+filepath.ToSlash(filepath.Dir(path))] = true
		return nil
	})
}

func actualCoverage(pkg string) (float64, error) {
	profile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return 0, err
	}
	profile.Close()
	defer os.Remove(profile.Name())

	if err := run("go", "test", "-coverprofile="+profile.Name(), pkg); err != nil {
		return 0, err
	}
	output, err := exec.Command("go", "tool", "cover", "-func="+profile.Name()).Output()
	if err != nil {
		return 0, err
	}
	return parseCoverage(string(output))
}

func run(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func parseCoverage(output string) (float64, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "total:" {
			return strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
		}
	}
	return 0, errors.New("coverage baseline invalid: total coverage missing")
}

func formatFloor(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	return strings.TrimRight(text, ".")
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
GO
}

run_go_program() {
  local program
  local status
  program="$(mktemp /tmp/verify-XXXXXX.go)"
  cat >"$program"
  set +e
  go run "$program"
  status="$?"
  set -e
  rm -f "$program"
  return "$status"
}
