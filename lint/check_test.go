package enumlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testdataDir is the testdata fixture root relative to the module root.
// Testdata fixtures import the real enum package ("github.com/donnol/enum"),
// so tests chdir to the module root and use modulePath "github.com/donnol/enum" for the
// package-qualification check to resolve correctly.
const testdataDir = "lint/testdata"

// runChecker runs the lint checker against testdata subdirs (relative to
// the module root) and returns the formatted violation lines.
func runChecker(t *testing.T, subdirs ...string) []string {
	t.Helper()
	dirs := make([]string, len(subdirs))
	for i, s := range subdirs {
		dirs[i] = testdataDir + "/" + s
	}
	return runDirs(t, dirs)
}

// runDirs runs the checker against explicit directory paths (relative to
// the module root).
func runDirs(t *testing.T, dirs []string) []string {
	t.Helper()
	chdirToModuleRoot(t)
	c := newCheckerWithModulePath(dirs, "github.com/donnol/enum")
	if err := c.scan(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return c.report()
}

// chdirToModuleRoot changes the process CWD to the directory containing
// go.mod, walking up from the package dir. Restores on test completion.
func chdirToModuleRoot(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above test package")
		}
		dir = parent
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
}

// ─── Behavior tests (testdata fixtures) ────────────────────────────────

func TestCleanEnumUsage(t *testing.T) {
	violations := runChecker(t, "good")
	if len(violations) != 0 {
		t.Errorf("expected 0 violations, got %d:", len(violations))
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

func TestDetectsEnumWrites(t *testing.T) {
	violations := runChecker(t, "bad")
	if len(violations) == 0 {
		t.Fatal("expected violations, got none")
	}
	got := strings.Join(violations, "\n")

	if !strings.Contains(got, "Statuses.Pending") {
		t.Errorf("missing field-write violation for Statuses.Pending\n%s", got)
	}
	if !strings.Contains(got, "variable") {
		t.Errorf("missing variable-reassign violation\n%s", got)
	}
	for _, v := range violations {
		if strings.Contains(v, "legitimateShadow") {
			t.Errorf("false positive on legitimate shadow: %s", v)
		}
	}
}

func TestCrossPackageWrite(t *testing.T) {
	// The import statement in xpkg.go is:
	//   import "github.com/donnol/enum/lint/testdata/enumdef"
	// With modulePath "github.com/donnol/enum", enumdef's importPath is computed as
	// "." + "/" + testdataDir + "/enumdef", which matches the import
	// path used by xpkg.go.
	violations := runDirs(t, []string{
		testdataDir + "/enumdef",
		testdataDir + "/xpkg",
	})

	if len(violations) == 0 {
		t.Fatal("expected cross-package violation, got none")
	}
	got := strings.Join(violations, "\n")
	if !strings.Contains(got, "enumdef.Statuses.Pending") {
		t.Errorf("missing cross-package field-write violation\n%s", got)
	}
}

func TestIncDecStmtFlagged(t *testing.T) {
	violations := runChecker(t, "edge")
	got := strings.Join(violations, "\n")
	if !strings.Contains(got, "Modes.Count") {
		t.Errorf("IncDecStmt (Modes.Count++) not flagged\n%s", got)
	}
}

func TestCompoundAssignmentFlagged(t *testing.T) {
	violations := runChecker(t, "edge")
	got := strings.Join(violations, "\n")
	// Modes.Count appears in both ++ and += — both should be flagged.
	count := strings.Count(got, "Modes.Count")
	if count < 2 {
		t.Errorf("expected ≥2 Modes.Count violations (++ and +=), got %d\n%s", count, got)
	}
}

func TestMultipleLHSBothFlagged(t *testing.T) {
	violations := runChecker(t, "edge")
	got := strings.Join(violations, "\n")
	if !strings.Contains(got, "Modes.Read") {
		t.Errorf("Modes.Read not flagged in multi-LHS assignment\n%s", got)
	}
	if !strings.Contains(got, "Modes.Write") {
		t.Errorf("Modes.Write not flagged in multi-LHS assignment\n%s", got)
	}
}

func TestReadAccessNotFlagged(t *testing.T) {
	violations := runChecker(t, "edge")
	for _, v := range violations {
		// legitimateReads and nonEnumReassign should not appear.
		if strings.Contains(v, "legitimateReads") || strings.Contains(v, "nonEnumReassign") {
			t.Errorf("false positive: %s", v)
		}
	}
}

func TestLowercaseFieldNotTracked(t *testing.T) {
	// `lower` is a lowercase field in the enum struct — exportedFields
	// should have skipped it. A write to Modes.lower would not appear
	// in violations because the field is not tracked.
	violations := runChecker(t, "edge")
	got := strings.Join(violations, "\n")
	if strings.Contains(got, "Modes.lower") {
		t.Errorf("lowercase field 'lower' should not be tracked\n%s", got)
	}
}

// ─── Helper function unit tests ────────────────────────────────────────

func TestExportedFieldsOnlyPascalCase(t *testing.T) {
	src := `package p
type S struct {
	Exported int
	private  int
	AlsoExp  string
	_lower   int
}
`
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var st *ast.StructType
	ast.Inspect(f, func(n ast.Node) bool {
		if t, ok := n.(*ast.StructType); ok {
			st = t
			return false
		}
		return true
	})
	if st == nil {
		t.Fatal("struct type not found")
	}
	got := exportedFields(st)
	want := map[string]bool{"Exported": true, "AlsoExp": true}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, name := range got {
		if !want[name] {
			t.Errorf("unexpected field %q (lowercase fields should be skipped)", name)
		}
	}
}

func TestIsInitForIdent(t *testing.T) {
	src := `package p
import "github.com/donnol/enum"
var X = enum.InitFor[T, struct{}]()
`
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var foundSel, foundIdent bool
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if isInitForIdent(x) {
				foundSel = true
			}
		case *ast.Ident:
			if x.Name == "InitFor" && isInitForIdent(x) {
				foundIdent = true
			}
		}
		return true
	})
	if !foundSel {
		t.Error("isInitForIdent did not match enum.InitFor selector")
	}
	_ = foundIdent // ident-form (no package qualifier) is also valid
}

