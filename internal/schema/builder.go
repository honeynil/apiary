// Package schema converts Go type information into JSON Schema objects
// suitable for embedding in an OpenAPI 3.1 document.
package schema

import (
	"strings"

	"github.com/honeynil/apiary/internal/parser"
)

// Schema is a subset of JSON Schema Draft 2020-12 used by OpenAPI 3.1.
type Schema struct {
	Ref                  string             `yaml:"$ref,omitempty"`
	AllOf                []*Schema          `yaml:"allOf,omitempty"`
	Type                 string             `yaml:"type,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Example              any                `yaml:"example,omitempty"`
	Default              any                `yaml:"default,omitempty"`
	Enum                 []any              `yaml:"enum,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
}

// Builder converts parser.TypeInfo values into JSON Schema objects and tracks
// which schemas have been placed in the components/schemas section.
type Builder struct {
	types        map[string]*parser.TypeInfo
	enums        map[string]*parser.EnumInfo
	components   map[string]*Schema
	processing   map[string]bool // guards against recursive types
	unknownTypes []string        // types not found in the parsed set
}

// NewBuilder creates a Builder that can resolve the provided types.
func NewBuilder(types map[string]*parser.TypeInfo, enums map[string]*parser.EnumInfo) *Builder {
	if enums == nil {
		enums = make(map[string]*parser.EnumInfo)
	}
	return &Builder{
		types:      types,
		enums:      enums,
		components: make(map[string]*Schema),
		processing: make(map[string]bool),
	}
}

// Components returns the map of schemas that will become components/schemas.
func (b *Builder) Components() map[string]*Schema {
	return b.components
}

// UnknownTypes returns the names of types that were referenced but not found
// in the parsed source. Callers can use this to warn the user that they may
// need to broaden their scan pattern (e.g. ./... instead of ./internal/handler/...).
func (b *Builder) UnknownTypes() []string {
	return b.unknownTypes
}

// BuildSchema returns a JSON Schema for the given TypeRef. Struct types are
// registered in components and returned as a $ref.
func (b *Builder) BuildSchema(ref *parser.TypeRef) *Schema {
	if ref == nil {
		return &Schema{Type: "object"}
	}
	if ref.IsSlice {
		return &Schema{Type: "array", Items: b.BuildSchema(ref.Elem)}
	}
	if ref.IsMap {
		return &Schema{Type: "object", AdditionalProperties: b.BuildSchema(ref.Elem)}
	}
	if s := primitiveSchema(ref.Name); s != nil {
		return s
	}
	// Named enum type — inline the base type schema (with enum values added in buildFieldSchema).
	if enumInfo, ok := b.enums[ref.Name]; ok {
		if s := primitiveSchema(enumInfo.BaseType); s != nil {
			return s
		}
	}
	// Struct type — register in components, return $ref.
	b.ensureComponent(ref.Name)
	return &Schema{Ref: "#/components/schemas/" + ref.Name}
}

// BuildSchemaByName is like BuildSchema but accepts a bare type name string.
func (b *Builder) BuildSchemaByName(name string) *Schema {
	if name == "" {
		return &Schema{Type: "object"}
	}
	if s := primitiveSchema(name); s != nil {
		return s
	}
	b.ensureComponent(name)
	return &Schema{Ref: "#/components/schemas/" + name}
}

// EnsureErrorResponse registers the standard error schema in components.
func (b *Builder) EnsureErrorResponse() {
	if _, ok := b.components["ErrorResponse"]; ok {
		return
	}
	b.components["ErrorResponse"] = &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"error": {Type: "string", Description: "Human-readable error message"},
		},
		Required: []string{"error"},
	}
}

