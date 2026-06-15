package checks

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const (
	diagnosticsPackagePath = "github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
	operationsPackagePath  = "github.com/pay-bye/agent-os/internal/transport/http/operations"
	probesPackagePath      = "github.com/pay-bye/agent-os/internal/transport/http/probes"
)

type importedType struct {
	path string
	name string
}

type typedFile struct {
	file *ast.File
	info *types.Info
}

type compatibilityPackages struct {
	compatibility typedFile
	diagnostics   typedFile
}

var endpointLocalDependencyTypes = []importedType{
	{path: operationsPackagePath, name: "Operations"},
	{path: probesPackagePath, name: "ReadinessFunc"},
}

func TestRejectsForbiddenInternalNames(t *testing.T) {
	policy := mustLoadPolicy(t)

	for _, group := range policy.ForbiddenPackages {
		for _, name := range group.Names {
			path := filepath.ToSlash(filepath.Join(group.Under, name, "file.go"))
			assertViolation(t, map[string]string{path: "package " + name}, group.Rule, packagePath(path))
		}
	}
}

func TestRejectsUnlistedInternalPackage(t *testing.T) {
	assertViolation(t, map[string]string{
		"internal/sample/file.go": "package sample",
	}, "package-without-manifest", "internal/sample")
}

func TestRejectsManifestForbiddenImports(t *testing.T) {
	root := fixture(t, map[string]string{
		"go.mod":                 "module github.com/pay-bye/agent-os\n",
		"internal/flow/file.go":  `package flow; import _ "github.com/pay-bye/agent-os/internal/model"`,
		"internal/model/file.go": "package model",
		"quality/boundary-manifest.json": `{
			"schema_version": 1,
			"root": ".",
			"allowed_top_level_roots": {
				"rule": "top-level-source-root-category-preservation",
				"names": ["go.mod", "internal", "quality"],
				"message": "top-level source roots are limited to the declared categories"
			},
			"internal_packages": ["internal/flow", "internal/model"],
			"forbidden_imports": [
				{
					"rule": "model-import-boundary",
					"from": ["internal/flow"],
					"to": ["github.com/pay-bye/agent-os/internal/model"],
					"message": "flow package must not import model package"
				}
			]
		}`,
	})
	policy, err := loadPolicy(root)
	if err != nil {
		t.Fatal(err)
	}

	violations, err := forbiddenImportViolations(root, policy.ForbiddenImports)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "internal/flow -> github.com/pay-bye/agent-os/internal/model")
}

func TestCurrentTreeKeepsForbiddenImportsOut(t *testing.T) {
	root := findRoot(t)
	policy := mustLoadPolicy(t)

	violations, err := forbiddenImportViolations(root, policy.ForbiddenImports)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("forbidden imports = %v", violations)
	}
}

func TestKernelSurfaceUsesCommandFamilyPackages(t *testing.T) {
	root := findRoot(t)
	graph, err := loadPackageGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range kernelChildren() {
		assertSourcePackage(t, graph, "internal/kernel/"+child)
		assertNoRootImport(t, graph, "internal/kernel/"+child)
	}
	if _, err := os.Stat(filepath.Join(root, "internal/kernel", "commands.go")); err != nil {
		t.Fatalf("kernel root commands.go missing: %v", err)
	}
	assertCommandsFacade(t, filepath.Join(root, "internal/kernel"))
}

func TestKernelChildrenDoNotOwnStorageAccessPorts(t *testing.T) {
	violations, err := kernelChildStoragePortViolations(findRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("kernel child storage access ports = %v", violations)
	}
}

func TestPostgresDoesNotAdaptInstructionAccessPorts(t *testing.T) {
	violations, err := postgresInstructionAccessViolations(findRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("postgres instruction access adapters = %v", violations)
	}
}

func TestPostgresInstructionApplicationDoesNotOwnInstructionSemantics(t *testing.T) {
	violations, err := postgresInstructionSemanticViolations(findRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("postgres instruction semantics = %v", violations)
	}
}

