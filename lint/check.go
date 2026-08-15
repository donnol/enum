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
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"unicode"
)

const (
	// enumInitFor is a keyword to find the enum variable definition like:
	// 	`var Permissions = enum.InitFor[Permission, struct { ... }]()`
	// Named struct types and IIFE-wrapped calls are supported too.
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
// clean), the number of violations, and a warning ("" when nothing to warn
// about). The warning fires when the enum package we match against could not
// be located during the scan — for an explicit EnumPkg that is a likely typo
// in -enum-pkg; without one it means the enum library is absent, so write
// checks are silently disabled and the user deserves a heads-up.
func Run(cfg Config) (report string, n int, warn string, err error) {
	roots, err := resolveRoots(cfg.Roots)
	if err != nil {
		return "", 0, "", err
	}
	if len(roots) == 0 {
		return "", 0, "", nil
	}
	c := newChecker(roots)
	c.enumPkg = cfg.EnumPkg
	if err := c.scan(); err != nil {
		return "", 0, "", err
	}
	if !c.enumPkgFound() {
		if cfg.EnumPkg != "" {
			warn = fmt.Sprintf("⚠️  enum package %q not found in the scanned module — 请检查 -enum-pkg 是否输入错误", cfg.EnumPkg)
		} else {
			warn = fmt.Sprintf("⚠️  enum package not found (looked for %q) — enum write checks are disabled", c.enumImportPath())
		}
	}
	violations := c.report()
	if len(violations) == 0 {
		return "", 0, warn, nil
	}
	return strings.Join(violations, "\n"), len(violations), warn, nil
}

// enumPkgFound reports whether the enum package we match against was located
// during the scan. A package counts as found either when it appears among the
// scanned files itself, or when at least one scanned file imports it (the
// case for an external enum module pulled in as a dependency). The path
// matched is the resolved one: explicit Config.EnumPkg when set, otherwise
// the discovered package or the zapp fallback.
func (c *checker) enumPkgFound() bool {
	path := c.enumImportPath()
	for _, p := range c.fileImportPath {
		if p == path {
			return true
		}
	}
	for _, imps := range c.imports {
		for _, info := range imps {
			if info.path == path {
				return true
			}
		}
	}
	return false
}

// enumIdent records a discovered enum variable (via enum.InitFor) along with
// the exported fields of its struct (inline anonymous or named).
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
	fset              *token.FileSet            // shared fallback FileSet (tests / positionFor)
	fileSets          map[string]*token.FileSet // abs file path → per-file FileSet (parallel parse)
	dirs              []string
	modulePath        string                           // e.g. "github.com/donnol/enum", read from go.mod
	enumPkg           string                           // explicit enum package path (Config.EnumPkg)
	discoveredEnumPkg string                           // package that declares InitFor, found during scan
	files             map[string]*ast.File             // abs file path → parsed AST (single-pass cache)
	fileImportPath    map[string]string                // abs file path → import path
	enums             []enumIdent                      // all enum vars found
	typeFields        map[string]map[string][]string   // import path → type name → exported fields (index)
	enumByPkgVar      map[string]map[string]enumIdent  // package name → variable → enum (index)
	enumByImportVar   map[string]map[string]enumIdent  // import path → variable → enum (index)
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
	c.indexTypeFields()
	c.indexFiles()
	c.fillDefaultAliases()
	return nil
}

// parseAll reads every Go file under the configured roots into c.files,
// recording each file's import path and package name. Files are parsed in
// parallel (bounded by GOMAXPROCS); each file gets its own token.FileSet so
// the parser never touches shared state. Positions are resolved per file
// via positionFor.
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
	if c.fileSets == nil {
		c.fileSets = make(map[string]*token.FileSet)
	}

	type fileJob struct {
		path, abs, importPath string
	}
	var jobs []fileJob
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
			abs, _ := filepath.Abs(fp)
			jobs = append(jobs, fileJob{path: fp, abs: abs, importPath: importPathFor(c.modulePath, root)})
		}
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs []error
	)
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for _, j := range jobs {
		wg.Add(1)
		go func(j fileJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, j.path, nil, parser.ParseComments)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("parse %s: %w", j.path, err))
				mu.Unlock()
				return
			}
			mu.Lock()
			c.files[j.abs] = f
			c.fileSets[j.abs] = fs
			c.fileImportPath[j.abs] = j.importPath
			c.pkgNameOf[j.importPath] = f.Name.Name
			mu.Unlock()
		}(j)
	}
	wg.Wait()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// positionFor resolves a token.Pos from a file parsed in parallel using that
