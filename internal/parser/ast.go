// Package parser walks Go source files via go/ast, collecting struct type
// definitions and functions annotated with // apiary:operation.
package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/honeynil/apiary/internal/annotation"
)

// OperationInfo holds everything extracted from a single annotated function.
type OperationInfo struct {
	Annotation   *annotation.Operation
	RequestType  *TypeRef // nil if no request body
	ResponseType *TypeRef // nil if no response body schema
}

// Parser accumulates type definitions and operations from Go source files.
type Parser struct {
	fset        *token.FileSet
	types       map[string]*TypeInfo
	operations  []*OperationInfo
	enums       map[string]*EnumInfo
	typeAliases map[string]string // "Status" → "string"
}

// New creates a ready-to-use Parser.
func New() *Parser {
	return &Parser{
		fset:        token.NewFileSet(),
		types:       make(map[string]*TypeInfo),
		enums:       make(map[string]*EnumInfo),
		typeAliases: make(map[string]string),
	}
}

// ParseDir reads every non-test .go file in dir and extracts types and
// operations. Subdirectories are not descended into; use the caller to handle
// recursive patterns.
func (p *Parser) ParseDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if err := p.parseFile(filepath.Join(dir, e.Name())); err != nil {
			// Best-effort: skip files that cannot be parsed (e.g. build-tag only)
			continue
		}
	}
	return nil
}

// Operations returns all operations found so far.
func (p *Parser) Operations() []*OperationInfo {
	return p.operations
}

// Types returns all struct types found so far.
func (p *Parser) Types() map[string]*TypeInfo {
	return p.types
}

// Enums returns all enum types (named types with const values) found so far.
func (p *Parser) Enums() map[string]*EnumInfo {
	return p.enums
}

// parseFile parses a single Go source file.
func (p *Parser) parseFile(filename string) error {
	file, err := goparser.ParseFile(p.fset, filename, nil, goparser.ParseComments)
	if err != nil {
		return err
	}

	// First pass: collect struct types and named type aliases.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					p.types[typeSpec.Name.Name] = parseStructType(typeSpec.Name.Name, structType)
				} else if ident, ok := typeSpec.Type.(*ast.Ident); ok {
					// type Status string, type Role int, etc.
					p.typeAliases[typeSpec.Name.Name] = ident.Name
				}
			}
		}
		if genDecl.Tok == token.CONST {
			p.collectConsts(genDecl)
		}
	}

	// Second pass: find annotated functions.
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if op := p.parseFunction(funcDecl); op != nil {
			p.operations = append(p.operations, op)
		}
	}
	return nil
}

// collectConsts extracts const values grouped by their named type.
// Handles string literals, integer literals, and iota expressions.
func (p *Parser) collectConsts(decl *ast.GenDecl) {
	var currentType string
	var useIota bool
	iotaVal := 0

	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if vs.Type != nil {
			if ident, ok := vs.Type.(*ast.Ident); ok {
				currentType = ident.Name
			} else {
				currentType = ""
			}
		}
		if currentType == "" {
			continue
		}
		baseType, ok := p.typeAliases[currentType]
		if !ok {
			iotaVal++
			continue
		}

		// Determine if this line uses iota or has explicit values.
		hasValues := len(vs.Values) > 0
		if hasValues && isIotaExpr(vs.Values[0]) {
			useIota = true
		}

		for i, nameIdent := range vs.Names {
			if nameIdent.Name == "_" {
				iotaVal++
				continue
			}

			var val any
			if useIota || !hasValues {
				// iota-based or inherited iota line (no values = previous iota continues)
				if baseType == "string" {
					// string iota doesn't make sense, skip
					iotaVal++
					continue
				}
				val = iotaVal
			} else if i < len(vs.Values) {
				val = constLiteral(vs.Values[i], baseType)
			}

			if val == nil {
				iotaVal++
				continue
			}

			info := p.enums[currentType]
			if info == nil {
				info = &EnumInfo{BaseType: baseType}
				p.enums[currentType] = info
			}
			info.Values = append(info.Values, val)
			iotaVal++
		}
	}
}

// isIotaExpr returns true if the expression is `iota` or contains iota
// (e.g. `iota + 1`).
func isIotaExpr(expr ast.Expr) bool {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name == "iota"
	case *ast.BinaryExpr:
		return isIotaExpr(v.X) || isIotaExpr(v.Y)
	case *ast.ParenExpr:
		return isIotaExpr(v.X)
	}
	return false
}

// constLiteral extracts the Go literal value, returning string for string
// base types and int for integer base types.
func constLiteral(expr ast.Expr, baseType string) any {
	switch v := expr.(type) {
	case *ast.BasicLit:
		s := v.Value
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		// Integer literal
		if baseType == "string" {
			return s
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return s
		}
		return n
	}
	return nil
}