func TestReadModulePath(t *testing.T) {
	dir := t.TempDir()
	modFile := "module github.com/example/mymod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(modFile), 0644); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := readModulePath()
	if got != "github.com/example/mymod" {
		t.Errorf("readModulePath = %q, want %q", got, "github.com/example/mymod")
	}
}

func TestReadModulePathMissing(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if got := readModulePath(); got != "" {
		t.Errorf("readModulePath in empty dir = %q, want empty", got)
	}
}

func TestContainsGo(t *testing.T) {
	dir := t.TempDir()
	if containsGo(dir) {
		t.Error("empty dir should not contain Go files")
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	if containsGo(dir) {
		t.Error("dir with only .md should not contain Go files")
	}
	if err := os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package p"), 0644); err != nil {
		t.Fatal(err)
	}
	if !containsGo(dir) {
		t.Error("dir with .go file should contain Go files")
	}
}

func TestResolveRootsExpandsEllipsis(t *testing.T) {
	dir := t.TempDir()
	// Put a .go file in the root so the root itself qualifies.
	if err := os.WriteFile(filepath.Join(dir, "root.go"), []byte("package p"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"a", "c"} {
		subDir := filepath.Join(dir, sub)
		os.MkdirAll(subDir, 0755)
		os.WriteFile(filepath.Join(subDir, "f.go"), []byte("package p"), 0644)
	}
	os.MkdirAll(filepath.Join(dir, "b"), 0755) // no .go files

	roots, err := resolveRoots([]string{dir + "..."})
	if err != nil {
		t.Fatalf("resolveRoots: %v", err)
	}
	got := map[string]bool{}
	for _, r := range roots {
		got[r] = true
	}
	if !got[dir] {
		t.Errorf("expected root dir %q in results: %v", dir, roots)
	}
	if !got[filepath.Join(dir, "a")] {
		t.Errorf("expected a/ in results: %v", roots)
	}
	if !got[filepath.Join(dir, "c")] {
		t.Errorf("expected c/ in results: %v", roots)
	}
	if got[filepath.Join(dir, "b")] {
		t.Errorf("b/ (no .go) should be excluded: %v", roots)
	}
}

func TestResolveRootsSingleDir(t *testing.T) {
	dir := t.TempDir()
	roots, err := resolveRoots([]string{dir})
	if err != nil {
		t.Fatalf("resolveRoots: %v", err)
	}
	if len(roots) != 1 || roots[0] != dir {
		t.Errorf("got %v, want [%s]", roots, dir)
	}
}

func TestNewCheckerWithModulePath(t *testing.T) {
	c := newCheckerWithModulePath([]string{"."}, "anymodule")
	if c.modulePath != "anymodule" {
		t.Errorf("modulePath = %q, want %q", c.modulePath, "anymodule")
	}
	if c.fset == nil {
		t.Error("fset not initialized")
	}
	if len(c.dirs) != 1 || c.dirs[0] != "." {
		t.Errorf("dirs = %v, want [.]", c.dirs)
	}
}

func TestFormatViolationsEmpty(t *testing.T) {
	c := newCheckerWithModulePath(nil, "test")
	got := c.formatViolations()
	if got != nil {
		t.Errorf("formatViolations on empty = %v, want nil", got)
	}
}

func TestFormatViolationsProducesTable(t *testing.T) {
	c := newCheckerWithModulePath(nil, "test")
	c.violations = []violation{
		{pos: token.Position{Filename: "foo.go", Line: 10}, kind: "field", target: "X.Y"},
		{pos: token.Position{Filename: "bar.go", Line: 20}, kind: "variable", target: "Z"},
	}
	got := c.formatViolations()
	if len(got) < 4 { // header + separator + 2 rows
		t.Fatalf("expected ≥4 lines, got %d: %v", len(got), got)
	}
	out := strings.Join(got, "\n")
	if !strings.Contains(out, "Location") || !strings.Contains(out, "Kind") || !strings.Contains(out, "Target") {
		t.Errorf("missing table headers\n%s", out)
	}
	if !strings.Contains(out, "foo.go:10") {
		t.Errorf("missing file:line for first violation\n%s", out)
	}
	if !strings.Contains(out, "X.Y") || !strings.Contains(out, "Z") {
		t.Errorf("missing targets in output\n%s", out)
	}
}

// ─── findEnumVars branch coverage ──────────────────────────────────────

func TestSingleTypeArgIndexExprBranch(t *testing.T) {
	// testdata/singleidx uses `enum.InitFor[Status]()` — single type arg.
	// This is parsed as *ast.IndexExpr (not IndexListExpr), exercising
	// that case in findEnumVars. Since there's no struct type arg,
	// the call is skipped (len(indices) < 2) and NotAnEnum is NOT
	// tracked as an enum. Writes to it must NOT be flagged.
	violations := runChecker(t, "singleidx")
	if len(violations) != 0 {
		t.Errorf("expected 0 violations (NotAnEnum should not be tracked), got %d:", len(violations))
		for _, v := range violations {
			t.Errorf("  %s", v)
		}
	}
}

func TestSelectorExprBranchNoCrash(t *testing.T) {
	// Exercises the *ast.SelectorExpr case in findEnumVars.
	// `obj.InitFor()` is a non-generic method call — ce.Fun is a
	// bare *ast.SelectorExpr (no type args). fun.X is an *ast.Ident
	// (obj), not IndexListExpr or IndexExpr, so the else-continue
	// sub-branch fires. Must not crash and must not track anything.
	src := `package p
type holder struct{}
func (holder) InitFor() {}
var X = holder{}.InitFor()
func write() {
	X = holder{}.InitFor()
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := newCheckerWithModulePath(nil, "test")
	c.findEnumVars(f, "test/p", "test.go")
	if len(c.enums) != 0 {
		t.Errorf("expected 0 enums tracked, got %d: %+v", len(c.enums), c.enums)
	}
}

func TestSelectorExprWithIndexedReceiver(t *testing.T) {
	// Exercises the SelectorExpr → IndexExpr sub-branch.
	// `Container[int].Method()` — ce.Fun is *ast.SelectorExpr,
	// fun.X is *ast.IndexExpr (Container[int]). Since Sel.Name is
	// "Method" (not "InitFor"), isInitForIdent returns false → skip.
	// Must not crash.
	src := `package p
type Container[T any] struct{ val T }
func (c Container[T]) Method() T { return c.val }
var X = Container[int].Method()
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := newCheckerWithModulePath(nil, "test")
	c.findEnumVars(f, "test/p", "test.go")
	if len(c.enums) != 0 {
		t.Errorf("expected 0 enums tracked, got %d", len(c.enums))
	}
}

// ─── readModulePathOrExit ──────────────────────────────────────────────

func TestReadModulePathOrExitSuccess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	got := readModulePathOrExit()
	if got != "example.com/m" {
		t.Errorf("readModulePathOrExit = %q, want %q", got, "example.com/m")
	}
}

