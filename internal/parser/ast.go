// Package parser walks Go source files via go/ast, collecting struct type
// definitions and functions annotated with // apiary:operation.
package parser

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
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
	fset       *token.FileSet
	types      map[string]*TypeInfo
	operations []*OperationInfo
}

// New creates a ready-to-use Parser.
func New() *Parser {
	return &Parser{
		fset:  token.NewFileSet(),
		types: make(map[string]*TypeInfo),
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

// parseFile parses a single Go source file.
func (p *Parser) parseFile(filename string) error {
	file, err := goparser.ParseFile(p.fset, filename, nil, goparser.ParseComments)
	if err != nil {
		return err
	}

	// First pass: collect all struct type declarations.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			p.types[typeSpec.Name.Name] = parseStructType(typeSpec.Name.Name, structType)
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

	// Gin-style handler: func(c *gin.Context) — no return types.
	// Types must come from annotation request:/response: fields.
	if isGinHandler(fn) {
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