// parseFunction tries to extract an OperationInfo from a function declaration.
// Returns nil if the function is not annotated or has the wrong signature.
func (p *Parser) parseFunction(fn *ast.FuncDecl) *OperationInfo {
	if fn.Doc == nil {
		return nil
	}

	// Strip "//" prefix from each comment line.
	var lines []string
	for _, c := range fn.Doc.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimSpace(text)
		lines = append(lines, text)
	}

	op, ok := annotation.Parse(lines)
	if !ok {
		return nil
	}

	// Framework handlers carry no type info in signature — types come
	// from annotation request:/response: fields.
	if isGinHandler(fn) || isStdlibHTTPHandler(fn) {
		return &OperationInfo{
			Annotation:   op,
			RequestType:  parseAnnotationTypeRef(op.Request),
			ResponseType: parseAnnotationTypeRef(op.Response),
		}
	}

	// Supported signatures (results must always be (R, error)):
	//   (ctx context.Context, req T) (R, error)  — standard
	//   (req T) (R, error)                        — no ctx
	//   (ctx context.Context) (R, error)          — no request body
	//   () (R, error)                             — no ctx, no body (rare)
	if fn.Type == nil || fn.Type.Results == nil {
		return nil
	}
	results := fn.Type.Results.List
	if len(results) != 2 {
		return nil
	}
	if !isErrorType(results[len(results)-1].Type) {
		return nil
	}

	respRef := parseTypeExpr(results[0].Type)
	if respRef == nil {
		return nil
	}

	var reqRef *TypeRef
	if fn.Type.Params != nil {
		params := fn.Type.Params.List
		switch len(params) {
		case 0:
			// () (R, error) — no request
		case 1:
			if !isContextType(params[0].Type) {
				// (req T) (R, error) — no ctx, has request
				reqRef = parseTypeExpr(params[0].Type)
				if reqRef == nil {
					return nil
				}
			}
			// else: (ctx) (R, error) — ctx only, no request
		case 2:
			if !isContextType(params[0].Type) {
				return nil // first param must be context when there are 2 params
			}
			reqRef = parseTypeExpr(params[1].Type)
			if reqRef == nil {
				return nil
			}
		default:
			return nil // more than 2 params — not supported
		}
	}

	// Annotation request:/response: fields override inferred types.
	if ann := parseAnnotationTypeRef(op.Request); ann != nil {
		reqRef = ann
	}
	if ann := parseAnnotationTypeRef(op.Response); ann != nil {
		respRef = ann
	}

	return &OperationInfo{
		Annotation:   op,
		RequestType:  reqRef,
		ResponseType: respRef,
	}
}

// isGinHandler returns true when fn has the gin handler signature:
// func(...) with a single *gin.Context parameter and no return values.
func isGinHandler(fn *ast.FuncDecl) bool {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	param := fn.Type.Params.List[0]
	expr := param.Type
	// Accept *gin.Context
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "gin" && sel.Sel.Name == "Context"
}

// isStdlibHTTPHandler returns true when fn matches:
//
//	func(w http.ResponseWriter, r *http.Request)
//
// Both method receivers and free functions are accepted.
func isStdlibHTTPHandler(fn *ast.FuncDecl) bool {
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	if fn.Type.Params == nil {
		return false
	}
	var types []ast.Expr
	for _, p := range fn.Type.Params.List {
		n := len(p.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			types = append(types, p.Type)
		}
	}
	if len(types) != 2 {
		return false
	}
	return isHTTPResponseWriter(types[0]) && isHTTPRequestPtr(types[1])
}

// isHTTPResponseWriter matches `http.ResponseWriter` (no pointer).
func isHTTPResponseWriter(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "http" && sel.Sel.Name == "ResponseWriter"
}

// isHTTPRequestPtr matches `*http.Request`.
func isHTTPRequestPtr(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "http" && sel.Sel.Name == "Request"
}

// parseAnnotationTypeRef converts an annotation type string to a TypeRef.
// Handles "TypeName", "[]TypeName", and "*TypeName".
// Returns nil for empty input.
func parseAnnotationTypeRef(s string) *TypeRef {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "[]") {
		elem := strings.TrimPrefix(s, "[]")
		return &TypeRef{Name: "array", IsSlice: true, Elem: &TypeRef{Name: strings.TrimSpace(elem)}}
	}
	if strings.HasPrefix(s, "*") {
		return &TypeRef{Name: strings.TrimPrefix(s, "*"), IsPtr: true}
	}
	return &TypeRef{Name: s}
}

// isContextType returns true when expr is context.Context.
func isContextType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "context" && sel.Sel.Name == "Context"
}

// isErrorType returns true when expr is the built-in error interface.
func isErrorType(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "error"
}