func TestStoragePostgresUsesAdapterChildPackages(t *testing.T) {
	root := findRoot(t)
	graph, err := loadPackageGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range storagePostgresChildren() {
		assertSourcePackage(t, graph, "internal/storage/postgres/"+child)
	}
	assertSourcePackage(t, graph, "internal/readmodel/postgres")
	assertStoragePostgresRootResidue(t, root)
	assertMigrationFilenames(t, root)
}

func TestRejectsCompatibilityRegisterEndpointLocalDependencies(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			type Settings struct{}
		`,
		"internal/transport/http/compatibility/route.go": `
				package compatibility
				import (
					nethttp "net/http"
					"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
					"github.com/pay-bye/agent-os/internal/transport/http/operations"
					"github.com/pay-bye/agent-os/internal/transport/http/probes"
				)
				func Register(mux *nethttp.ServeMux, settings diagnostics.Settings, operations operations.Operations, probe probes.ReadinessFunc) {}
			`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Register.operations operations.Operations")
	assertContains(t, violations, "Register.probe probes.ReadinessFunc")
}

func TestRejectsCompatibilityEndpointLocalDependencies(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			type Settings struct{}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
				"github.com/pay-bye/agent-os/internal/transport/http/operations"
				"github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			func endpoint(settings diagnostics.Settings, operations operations.Operations, probe probes.ReadinessFunc) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "endpoint.operations operations.Operations")
	assertContains(t, violations, "endpoint.probe probes.ReadinessFunc")
}

func TestRejectsCompatibilityAliasedEndpointLocalDependencyParameters(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			type Settings struct{}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
				ops "github.com/pay-bye/agent-os/internal/transport/http/operations"
				ready "github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			func Register(mux *nethttp.ServeMux, settings diagnostics.Settings, operations ops.Operations, probe ready.ReadinessFunc) {}
			func endpoint(settings diagnostics.Settings, operations ops.Operations, probe ready.ReadinessFunc) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Register.operations ops.Operations")
	assertContains(t, violations, "Register.probe ready.ReadinessFunc")
	assertContains(t, violations, "endpoint.operations ops.Operations")
	assertContains(t, violations, "endpoint.probe ready.ReadinessFunc")
}

func TestRejectsCompatibilityAliasedEndpointLocalDependencyParameterTypes(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			type Settings struct{}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
				ops "github.com/pay-bye/agent-os/internal/transport/http/operations"
				ready "github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			type Ops = ops.Operations
			type Ready = ready.ReadinessFunc
			func Register(mux *nethttp.ServeMux, settings diagnostics.Settings, operations Ops, probe Ready) {}
			func endpoint(settings diagnostics.Settings, operations Ops, probe Ready) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Register.operations Ops")
	assertContains(t, violations, "Register.probe Ready")
	assertContains(t, violations, "endpoint.operations Ops")
	assertContains(t, violations, "endpoint.probe Ready")
}

func TestRejectsCompatibilitySharedSettingsEndpointLocalDependencies(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			import (
				"github.com/pay-bye/agent-os/internal/transport/http/operations"
				"github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			type Settings struct {
				operations operations.Operations
				probe probes.ReadinessFunc
			}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
			)
			func Register(mux *nethttp.ServeMux, settings diagnostics.Settings) {}
			func endpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Settings.operations operations.Operations")
	assertContains(t, violations, "Settings.probe probes.ReadinessFunc")
}

func TestRejectsCompatibilitySharedSettingsAliasedEndpointLocalDependencyFields(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			import (
				ops "github.com/pay-bye/agent-os/internal/transport/http/operations"
				ready "github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			type Settings struct {
				operations ops.Operations
				probe ready.ReadinessFunc
			}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
			)
			func Register(mux *nethttp.ServeMux, settings diagnostics.Settings) {}
			func endpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Settings.operations ops.Operations")
	assertContains(t, violations, "Settings.probe ready.ReadinessFunc")
}

func TestRejectsCompatibilitySharedSettingsAliasedEndpointLocalDependencyFieldTypes(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			import (
				ops "github.com/pay-bye/agent-os/internal/transport/http/operations"
				ready "github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			type Ops = ops.Operations
			type Ready = ready.ReadinessFunc
			type Settings struct {
				operations Ops
				probe Ready
			}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				"github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
			)
			func Register(mux *nethttp.ServeMux, settings diagnostics.Settings) {}
			func endpoint(settings diagnostics.Settings) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Settings.operations Ops")
	assertContains(t, violations, "Settings.probe Ready")
}

func TestRejectsCompatibilitySharedSettingsAliasedImportEndpointLocalDependencies(t *testing.T) {
	root := fixture(t, withEndpointLocalPackages(map[string]string{
		"go.mod": "module github.com/pay-bye/agent-os\n",
		"internal/transport/http/diagnostics/endpoint.go": `
			package diagnostics
			import (
				"github.com/pay-bye/agent-os/internal/transport/http/operations"
				"github.com/pay-bye/agent-os/internal/transport/http/probes"
			)
			type Settings struct {
				operations operations.Operations
				probe probes.ReadinessFunc
			}
		`,
		"internal/transport/http/compatibility/route.go": `
			package compatibility
			import (
				nethttp "net/http"
				diag "github.com/pay-bye/agent-os/internal/transport/http/diagnostics"
			)
			func Register(mux *nethttp.ServeMux, settings diag.Settings) {}
			func endpoint(settings diag.Settings) nethttp.HandlerFunc {
				return nil
			}
		`,
	}))

	violations, err := compatibilityDependencyViolations(root)
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, violations, "Settings.operations operations.Operations")
	assertContains(t, violations, "Settings.probe probes.ReadinessFunc")
}

func TestCurrentTreeScopesCompatibilityDependencies(t *testing.T) {
	violations, err := compatibilityDependencyViolations(findRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) > 0 {
		t.Fatalf("endpoint-local dependency violations = %v", violations)
	}
}

func kernelChildren() []string {
	return []string{
		"claiming",
		"instructions",
		"leases",
		"pause",
		"resolution",
		"routing",
		"submission",
	}
}

func storagePostgresChildren() []string {
	return []string{
		"catalog",
		"channel",
		"journal",
		"kernel",
		"metrics",
		"migrations",
		"registry",
	}
}

func withEndpointLocalPackages(files map[string]string) map[string]string {
	files["internal/transport/http/operations/operations.go"] = `
		package operations
		type Operations interface{}
	`
	files["internal/transport/http/probes/probe_response.go"] = `
		package probes
		type ReadinessFunc func()
	`
	return files
}

func assertSourcePackage(t *testing.T, graph packageGraph, path string) {
	t.Helper()

	if _, ok := graph.sourcePackage(path); !ok {
		t.Fatalf("source package %q missing", path)
	}
}

func assertStoragePostgresRootResidue(t *testing.T, root string) {
	t.Helper()

	files, err := rootGoFiles(filepath.Join(root, "internal/storage/postgres"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"database.go",
		"decode.go",
		"records.go",
		"search_path.go",
	}
	if !slices.Equal(files, want) {
		t.Fatalf("storage postgres root files = %v, want %v", files, want)
	}
}

func rootGoFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		files = append(files, entry.Name())
	}
	slices.Sort(files)
	return files, nil
}

func assertMigrationFilenames(t *testing.T, root string) {
	t.Helper()

	files, err := migrationFiles(filepath.Join(root, "internal/storage/postgres/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"001_registry_vocabulary.sql",
		"002_node_channel_journal.sql",
		"003_kernel_routing.sql",
		"004_journal_events.sql",
		"005_lease_token_digest.sql",
		"006_journal_coordinates.sql",
		"007_instruction_records.sql",
		"008_instruction_event_refs.sql",
	}
	if !slices.Equal(files, want) {
		t.Fatalf("migration files = %v, want %v", files, want)
	}
}

func migrationFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	slices.Sort(files)
	return files, nil
}

func assertNoRootImport(t *testing.T, graph packageGraph, path string) {
	t.Helper()

	item, ok := graph.sourcePackage(path)
	if !ok {
		return
	}
	if item.Imports[graph.modulePath+"/internal/kernel"] != nil {
		t.Fatalf("%s imports kernel root", path)
	}
}

func assertCommandsFacade(t *testing.T, path string) {
	t.Helper()

	methods, err := commandsMethods(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Ack",
		"Claim",
		"DropInstruction",
		"Extend",
		"ForceReleaseLeaseInstruction",
		"Heartbeat",
		"MoveAvailableInstruction",
		"MoveEntriesInstruction",
		"MoveItemInstruction",
		"Nack",
		"Pause",
		"PauseInstruction",
		"ReleaseExpiredLeaseInstruction",
		"RouteOutstandingInstruction",
		"Submit",
	}
	slices.Sort(methods)
	if !slices.Equal(methods, want) {
		t.Fatalf("Commands methods = %v, want %v", methods, want)
	}
}

func commandsMethods(path string) ([]string, error) {
	var methods []string
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".go" || file.Name() == "commands.go" {
			continue
		}
		item, err := parseGoFile(filepath.Join(path, file.Name()))
		if err != nil {
			return nil, err
		}
		methods = append(methods, commandsMethodsInFile(item)...)
	}
	item, err := parseGoFile(filepath.Join(path, "commands.go"))
	if err != nil {
		return nil, err
	}
	methods = append(methods, commandsMethodsInFile(item)...)
	return methods, nil
}

func commandsMethodsInFile(file *ast.File) []string {
	var methods []string
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || !function.Name.IsExported() {
			continue
		}
		if receiverName(function.Recv.List[0].Type) == "Commands" {
			methods = append(methods, function.Name.Name)
		}
	}
	return methods
}

func receiverName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return receiverName(value.X)
	default:
		return ""
	}
}

func kernelChildStoragePortViolations(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(filepath.Join(root, "internal/kernel"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || shouldSkipKernelChildFile(root, path, entry) {
			return err
		}
		file, err := parseGoFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		violations = append(violations, storagePortDeclarations(filepath.ToSlash(relative), file)...)
		return nil
	})
	return violations, err
}

func shouldSkipKernelChildFile(root string, path string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	relative, err := filepath.Rel(filepath.Join(root, "internal/kernel"), path)
	if err != nil {
		return true
	}
	return !isGoSource(path) || !strings.Contains(filepath.ToSlash(relative), "/")
}

func storagePortDeclarations(path string, file *ast.File) []string {
	var violations []string
	for _, declaration := range file.Decls {
		for _, typeSpec := range typeDeclarations(declaration) {
			if isStoragePortType(typeSpec) {
				violations = append(violations, path+":"+typeSpec.Name.Name)
			}
		}
	}
	return violations
}

func isStoragePortType(typeSpec *ast.TypeSpec) bool {
	if _, ok := typeSpec.Type.(*ast.InterfaceType); !ok {
		return false
	}
	return strings.HasSuffix(typeSpec.Name.Name, "Access")
}

func postgresInstructionAccessViolations(root string) ([]string, error) {
	var violations []string
	err := filepath.WalkDir(filepath.Join(root, "internal/storage/postgres"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !isGoSource(path) {
			return err
		}
		file, err := parseGoFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		violations = append(violations, instructionAccessDeclarations(filepath.ToSlash(relative), file)...)
		return nil
	})
	return violations, err
}

func instructionAccessDeclarations(path string, file *ast.File) []string {
	var violations []string
	for _, declaration := range file.Decls {
		for _, typeSpec := range typeDeclarations(declaration) {
			if typeSpec.Name.Name == "instructionAccess" {
				violations = append(violations, path+":"+typeSpec.Name.Name)
			}
		}
	}
	return violations
}

func postgresInstructionSemanticViolations(root string) ([]string, error) {
	path := filepath.Join(root, "internal/storage/postgres/kernel/instruction_application.go")
	file, err := parseGoFile(path)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	return instructionSemanticDeclarations(filepath.ToSlash(relative), file), nil
}

func instructionSemanticDeclarations(path string, file *ast.File) []string {
	var violations []string
	for _, spec := range file.Imports {
		name := strings.Trim(spec.Path.Value, `"`)
		if isInstructionSemanticImport(name) {
			violations = append(violations, path+":import "+name)
		}
	}
	for _, name := range instructionSemanticFunctionNames(file) {
		violations = append(violations, path+":"+name)
	}
	return violations
}