// ensureComponent builds and registers the schema for typeName if it has not
// been registered yet. Recursive types are handled safely via the processing set.
func (b *Builder) ensureComponent(name string) {
	if _, exists := b.components[name]; exists {
		return
	}
	if b.processing[name] {
		// Recursive reference — the $ref will point to a schema that will be
		// completed by the outer call; no further action needed.
		return
	}

	typeInfo, exists := b.types[name]
	if !exists {
		// Unknown / external type (e.g. from another package not scanned).
		// Emit a placeholder and record the name so callers can warn the user.
		b.components[name] = &Schema{Type: "object"}
		b.unknownTypes = append(b.unknownTypes, name)
		return
	}

	b.processing[name] = true
	defer func() { delete(b.processing, name) }()

	// Handle embedded structs via allOf.
	var allOf []*Schema
	for _, embName := range typeInfo.Embedded {
		b.ensureComponent(embName)
		allOf = append(allOf, &Schema{Ref: "#/components/schemas/" + embName})
	}

	// Own fields → properties object.
	ownSchema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}
	var required []string
	for _, field := range typeInfo.Fields {
		// Path, query and header params are represented as OpenAPI parameters,
		// not as properties of the request body schema.
		if field.PathParam != "" || field.QueryParam != "" || field.Header != "" {
			continue
		}
		fieldSchema := b.buildFieldSchema(field)
		jsonName := field.JSONName
		if jsonName == "" {
			jsonName = strings.ToLower(field.Name[:1]) + field.Name[1:]
		}
		ownSchema.Properties[jsonName] = fieldSchema
		if field.Required {
			required = append(required, jsonName)
		}
	}
	if len(required) > 0 {
		ownSchema.Required = required
	}

	// Register before returning so recursive refs can resolve.
	if len(allOf) > 0 {
		// Merge: embedded refs + own properties (only if non-empty).
		if len(ownSchema.Properties) > 0 {
			allOf = append(allOf, ownSchema)
		}
		b.components[name] = &Schema{AllOf: allOf}
	} else {
		b.components[name] = ownSchema
	}
}

func (b *Builder) buildFieldSchema(field *parser.FieldInfo) *Schema {
	typeName := field.Type.Name
	if field.Type.IsPtr {
		typeName = field.Type.Name
	}

	// Check if the field's type is a named enum type.
	enumInfo := b.enums[typeName]

	s := b.BuildSchema(field.Type)

	needsCopy := field.Doc != "" || field.Example != "" || field.Default != "" || enumInfo != nil
	if needsCopy {
		cp := *s
		s = &cp
		if field.Doc != "" {
			s.Description = field.Doc
		}
		if field.Example != "" {
			s.Example = field.Example
		}
		if field.Default != "" {
			s.Default = field.Default
		}
		if enumInfo != nil {
			s.Enum = enumInfo.Values
		}
	}
	return s
}

// primitiveSchema maps Go primitive type names to their JSON Schema equivalents.
// Returns nil for non-primitive (struct) types.
func primitiveSchema(name string) *Schema {
	// Strip pointer sigil if somehow present in the name.
	name = strings.TrimPrefix(name, "*")

	switch name {
	case "string":
		return &Schema{Type: "string"}
	case "bool":
		return &Schema{Type: "boolean"}
	case "int", "int8", "int16", "int32",
		"uint", "uint8", "uint16", "uint32", "byte", "rune":
		return &Schema{Type: "integer", Format: "int32"}
	case "int64", "uint64":
		return &Schema{Type: "integer", Format: "int64"}
	case "float32":
		return &Schema{Type: "number", Format: "float"}
	case "float64":
		return &Schema{Type: "number", Format: "double"}
	case "interface{}", "any":
		return &Schema{}
	case "time.Time":
		return &Schema{Type: "string", Format: "date-time"}
	case "time.Duration":
		return &Schema{Type: "integer", Format: "int64"}
	case "uuid.UUID", "uuid.NullUUID":
		return &Schema{Type: "string", Format: "uuid"}
	case "net.IP":
		return &Schema{Type: "string", Format: "ipv4"}
	case "url.URL":
		return &Schema{Type: "string", Format: "uri"}
	case "json.RawMessage":
		return &Schema{} // any
	case "sql.NullString":
		return &Schema{Type: "string"}
	case "sql.NullInt32":
		return &Schema{Type: "integer", Format: "int32"}
	case "sql.NullInt64":
		return &Schema{Type: "integer", Format: "int64"}
	case "sql.NullFloat64":
		return &Schema{Type: "number", Format: "double"}
	case "sql.NullBool":
		return &Schema{Type: "boolean"}
	case "sql.NullTime":
		return &Schema{Type: "string", Format: "date-time"}
	}

	// Unknown package-qualified type — treat as string (best-effort fallback).
	if strings.Contains(name, ".") {
		return &Schema{Type: "string"}
	}

	return nil
}