func TestReadModulePathOrExitCallsOsExit(t *testing.T) {
	// readModulePathOrExit calls os.Exit(1) when go.mod is missing.
	// We test this in a subprocess so the test process itself survives.
	if os.Getenv("TEST_READMODULE_EXIT") == "1" {
		readModulePathOrExit()
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestReadModulePathOrExitCallsOsExit")
	cmd.Env = append(os.Environ(), "TEST_READMODULE_EXIT=1")
	cmd.Dir = dir // no go.mod here
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit, got nil error")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(stderr.String(), "go.mod") {
		t.Errorf("stderr should mention go.mod, got: %s", stderr.String())
	}
}

// ─── Regression tests for recent fixes ────────────────────────────────

func TestShadowingScopedPerFunction(t *testing.T) {
	// testdata/shadow has two functions:
	//   shadow()    — local `Statuses := "local"` then `= "x"` — NOT flagged
	//   realWrite() — Statuses.Pending = "hacked" — MUST be flagged
	// The local shadow in one function must not suppress detection in
	// the other.
	violations := runChecker(t, "shadow")
	if len(violations) == 0 {
		t.Fatal("expected violation in realWrite, got none")
	}
	got := strings.Join(violations, "\n")
	if !strings.Contains(got, "Statuses.Pending") {
		t.Errorf("missing realWrite violation\n%s", got)
	}
	for _, v := range violations {
		if strings.Contains(v, "shadow.go:12") { // shadow() local assignment
			t.Errorf("false positive on local shadow: %s", v)
		}
	}
}