func isInstructionSemanticImport(name string) bool {
	return name == "github.com/pay-bye/agent-os/internal/journal/payloads" ||
		name == "github.com/pay-bye/agent-os/internal/kernel/routing"
}

func instructionSemanticFunctionNames(file *ast.File) []string {
	names := []string{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && isInstructionSemanticFunction(function.Name.Name) {
			names = append(names, function.Name.Name)
		}
	}
	return names
}

func isInstructionSemanticFunction(name string) bool {
	switch name {
	case "appliedInstruction",
		"dropAudit",
		"instructionResult",
		"leaseAudit",
		"moveAvailableAudit",
		"moveEntriesAudit",
		"moveItemAudit",
		"pauseAudit",
		"rejectedInstruction",
		"routeAudit",
		"validateOutstandingRoute":
		return true
	default:
		return false
	}
}

func isGoSource(path string) bool {
	return filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go")
}

func forbiddenImportViolations(root string, rules []importRule) ([]string, error) {
	graph, err := loadPackageGraph(root)
	if err != nil {
		return nil, err
	}
	if err := validateImportRules(rules, graph); err != nil {
		return nil, err
	}
	return graph.forbiddenImportViolations(rules), nil
}

func compatibilityDependencyViolations(root string) ([]string, error) {
	packages, err := loadCompatibilityPackages(root)
	if err != nil {
		return nil, err
	}
	var violations []string
	violations = append(violations, functionDependencyViolations(packages.compatibility, "Register")...)
	violations = append(violations, functionDependencyViolations(packages.compatibility, "endpoint")...)
	if compatibilityUsesSharedSettings(packages.compatibility) {
		violations = append(violations, structDependencyViolations(packages.diagnostics, "Settings")...)
	}
	return violations, nil
}