// file's own FileSet, falling back to the shared c.fset.
func (c *checker) positionFor(fileAbs string, pos token.Pos) token.Position {
	if fs := c.fileSets[fileAbs]; fs != nil {
		return fs.Position(pos)
	}
	return c.fset.Position(pos)
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
		c.extractPackage(f, abs)
		c.extractImports(f, abs)
		c.findEnumVars(f, c.fileImportPath[abs], abs)
	}
}

// extractPackage records the package name of a file.
func (c *checker) extractPackage(f *ast.File, abs string) {
	if c.pkgOf == nil {
		c.pkgOf = make(map[string]string)
	}
	if abs == "" {
		abs, _ = filepath.Abs(c.fset.File(f.Pos()).Name())
	}
	c.pkgOf[abs] = f.Name.Name
}

// extractImports records imports per file. The alias for an import with
// no explicit name is inferred as the last path segment; it is corrected
// to the real package name in fillDefaultAliases once all packages are
// scanned.
func (c *checker) extractImports(f *ast.File, abs string) {
	if c.imports == nil {
		c.imports = make(map[string]map[string]importInfo)
	}
	if abs == "" {
		abs, _ = filepath.Abs(c.fset.File(f.Pos()).Name())
	}
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
// declarations whose value is a call to enum.InitFor[...](), either
// directly or wrapped in an IIFE:
//
//	var A = enum.InitFor[T, struct { ... }]()
//	var B = enum.InitFor[T, MyCols]()                     // named type
//	var C = func() MyCols { c := enum.InitFor[T, MyCols](); return c }()
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
			fields := c.enumFieldsFromValue(vs.Values[0], f, fileAbs)
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

// enumFieldsFromValue extracts the exported enum field names from a
// package-level var initializer. Supported forms:
//   - a direct call: enum.InitFor[T, struct{...}]() or InitFor[T, Named]()
//   - an IIFE:       func() R { ...; c := enum.InitFor[T, ...](); ...; return c }()
func (c *checker) enumFieldsFromValue(val ast.Expr, f *ast.File, fileAbs string) []string {
	switch v := val.(type) {
	case *ast.CallExpr:
		if fl, ok := v.Fun.(*ast.FuncLit); ok {
			return c.enumFieldsFromFuncLit(fl, f, fileAbs)
		}
		return c.enumFieldsFromInitFor(v, f, fileAbs)
	case *ast.ParenExpr:
		return c.enumFieldsFromValue(v.X, f, fileAbs)
	}
	return nil
}

// enumFieldsFromInitFor returns the exported fields of the struct type
// argument of an enum.InitFor call, or nil when the call is not one.
func (c *checker) enumFieldsFromInitFor(ce *ast.CallExpr, _ *ast.File, fileAbs string) []string {
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
			return nil
		}
	default:
		return nil
	}
	if !c.isEnumInitFor(ix, fileAbs) {
		return nil
	}
	if len(indices) < 2 {
		return nil
	}
	return c.structFields(indices[1], fileAbs)
}

// enumFieldsFromFuncLit finds the enum.InitFor call inside an IIFE body and
// returns its struct fields. The outermost matching call wins.
func (c *checker) enumFieldsFromFuncLit(fl *ast.FuncLit, f *ast.File, fileAbs string) []string {
	var found []string
	ast.Inspect(fl.Body, func(n ast.Node) bool {
		if len(found) > 0 {
			return false
		}
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if fields := c.enumFieldsFromInitFor(ce, f, fileAbs); len(fields) > 0 {
			found = fields
			return false
		}
		return true
	})
	return found
}

// structFields resolves the exported fields of a struct type expression:
// either an inline StructType or a named type declared in the same package.
func (c *checker) structFields(typ ast.Expr, fileAbs string) []string {
	switch t := typ.(type) {
	case *ast.StructType:
		return exportedFields(t)
	case *ast.Ident:
		return c.namedStructFields(t.Name, fileAbs)
	}
	return nil
}

