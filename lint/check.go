// Package enumlint checks that enum values (created with enum.InitFor)
// are never written to after initialization. Enum variables and their
// exported fields must remain read-only across the entire project.
//
// Use Run to perform a lint pass programmatically, or the cmd/enumlint
// CLI for command-line use.
package enumlint

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"unicode"
)

const (
	// enumInitFor is a keyword to find the enum variable definition like:
	// 	`var Permissions = enum.InitFor[Permission, struct { ... }]()`
	enumInitFor = "InitFor"
)

// Config controls a lint run.
type Config struct {
	// Roots are the directories to scan. `./...` expands recursively.
	Roots []string
	// EnumPkg optionally pins the enum package's import path, e.g.
	// "github.com/x/enum". When empty the tool discovers the package
	// that declares InitFor, falling back to modulePath+"/server/util/enum".
	EnumPkg string
}

// Run lints the configured roots and returns the formatted report ("" if
// clean) and the number of violations.
func Run(cfg Config) (report string, n int, err error) {
	roots, err := resolveRoots(cfg.Roots)
	if err != nil {
		return "", 0, err
	}
	if len(roots) == 0 {
		return "", 0, nil
	}
	c := newChecker(roots)
	c.enumPkg = cfg.EnumPkg
	if err := c.scan(); err != nil {
		return "", 0, err
	}
	violations := c.report()
	if len(violations) == 0 {
		return "", 0, nil
	}
	return strings.Join(violations, "\n"), len(violations), nil
}

// enumIdent records a discovered enum variable (via enum.InitFor) along with
// the exported fields of its anonymous struct.
type enumIdent struct {
	variable   string   // e.g. "Events"
	pkg        string   // package name (short name)
	importPath string   // full import path within the module, e.g. "github.com/x/y/server/biz/event"
	fields     []string // exported field names of the anonymous struct
}

// violation is a write to an enum variable or field.
type violation struct {
	pos    token.Position
	kind   string // "variable" or "field"
	target string // e.g. "Events" or "Events.UserCreated" or "event.Events.UserCreated"
}

// importInfo records an import in a specific file.
type importInfo struct {
	path     string
	explicit bool // true when the file used `import alias "path"`
}

// checker holds the state for a full-module lint run.
type checker struct {
	fset              *token.FileSet
	dirs              []string
	modulePath        string                           // e.g. "github.com/donnol/enum", read from go.mod
	enumPkg           string                           // explicit enum package path (Config.EnumPkg)
	discoveredEnumPkg string                           // package that declares InitFor, found during scan
	files             map[string]*ast.File             // abs file path → parsed AST (single-pass cache)
	fileImportPath    map[string]string                // abs file path → import path
	enums             []enumIdent                      // all enum vars found
	pkgOf             map[string]string                // file (abs) → package name
	imports           map[string]map[string]importInfo // file (abs) → alias → import info
	pkgNameOf         map[string]string                // import path → real package name
	violations        []violation
}

// newChecker creates a checker with the module path auto-detected from
// go.mod in the current directory. Exits the process if go.mod is missing.
func newChecker(roots []string) *checker {
	return newCheckerWithModulePath(roots, readModulePathOrExit())
}

// newCheckerWithModulePath creates a checker with an explicit module path.
// Used by tests that need to bypass go.mod discovery.
func newCheckerWithModulePath(roots []string, modulePath string) *checker {
	return &checker{
		fset:       token.NewFileSet(),
		dirs:       roots,
		modulePath: modulePath,
	}
}

// readModulePathOrExit reads go.mod from CWD; exits the process if missing.
func readModulePathOrExit() string {
	mp := readModulePath()
	if mp == "" {
		fmt.Fprintln(os.Stderr, "lint: cannot read module path from go.mod — run from the module root")
		os.Exit(1)
	}
	return mp
}

func (c *checker) scan() error {
	if err := c.parseAll(); err != nil {
		return err
	}
	c.discoverEnumPackage()
	c.indexFiles()
	c.fillDefaultAliases()
	return nil
}

// parseAll reads every Go file under the configured roots into c.files,
// recording each file's import path and package name.
func (c *checker) parseAll() error {
	if c.files == nil {
		c.files = make(map[string]*ast.File)
	}
	if c.fileImportPath == nil {
		c.fileImportPath = make(map[string]string)
	}
	if c.pkgNameOf == nil {
		c.pkgNameOf = make(map[string]string)
	}
	for _, root := range c.dirs {
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			fp := filepath.Join(root, e.Name())
			f, err := parser.ParseFile(c.fset, fp, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse %s: %w", fp, err)
			}
			abs, _ := filepath.Abs(fp)
			c.files[abs] = f
			c.fileImportPath[abs] = importPathFor(c.modulePath, root)
			c.pkgNameOf[c.fileImportPath[abs]] = f.Name.Name
		}
	}
	return nil
}