func loadCompatibilityPackages(root string) (compatibilityPackages, error) {
	items, err := packages.Load(typedPackageConfig(root),
		"./internal/transport/http/compatibility",
		"./internal/transport/http/diagnostics",
	)
	if err != nil {
		return compatibilityPackages{}, fmt.Errorf("load compatibility packages: %w", err)
	}
	if err := packageErrors(items); err != nil {
		return compatibilityPackages{}, err
	}
	compatibility, err := typedFileByPackageSuffix(items, "/internal/transport/http/compatibility", "route.go")
	if err != nil {
		return compatibilityPackages{}, err
	}
	diagnostics, err := typedFileByPackageSuffix(items, "/internal/transport/http/diagnostics", "endpoint.go")
	if err != nil {
		return compatibilityPackages{}, err
	}
	return compatibilityPackages{compatibility: compatibility, diagnostics: diagnostics}, nil
}

func typedPackageConfig(root string) *packages.Config {
	return &packages.Config{
		Dir:   root,
		Tests: false,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
	}
}

func packageErrors(items []*packages.Package) error {
	for _, item := range items {
		if len(item.Errors) > 0 {
			return fmt.Errorf("load package %s: %s", item.PkgPath, item.Errors[0].Msg)
		}
	}
	return nil
}

func typedFileByPackageSuffix(items []*packages.Package, suffix string, filename string) (typedFile, error) {
	for _, item := range items {
		if strings.HasSuffix(item.PkgPath, suffix) {
			return typedFileByName(item, filename)
		}
	}
	return typedFile{}, fmt.Errorf("package suffix %s missing", suffix)
}

