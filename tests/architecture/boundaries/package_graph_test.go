package checks

import (
	"fmt"
	"sync"

	"golang.org/x/tools/go/packages"
)

var loadedGraphs sync.Map

type packageGraph struct {
	modulePath   string
	byImportPath map[string]*packages.Package
}

func loadPackageGraph(root string) (packageGraph, error) {
	if cached, ok := loadedGraphs.Load(root); ok {
		return cached.(packageGraph), nil
	}
	graph, err := newPackageGraph(root)
	if err != nil {
		return packageGraph{}, err
	}
	loadedGraphs.Store(root, graph)
	return graph, nil
}

func newPackageGraph(root string) (packageGraph, error) {
	items, err := packages.Load(packageConfig(root), "./...")
	if err != nil {
		return packageGraph{}, fmt.Errorf("boundary manifest invalid: load packages: %w", err)
	}
	graph := packageGraph{
		modulePath:   mainModulePath(items),
		byImportPath: collectPackages(items),
	}
	if graph.modulePath == "" {
		return packageGraph{}, fmt.Errorf("boundary manifest invalid: module path is missing")
	}
	if err := graph.loadError(); err != nil {
		return packageGraph{}, err
	}
	return graph, nil
}

func packageConfig(root string) *packages.Config {
	return &packages.Config{
		Dir:   root,
		Tests: false,
		Mode: packages.NeedName |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule |
			packages.NeedCompiledGoFiles,
	}
}

func mainModulePath(items []*packages.Package) string {
	for _, item := range items {
		if item.Module != nil && item.Module.Main {
			return item.Module.Path
		}
	}
	return ""
}

func collectPackages(items []*packages.Package) map[string]*packages.Package {
	byImportPath := map[string]*packages.Package{}
	for _, item := range items {
		collectPackage(byImportPath, item)
	}
	return byImportPath
}

func collectPackage(byImportPath map[string]*packages.Package, item *packages.Package) {
	if item == nil || byImportPath[item.PkgPath] != nil {
		return
	}
	byImportPath[item.PkgPath] = item
	for _, imported := range item.Imports {
		collectPackage(byImportPath, imported)
	}
}

func (g packageGraph) loadError() error {
	for _, item := range g.byImportPath {
		for _, itemError := range item.Errors {
			return fmt.Errorf("boundary manifest invalid: load package %s: %s", item.PkgPath, itemError.Msg)
		}
	}
	return nil
}

func (g packageGraph) sourcePackage(path string) (*packages.Package, bool) {
	return g.importPackage(g.modulePath + "/" + path)
}

func (g packageGraph) importPackage(path string) (*packages.Package, bool) {
	item, ok := g.byImportPath[path]
	return item, ok && len(item.CompiledGoFiles) > 0
}

func (g packageGraph) forbiddenImportViolations(rules []importRule) []string {
	var violations []string
	for _, rule := range rules {
		violations = append(violations, g.ruleViolations(rule)...)
	}
	return violations
}

func (g packageGraph) ruleViolations(rule importRule) []string {
	var violations []string
	for _, source := range rule.From {
		item, ok := g.sourcePackage(source)
		if !ok {
			continue
		}
		violations = append(violations, importViolations(source, item, rule.To)...)
	}
	return violations
}

func importViolations(source string, item *packages.Package, targets []string) []string {
	var violations []string
	for _, target := range targets {
		if item.Imports[target] != nil {
			violations = append(violations, source+" -> "+target)
		}
	}
	return violations
}