// importPathFor builds the module import path for a scan root. Absolute
// roots are converted to module-relative first.
func importPathFor(modulePath, root string) string {
	relRoot := root
	if filepath.IsAbs(root) {
		if rel, err := filepath.Rel(".", root); err == nil {
			relRoot = rel
		}
	}
	if relRoot == "." {
		return modulePath
	}
	return modulePath + "/" + filepath.ToSlash(relRoot)
}

// discoverEnumPackage locates the package that declares the enum
// library's InitFor generic function, so qualified references can be
// matched regardless of where the enum module lives.
func (c *checker) discoverEnumPackage() {
	for abs, f := range c.files {
		if declaresInitFor(f) {
			c.discoveredEnumPkg = c.fileImportPath[abs]
			return
		}
	}
}

// declaresInitFor reports whether a file declares
// `func InitFor[T EnumBase, R any]() R`.
func declaresInitFor(f *ast.File) bool {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != enumInitFor || fd.Type.TypeParams == nil {
			continue
		}
		// Signature: no params, at least one result.
		if len(fd.Type.Params.List) == 0 && fd.Type.Results != nil {
			return true
		}
	}
	return false
}

// enumImportPath returns the enum package import path to match against.
// Order: explicit Config.EnumPkg → discovered package.
func (c *checker) enumImportPath() string {
	if c.enumPkg != "" {
		return c.enumPkg
	}
	if c.discoveredEnumPkg != "" {
		return c.discoveredEnumPkg
	}
	return c.modulePath
}

// indexFiles runs extraction and enum-var discovery over all cached files.
func (c *checker) indexFiles() {
	for abs, f := range c.files {
		c.extractPackage(f)
		c.extractImports(f)
		c.findEnumVars(f, c.fileImportPath[abs], abs)
	}
}

// extractPackage records the package name of a file.
func (c *checker) extractPackage(f *ast.File) {
	if c.pkgOf == nil {
		c.pkgOf = make(map[string]string)
	}
	abs, _ := filepath.Abs(c.fset.File(f.Pos()).Name())
	c.pkgOf[abs] = f.Name.Name
}

// extractImports records imports per file. The alias for an import with
// no explicit name is inferred as the last path segment; it is corrected
// to the real package name in fillDefaultAliases once all packages are
// scanned.
func (c *checker) extractImports(f *ast.File) {
	if c.imports == nil {
		c.imports = make(map[string]map[string]importInfo)
	}
	abs, _ := filepath.Abs(c.fset.File(f.Pos()).Name())
	m := c.imports[abs]
	if m == nil {
		m = make(map[string]importInfo)
		c.imports[abs] = m
	}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		explicit := imp.Name != nil
		name := ""
		if explicit {
			name = imp.Name.Name
		} else {
			// Placeholder; corrected in fillDefaultAliases.
			name = filepath.Base(path)
		}
		m[name] = importInfo{path: path, explicit: explicit}
	}
}

// fillDefaultAliases replaces inferred aliases (last path segment) with
// the real package name for imports inside the scanned module.
func (c *checker) fillDefaultAliases() {
	for _, m := range c.imports {
		for alias, info := range m {
			if info.explicit {
				continue
			}
			if realName, ok := c.pkgNameOf[info.path]; ok && realName != "" && realName != alias {
				delete(m, alias)
				m[realName] = info
			}
		}
	}
}