func typedFileByName(item *packages.Package, filename string) (typedFile, error) {
	for index, path := range item.CompiledGoFiles {
		if filepath.Base(path) == filename {
			return typedFile{file: item.Syntax[index], info: item.TypesInfo}, nil
		}
	}
	return typedFile{}, fmt.Errorf("package %s file %s missing", item.PkgPath, filename)
}

func compatibilityUsesSharedSettings(source typedFile) bool {
	settings := importedType{path: diagnosticsPackagePath, name: "Settings"}
	return functionHasParameterType(source, "Register", settings) ||
		functionHasParameterType(source, "endpoint", settings)
}

func functionHasParameterType(source typedFile, name string, target importedType) bool {
	function := functionDeclaration(source.file, name)
	if function == nil || function.Type.Params == nil {
		return false
	}
	for _, field := range function.Type.Params.List {
		if typeMatches(source.info.TypeOf(field.Type), target) {
			return true
		}
	}
	return false
}

func parseGoFile(path string) (*ast.File, error) {
	return parser.ParseFile(token.NewFileSet(), path, nil, 0)
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()

	if !slices.Contains(values, want) {
		t.Fatalf("values = %v, want %q", values, want)
	}
}

func typeDeclarations(declaration ast.Decl) []*ast.TypeSpec {
	group, ok := declaration.(*ast.GenDecl)
	if !ok {
		return nil
	}
	var declarations []*ast.TypeSpec
	for _, spec := range group.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if ok {
			declarations = append(declarations, typeSpec)
		}
	}
	return declarations
}