func TestAbsolutePathRoots(t *testing.T) {
	// Absolute directory paths must produce the same import path as
	// relative ones. Run both and compare violation output.
	chdirToModuleRoot(t)
	absGood, _ := filepath.Abs(testdataDir + "/good")
	absBad, _ := filepath.Abs(testdataDir + "/bad")

	relViolations := runDirs(t, []string{testdataDir + "/bad"})
	_ = relViolations

	c := newCheckerWithModulePath([]string{absGood, absBad}, "github.com/donnol/enum")
	if err := c.scan(); err != nil {
		t.Fatalf("scan(abs): %v", err)
	}
	absViolations := c.report()
	if len(absViolations) == 0 {
		t.Fatal("absolute-path scan found no violations in testdata/bad")
	}
	got := strings.Join(absViolations, "\n")
	if !strings.Contains(got, "Statuses.Pending") {
		t.Errorf("absolute-path scan missed field write\n%s", got)
	}
}

func TestNonEnumInitForRejected(t *testing.T) {
	// A package named `enum` aliasing something else, or a local InitFor
	// that is NOT the enum package's InitFor, must not be tracked.
	src := `package p

import myenum "github.com/donnol/enum"
import other "example.com/other"

type S string

// Correct enum import (aliased) — should be tracked.
var Real = myenum.InitFor[S, struct {
	Field S ` + "`" + `enum:"a,A"` + "`" + `
}]()

// other.InitFor — a different package's InitFor — must NOT be tracked.
var Fake = other.InitFor[S, struct {
	Field S ` + "`" + `enum:"a,A"` + "`" + `
}]()
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	absTest, _ := filepath.Abs("test.go")
	c := newCheckerWithModulePath(nil, "github.com/donnol/enum")
	c.fset = fset // reuse the same fset so extractImports can resolve positions
	c.extractImports(f)
	c.findEnumVars(f, "github.com/donnol/enum/p", absTest)
	// Only `Real` (via myenum alias) resolves to the enum package.
	if len(c.enums) != 1 {
		t.Errorf("expected 1 tracked enum, got %d: %+v", len(c.enums), c.enums)
	}
	if len(c.enums) == 1 && c.enums[0].variable != "Real" {
		t.Errorf("expected 'Real' tracked, got %q", c.enums[0].variable)
	}
}

// ─── A1 discovery + Run API ────────────────────────────────────────────

func TestDiscoverEnumPackage(t *testing.T) {
	// fakeenum declares `func InitFor[T EB, R any]() R`; usefake imports
	// it and defines an enum via fakeenum.InitFor. Discovery (no
	// -enum-pkg flag) must match the qualified call and flag the write.
	violations := runDirs(t, []string{
		testdataDir + "/discover/fakeenum",
		testdataDir + "/discover/usefake",
	})
	if len(violations) == 0 {
		t.Fatal("expected violation via discovered enum package, got none")
	}
	got := strings.Join(violations, "\n")
	if !strings.Contains(got, "Statuses.Pending") {
		t.Errorf("missing discovered-package field write\n%s", got)
	}
}

func TestDeclaresInitFor(t *testing.T) {
	with := `package p
type EB interface{ ~int | ~string }
func InitFor[T EB, R any]() R { var z R; return z }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "yes.go", with, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !declaresInitFor(f) {
		t.Error("declaresInitFor should be true for generic InitFor definition")
	}

	without := `package p
func InitFor() int { return 1 }
`
	f2, err := parser.ParseFile(fset, "no.go", without, 0)
	if err != nil {
		t.Fatal(err)
	}
	if declaresInitFor(f2) {
		t.Error("declaresInitFor should be false for non-generic InitFor")
	}
}

func TestRunAPIClean(t *testing.T) {
	chdirToModuleRoot(t)
	report, n, err := Run(Config{Roots: []string{testdataDir + "/good"}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("clean run n = %d, want 0", n)
	}
	if report != "" {
		t.Errorf("clean run report = %q, want empty", report)
	}
}

func TestRunAPIDirty(t *testing.T) {
	chdirToModuleRoot(t)
	report, n, err := Run(Config{Roots: []string{testdataDir + "/bad"}})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("dirty run expected violations")
	}
	if !strings.Contains(report, "Statuses.Pending") {
		t.Errorf("dirty run report missing violation: %s", report)
	}
}

func TestRunAPIExplicitEnumPkg(t *testing.T) {
	chdirToModuleRoot(t)
	// With an explicit enum-pkg that does NOT match, nothing is tracked.
	_, n, err := Run(Config{
		Roots:   []string{testdataDir + "/bad"},
		EnumPkg: "example.com/other-enum",
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("explicit non-matching enum-pkg: n = %d, want 0", n)
	}
}