// findEnumVars walks the AST of a file looking for package-level var
// declarations whose value is a call to enum.InitFor[...]() or
// InitFor[...]().
func (c *checker) findEnumVars(f *ast.File, importDir, fileAbs string) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Values) == 0 {
				continue
			}
			// Look for enum.InitFor[T, struct{...}]() or InitFor[T, struct{...}]()
			ce, ok := vs.Values[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			var ix ast.Expr
			var indices []ast.Expr
			switch fun := ce.Fun.(type) {
			case *ast.IndexListExpr:
				ix = fun.X
				indices = fun.Indices
			case *ast.IndexExpr:
				ix = fun.X
				indices = []ast.Expr{fun.Index}
			case *ast.SelectorExpr:
				if il, ok := fun.X.(*ast.IndexListExpr); ok {
					ix = il.X
					indices = il.Indices
				} else if ie, ok := fun.X.(*ast.IndexExpr); ok {
					ix = ie.X
					indices = []ast.Expr{ie.Index}
				} else {
					continue
				}
			default:
				continue
			}
			if !c.isEnumInitFor(ix, fileAbs) {
				continue
			}
			if len(indices) < 2 {
				continue
			}
			st, ok := indices[1].(*ast.StructType)
			if !ok {
				continue
			}
			fields := exportedFields(st)
			if len(fields) == 0 {
				continue
			}
			for _, ident := range vs.Names {
				c.enums = append(c.enums, enumIdent{
					variable:   ident.Name,
					pkg:        f.Name.Name,
					importPath: importDir,
					fields:     fields,
				})
			}
		}
	}
}

// isInitForIdent checks whether an ast.Expr is a bare reference to a
// function named InitFor (either `InitFor` or `pkg.InitFor`). It does
// NOT validate which package — use isEnumInitFor for that.
func isInitForIdent(ix ast.Expr) bool {
	if ix == nil {
		return false
	}
	switch x := ix.(type) {
	case *ast.SelectorExpr:
		return x.Sel.Name == enumInitFor
	case *ast.Ident:
		return x.Name == enumInitFor
	}
	return false
}

// isEnumInitFor reports whether ix refers to the enum package's InitFor.
// A qualified reference (`enum.InitFor`) must resolve to the enum
// package import path; a bare `InitFor` matches by name only.
func (c *checker) isEnumInitFor(ix ast.Expr, fileAbs string) bool {
	sel, ok := ix.(*ast.SelectorExpr)
	if !ok {
		return isInitForIdent(ix)
	}
	if sel.Sel.Name != enumInitFor {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	path := c.resolveImport(fileAbs, pkgIdent.Name)
	return path == c.enumImportPath()
}

// exportedFields returns the names of exported (PascalCase) struct fields.
func exportedFields(st *ast.StructType) []string {
	var names []string
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		name := f.Names[0].Name
		if unicode.IsUpper([]rune(name)[0]) {
			names = append(names, name)
		}
	}
	return names
}

// ── Report (write-detection) ────────────────────────────────────────────

// report inspects every cached file for writes to discovered enum
// variables and their exported fields.
func (c *checker) report() []string {
	for abs, f := range c.files {
		c.checkFileWrites(f, abs)
	}
	return c.formatViolations()
}

// formatViolations writes violations to a tab-aligned table for CLI output.
func (c *checker) formatViolations() []string {
	if len(c.violations) == 0 {
		return nil
	}
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Location\tKind\tTarget")
	fmt.Fprintln(w, "--------\t----\t------")
	for _, v := range c.violations {
		fmt.Fprintf(w, "%s:%d\t%s\t%s\n",
			v.pos.Filename, v.pos.Line, v.kind, v.target)
	}
	w.Flush()
	return strings.Split(strings.TrimSpace(buf.String()), "\n")
}

func (c *checker) checkFileWrites(f *ast.File, abs string) {
	pkg := c.pkgOf[abs]

	// Walk each function with its own local scope so a local shadowing an
	// enum name inside one function does not suppress detection in others.
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		visitor := &writeVisitor{c: c, pkg: pkg, abs: abs, locals: make(map[string]bool)}
		declareFieldList(fd.Type.Params, visitor.locals)
		declareFieldList(fd.Type.Results, visitor.locals)
		ast.Walk(visitor, fd)
	}
}

// writeVisitor walks a function body, tracking declared names per scope
// and flagging writes to enum vars/fields. Switches to a fresh local
// scope (copying captured outer locals) when entering a nested FuncLit.
type writeVisitor struct {
	c      *checker
	pkg    string
	abs    string
	locals map[string]bool
}