func structDependencyViolations(source typedFile, name string) []string {
	var violations []string
	for _, field := range structFieldList(source.file, name) {
		fieldType := typeName(field.Type)
		if !isEndpointLocalDependency(source.info.TypeOf(field.Type)) {
			continue
		}
		for _, fieldName := range fieldNames(field) {
			violations = append(violations, name+"."+fieldName+" "+fieldType)
		}
	}
	return violations
}

func structFieldList(file *ast.File, name string) []*ast.Field {
	for _, declaration := range file.Decls {
		for _, typeSpec := range typeDeclarations(declaration) {
			if typeSpec.Name.Name != name {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return nil
			}
			return structType.Fields.List
		}
	}
	return nil
}

func functionDependencyViolations(source typedFile, name string) []string {
	function := functionDeclaration(source.file, name)
	if function == nil || function.Type.Params == nil {
		return nil
	}
	var violations []string
	for _, field := range function.Type.Params.List {
		fieldType := typeName(field.Type)
		if !isEndpointLocalDependency(source.info.TypeOf(field.Type)) {
			continue
		}
		for _, fieldName := range fieldNames(field) {
			violations = append(violations, name+"."+fieldName+" "+fieldType)
		}
	}
	return violations
}

func functionDeclaration(file *ast.File, name string) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	return nil
}

func fieldNames(field *ast.Field) []string {
	if len(field.Names) == 0 {
		return []string{typeName(field.Type)}
	}
	var names []string
	for _, name := range field.Names {
		names = append(names, name.Name)
	}
	return names
}

func isEndpointLocalDependency(value types.Type) bool {
	for _, target := range endpointLocalDependencyTypes {
		if typeMatches(value, target) {
			return true
		}
	}
	return false
}

func typeMatches(value types.Type, target importedType) bool {
	named := namedType(value)
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == target.path && named.Obj().Name() == target.name
}

func namedType(value types.Type) *types.Named {
	if value == nil {
		return nil
	}
	switch item := types.Unalias(value).(type) {
	case *types.Named:
		return item
	case *types.Pointer:
		return namedType(item.Elem())
	default:
		return nil
	}
}

func typeName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return typeName(value.X) + "." + value.Sel.Name
	case *ast.StarExpr:
		return "*" + typeName(value.X)
	default:
		return ""
	}
}