// indexTypeFields builds a per-package map of named struct types to their
// exported fields, so namedStructFields is an O(1) lookup instead of a
// linear scan over every file.
func (c *checker) indexTypeFields() {
	if c.typeFields != nil {
		return
	}
	idx := make(map[string]map[string][]string)
	for abs, f := range c.files {
		importPath := c.fileImportPath[abs]
		m := idx[importPath]
		if m == nil {
			m = make(map[string][]string)
			idx[importPath] = m
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.TypeParams != nil {
					continue
				}
				if st, ok := ts.Type.(*ast.StructType); ok {
					m[ts.Name.Name] = exportedFields(st)
				}
			}
		}
	}
	c.typeFields = idx
}

// namedStructFields returns the exported fields of the named struct type
// declared in the same package as fileAbs, or nil when not found. Uses the
// type index built by indexTypeFields; falls back to a linear scan when the
// index is unavailable (direct-call tests).
func (c *checker) namedStructFields(name, fileAbs string) []string {
	if c.typeFields != nil {
		return c.typeFields[c.fileImportPath[fileAbs]][name]
	}
	importPath := c.fileImportPath[fileAbs]
	for abs, f := range c.files {
		if c.fileImportPath[abs] != importPath {
			continue
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || ts.Name.Name != name || ts.TypeParams != nil {
					continue
				}
				if st, ok := ts.Type.(*ast.StructType); ok {
					return exportedFields(st)
				}
			}
		}
	}
	return nil
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

// indexEnums builds lookup maps for enum variables so write checks are O(1)
// instead of a linear scan over every enum. Package-level variable names are
// unique per package, so single-value maps suffice.
func (c *checker) indexEnums() {
	if c.enumByPkgVar != nil {
		return
	}
	c.enumByPkgVar = make(map[string]map[string]enumIdent)
	c.enumByImportVar = make(map[string]map[string]enumIdent)
	for _, en := range c.enums {
		if m := c.enumByPkgVar[en.pkg]; m != nil {
			m[en.variable] = en
		} else {
			c.enumByPkgVar[en.pkg] = map[string]enumIdent{en.variable: en}
		}
		if m := c.enumByImportVar[en.importPath]; m != nil {
			m[en.variable] = en
		} else {
			c.enumByImportVar[en.importPath] = map[string]enumIdent{en.variable: en}
		}
	}
}

// report inspects every cached file for writes to discovered enum
// variables and their exported fields.
func (c *checker) report() []string {
	c.indexEnums()
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
		if en, ok := c.enumByPkgVar[filePkg][e.Name]; ok {
			c.violations = append(c.violations, violation{
				pos:    c.positionFor(fileAbs, pos),
				kind:   "variable",
				target: en.variable,
			})
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

	// Case 0: pkgAlias.Var = ... — a write to the enum variable itself
	// through a package qualifier (e.g. `role.Permissions = ...`).
	if x, ok := sel.X.(*ast.Ident); ok && !local[x.Name] {
		importedPkg := c.resolveImport(fileAbs, x.Name)
		if en, ok := c.enumByImportVar[importedPkg][field]; ok {
			c.violations = append(c.violations, violation{
				pos:    c.positionFor(fileAbs, pos),
				kind:   "variable",
				target: x.Name + "." + en.variable,
			})
			return
		}
	}

	// Case 1: X.Field — X is a simple ident in the same package.
	if x, ok := sel.X.(*ast.Ident); ok {
		if local[x.Name] {
			return
		}
		if en, ok := c.enumByPkgVar[filePkg][x.Name]; ok {
			for _, f := range en.fields {
				if f == field {
					c.violations = append(c.violations, violation{
						pos:    c.positionFor(fileAbs, pos),
						kind:   "field",
						target: en.variable + "." + field,
					})
					return
				}
			}
		}
	}

	// Case 2: pkgAlias.X.Field — the enum var is from another package.
	if pkgSel, ok := sel.X.(*ast.SelectorExpr); ok {
		if alias, ok := pkgSel.X.(*ast.Ident); ok {
			varName := pkgSel.Sel.Name
			importedPkg := c.resolveImport(fileAbs, alias.Name)
			if en, ok := c.enumByImportVar[importedPkg][varName]; ok {
				for _, f := range en.fields {
					if f == field {
						c.violations = append(c.violations, violation{
							pos:    c.positionFor(fileAbs, pos),
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