func (v *writeVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncLit:
		// Closure: captures outer locals plus its own params.
		inner := make(map[string]bool, len(v.locals)+4)
		for k := range v.locals {
			inner[k] = true
		}
		declareFieldList(n.Type.Params, inner)
		return &writeVisitor{c: v.c, pkg: v.pkg, abs: v.abs, locals: inner}
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			// `:=` declares new locals, not writes to package-level enums.
			for _, lhs := range n.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					v.locals[id.Name] = true
				}
			}
			return v
		}
		for _, lhs := range n.Lhs {
			v.c.checkWriteExpr(lhs, v.pkg, v.abs, n.Pos(), v.locals)
		}
	case *ast.GenDecl:
		if n.Tok == token.VAR {
			for _, spec := range n.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						v.locals[name.Name] = true
					}
				}
			}
		}
	case *ast.RangeStmt:
		if n.Tok == token.DEFINE {
			for _, e := range []ast.Expr{n.Key, n.Value} {
				if id, ok := e.(*ast.Ident); ok {
					v.locals[id.Name] = true
				}
			}
		}
	case *ast.IncDecStmt:
		// X++ / X-- also mutates; treat like an assignment.
		v.c.checkWriteExpr(n.X, v.pkg, v.abs, n.Pos(), v.locals)
	}
	return v
}

// declareFieldList records field names (params/results) as locals.
func declareFieldList(fl *ast.FieldList, local map[string]bool) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		for _, name := range f.Names {
			local[name.Name] = true
		}
	}
}

func (c *checker) checkWriteExpr(expr ast.Expr, filePkg, fileAbs string, pos token.Pos, local map[string]bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		// Direct write to the enum variable itself: Events = ...
		if local[e.Name] {
			return
		}
		for _, en := range c.enums {
			if en.pkg == filePkg && en.variable == e.Name {
				c.violations = append(c.violations, violation{
					pos:    c.fset.Position(pos),
					kind:   "variable",
					target: en.variable,
				})
			}
		}
	case *ast.SelectorExpr:
		// Two cases:
		// 1. X.Field = ... where X is the enum var (same package)
		// 2. pkg.X.Field = ... where pkg references our package
		c.checkFieldWrite(e, filePkg, fileAbs, pos, local)
	}
}

func (c *checker) checkFieldWrite(sel *ast.SelectorExpr, filePkg, fileAbs string, pos token.Pos, local map[string]bool) {
	field := sel.Sel.Name

	// Case 1: X.Field — X is a simple ident in the same package.
	if x, ok := sel.X.(*ast.Ident); ok {
		if local[x.Name] {
			return
		}
		for _, en := range c.enums {
			if en.pkg == filePkg && en.variable == x.Name {
				for _, f := range en.fields {
					if f == field {
						c.violations = append(c.violations, violation{
							pos:    c.fset.Position(pos),
							kind:   "field",
							target: en.variable + "." + field,
						})
						return
					}
				}
			}
		}
	}

	// Case 2: pkgAlias.X.Field — the enum var is from another package.
	if pkgSel, ok := sel.X.(*ast.SelectorExpr); ok {
		if alias, ok := pkgSel.X.(*ast.Ident); ok {
			varName := pkgSel.Sel.Name
			importedPkg := c.resolveImport(fileAbs, alias.Name)
			for _, en := range c.enums {
				if en.variable == varName && en.importPath == importedPkg {
					for _, f := range en.fields {
						if f == field {
							c.violations = append(c.violations, violation{
								pos:    c.fset.Position(pos),
								kind:   "field",
								target: alias.Name + "." + varName + "." + field,
							})
							return
						}
					}
				}
			}
		}
	}
}

// resolveImport resolves a local import alias to its full import path.
func (c *checker) resolveImport(fileAbs, alias string) string {
	if m, ok := c.imports[fileAbs]; ok {
		if info, ok := m[alias]; ok {
			return info.path
		}
	}
	return ""
}

// readModulePath extracts the module path from go.mod in the current
// working directory. Returns "" if go.mod is not found.
func readModulePath() string {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("module ")) {
			return strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("module "))))
		}
	}
	return ""
}

// Ensure fs import is used for future extensions.

// resolveRoots expands glob-like `./...` patterns into concrete directories
// containing Go source files. A single `./...` scans the module root and all
// subdirectories, skipping Go's reserved testdata directories.
func resolveRoots(dirs []string) ([]string, error) {
	var roots []string
	for _, d := range dirs {
		if strings.HasSuffix(d, "...") {
			base := strings.TrimSuffix(d, "...")
			base = strings.TrimSuffix(base, string(filepath.Separator))
			err := filepath.Walk(base, func(path string, info os.FileInfo, _ error) error {
				if info.IsDir() {
					if info.Name() == "testdata" {
						return filepath.SkipDir
					}
					if containsGo(path) {
						roots = append(roots, path)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			roots = append(roots, d)
		}
	}
	return roots, nil
}

func containsGo(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}
