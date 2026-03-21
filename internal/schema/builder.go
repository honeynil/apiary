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
	Type                 string             `yaml:"type,omitempty"`
	Format               string             `yaml:"format,omitempty"`
	Description          string             `yaml:"description,omitempty"`
	Example              interface{}        `yaml:"example,omitempty"`
	Default              interface{}        `yaml:"default,omitempty"`
	Properties           map[string]*Schema `yaml:"properties,omitempty"`
	AdditionalProperties *Schema            `yaml:"additionalProperties,omitempty"`
	Items                *Schema            `yaml:"items,omitempty"`
	Required             []string           `yaml:"required,omitempty"`
}

// Builder converts parser.TypeInfo values into JSON Schema objects and tracks
// which schemas have been placed in the components/schemas section.
type Builder struct {
	types        map[string]*parser.TypeInfo
	components   map[string]*Schema
	processing   map[string]bool // guards against recursive types
	unknownTypes []string        // types not found in the parsed set
}

// NewBuilder creates a Builder that can resolve the provided types.
func NewBuilder(types map[string]*parser.TypeInfo) *Builder {
	return &Builder{
		types:      types,
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

	schema := &Schema{
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
		schema.Properties[jsonName] = fieldSchema
		if field.Required {
			required = append(required, jsonName)
		}
	}
	if len(required) > 0 {
		schema.Required = required
	}

	// Register before returning so recursive refs can resolve.
	b.components[name] = schema
}

func (b *Builder) buildFieldSchema(field *parser.FieldInfo) *Schema {
	s := b.BuildSchema(field.Type)
	// Decorate with doc/example/default without mutating a shared primitive schema.
	if field.Doc != "" || field.Example != "" || field.Default != "" {
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
	}

	// Unknown package-qualified type (e.g. uuid.UUID) — treat as string.
	if strings.Contains(name, ".") {
		return &Schema{Type: "string"}
	}

	return nil
}
