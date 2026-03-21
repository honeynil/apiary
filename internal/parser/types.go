package parser

import (
	"go/ast"
	"reflect"
	"strings"
)

// TypeRef is a structured representation of a Go type expression.
type TypeRef struct {
	Name    string   // base type name (e.g. "string", "UserDTO", "time.Time")
	IsPtr   bool     // *T
	IsSlice bool     // []T
	IsMap   bool     // map[K]V
	MapKey  string   // key type name for maps
	Elem    *TypeRef // element type for slices and maps
}

// FieldInfo describes a single struct field relevant to API schema generation.
type FieldInfo struct {
	Name       string
	Type       *TypeRef
	JSONName   string
	Doc        string
	Example    string
	Default    string
	Required   bool
	PathParam  string // non-empty when field has path:"name" tag
	QueryParam string // non-empty when field has query:"name" tag
	Header     string // non-empty when field has header:"name" tag
	OmitEmpty  bool
}

// TypeInfo describes a parsed struct type.
type TypeInfo struct {
	Name   string
	Fields []*FieldInfo
}

// parseTypeExpr converts an AST type expression to a TypeRef.
func parseTypeExpr(expr ast.Expr) *TypeRef {
	switch t := expr.(type) {
	case *ast.Ident:
		return &TypeRef{Name: t.Name}
	case *ast.StarExpr:
		inner := parseTypeExpr(t.X)
		if inner != nil {
			inner.IsPtr = true
		}
		return inner
	case *ast.ArrayType:
		elem := parseTypeExpr(t.Elt)
		return &TypeRef{Name: "array", IsSlice: true, Elem: elem}
	case *ast.MapType:
		key := parseTypeExpr(t.Key)
		elem := parseTypeExpr(t.Value)
		keyName := ""
		if key != nil {
			keyName = key.Name
		}
		return &TypeRef{Name: "map", IsMap: true, MapKey: keyName, Elem: elem}
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return &TypeRef{Name: pkg.Name + "." + t.Sel.Name}
		}
		return &TypeRef{Name: t.Sel.Name}
	case *ast.InterfaceType:
		return &TypeRef{Name: "interface{}"}
	}
	return &TypeRef{Name: "interface{}"}
}

type fieldTags struct {
	json      string
	doc       string
	example   string
	defaultV  string
	validate  string
	path      string
	query     string
	header    string
	omitEmpty bool
}

func parseStructTag(raw string) fieldTags {
	st := reflect.StructTag(raw)
	tags := fieldTags{}

	jsonTag := st.Get("json")
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		tags.json = parts[0]
		for _, p := range parts[1:] {
			if p == "omitempty" {
				tags.omitEmpty = true
			}
		}
	}

	tags.doc = st.Get("doc")
	tags.example = st.Get("example")
	tags.defaultV = st.Get("default")
	tags.validate = st.Get("validate")
	tags.path = st.Get("path")
	tags.query = st.Get("query")
	tags.header = st.Get("header")
	return tags
}

// goNameToJSON converts a Go field name to a JSON key using the same heuristic
// as encoding/json: lowercase the first letter. Additionally, pure-uppercase
// acronyms (ID, URL, UUID) are fully lowercased so "ID" → "id", not "iD".
func goNameToJSON(name string) string {
	if name == "" {
		return name
	}
	allUpper := true
	for _, r := range name {
		if r < 'A' || r > 'Z' {
			allUpper = false
			break
		}
	}
	if allUpper {
		return strings.ToLower(name)
	}
	return strings.ToLower(name[:1]) + name[1:]
}

// parseStructField converts an *ast.Field to zero or more FieldInfo values.
func parseStructField(field *ast.Field) []*FieldInfo {
	// Skip embedded fields (no explicit name)
	if len(field.Names) == 0 {
		return nil
	}

	typeRef := parseTypeExpr(field.Type)

	tagStr := ""
	if field.Tag != nil {
		tagStr = strings.Trim(field.Tag.Value, "`")
	}
	tags := parseStructTag(tagStr)

	var result []*FieldInfo
	for _, nameIdent := range field.Names {
		name := nameIdent.Name
		// Skip unexported fields
		if len(name) == 0 || !(name[0] >= 'A' && name[0] <= 'Z') {
			continue
		}

		jsonName := tags.json
		if jsonName == "" {
			jsonName = goNameToJSON(name)
		}
		if jsonName == "-" {
			continue
		}

		result = append(result, &FieldInfo{
			Name:       name,
			Type:       typeRef,
			JSONName:   jsonName,
			Doc:        tags.doc,
			Example:    tags.example,
			Default:    tags.defaultV,
			Required:   strings.Contains(tags.validate, "required"),
			PathParam:  tags.path,
			QueryParam: tags.query,
			Header:     tags.header,
			OmitEmpty:  tags.omitEmpty,
		})
	}
	return result
}

// parseStructType builds a TypeInfo from an AST struct type declaration.
func parseStructType(name string, st *ast.StructType) *TypeInfo {
	info := &TypeInfo{Name: name}
	if st.Fields == nil {
		return info
	}
	for _, field := range st.Fields.List {
		info.Fields = append(info.Fields, parseStructField(field)...)
	}
	return info
}
