package schema_test

import (
	"testing"

	"github.com/honeynil/apiary/internal/parser"
	"github.com/honeynil/apiary/internal/schema"
)

func TestBuildPrimitiveSchemas(t *testing.T) {
	b := schema.NewBuilder(nil, nil)

	cases := []struct {
		typeName   string
		wantType   string
		wantFormat string
	}{
		{"string", "string", ""},
		{"bool", "boolean", ""},
		{"int", "integer", "int32"},
		{"int64", "integer", "int64"},
		{"float32", "number", "float"},
		{"float64", "number", "double"},
		{"time.Time", "string", "date-time"},
	}

	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			s := b.BuildSchema(&parser.TypeRef{Name: c.typeName})
			if s.Type != c.wantType {
				t.Errorf("type: want %q got %q", c.wantType, s.Type)
			}
			if s.Format != c.wantFormat {
				t.Errorf("format: want %q got %q", c.wantFormat, s.Format)
			}
			if s.Ref != "" {
				t.Errorf("unexpected $ref %q for primitive", s.Ref)
			}
		})
	}
}

func TestBuildSliceSchema(t *testing.T) {
	b := schema.NewBuilder(nil, nil)
	ref := &parser.TypeRef{
		Name:    "array",
		IsSlice: true,
		Elem:    &parser.TypeRef{Name: "string"},
	}
	s := b.BuildSchema(ref)
	if s.Type != "array" {
		t.Errorf("expected array, got %s", s.Type)
	}
	if s.Items == nil || s.Items.Type != "string" {
		t.Errorf("expected string items, got %+v", s.Items)
	}
}

func TestBuildMapSchema(t *testing.T) {
	b := schema.NewBuilder(nil, nil)
	ref := &parser.TypeRef{
		Name:   "map",
		IsMap:  true,
		MapKey: "string",
		Elem:   &parser.TypeRef{Name: "int"},
	}
	s := b.BuildSchema(ref)
	if s.Type != "object" {
		t.Errorf("expected object, got %s", s.Type)
	}
	if s.AdditionalProperties == nil {
		t.Fatal("expected additionalProperties")
	}
	if s.AdditionalProperties.Type != "integer" {
		t.Errorf("expected integer additionalProperties, got %s", s.AdditionalProperties.Type)
	}
}

func TestBuildStructSchema(t *testing.T) {
	types := map[string]*parser.TypeInfo{
		"User": {
			Name: "User",
			Fields: []*parser.FieldInfo{
				{Name: "ID", JSONName: "id", Type: &parser.TypeRef{Name: "int64"}, Doc: "User ID"},
				{Name: "Name", JSONName: "name", Type: &parser.TypeRef{Name: "string"}, Required: true},
			},
		},
	}

	b := schema.NewBuilder(types, nil)
	s := b.BuildSchema(&parser.TypeRef{Name: "User"})

	if s.Ref != "#/components/schemas/User" {
		t.Errorf("expected $ref to User, got %q", s.Ref)
	}

	comps := b.Components()
	userSchema, ok := comps["User"]
	if !ok {
		t.Fatal("User not in components")
	}
	if userSchema.Type != "object" {
		t.Errorf("expected object, got %s", userSchema.Type)
	}
	if len(userSchema.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(userSchema.Properties))
	}
	idProp := userSchema.Properties["id"]
	if idProp == nil {
		t.Fatal("missing 'id' property")
	}
	if idProp.Description != "User ID" {
		t.Errorf("unexpected description: %q", idProp.Description)
	}
	if len(userSchema.Required) != 1 || userSchema.Required[0] != "name" {
		t.Errorf("expected [name] as required, got %v", userSchema.Required)
	}
}

func TestBuildNestedStructSchema(t *testing.T) {
	types := map[string]*parser.TypeInfo{
		"Response": {
			Name: "Response",
			Fields: []*parser.FieldInfo{
				{Name: "User", JSONName: "user", Type: &parser.TypeRef{Name: "UserDTO"}},
			},
		},
		"UserDTO": {
			Name: "UserDTO",
			Fields: []*parser.FieldInfo{
				{Name: "ID", JSONName: "id", Type: &parser.TypeRef{Name: "int64"}},
			},
		},
	}

	b := schema.NewBuilder(types, nil)
	b.BuildSchema(&parser.TypeRef{Name: "Response"})

	comps := b.Components()
	if _, ok := comps["Response"]; !ok {
		t.Error("Response missing from components")
	}
	if _, ok := comps["UserDTO"]; !ok {
		t.Error("UserDTO missing from components — nested struct not resolved")
	}
}

func TestBuildRecursiveStructNoPanic(t *testing.T) {
	// Node.Next → *Node (recursive)
	types := map[string]*parser.TypeInfo{
		"Node": {
			Name: "Node",
			Fields: []*parser.FieldInfo{
				{Name: "Val", JSONName: "val", Type: &parser.TypeRef{Name: "string"}},
				{Name: "Next", JSONName: "next", Type: &parser.TypeRef{Name: "Node", IsPtr: true}},
			},
		},
	}

	b := schema.NewBuilder(types, nil)
	// Must not panic or loop forever.
	s := b.BuildSchema(&parser.TypeRef{Name: "Node"})
	if s == nil {
		t.Fatal("expected non-nil schema")
	}
	comps := b.Components()
	if _, ok := comps["Node"]; !ok {
		t.Error("Node missing from components")
	}
}

func TestBuildEmbeddedStructSchema(t *testing.T) {
	types := map[string]*parser.TypeInfo{
		"Base": {
			Name:   "Base",
			Fields: []*parser.FieldInfo{{Name: "ID", JSONName: "id", Type: &parser.TypeRef{Name: "int64"}}},
		},
		"Extended": {
			Name:     "Extended",
			Embedded: []string{"Base"},
			Fields:   []*parser.FieldInfo{{Name: "Extra", JSONName: "extra", Type: &parser.TypeRef{Name: "string"}}},
		},
	}

	b := schema.NewBuilder(types, nil)
	b.BuildSchema(&parser.TypeRef{Name: "Extended"})

	comps := b.Components()
	ext, ok := comps["Extended"]
	if !ok {
		t.Fatal("Extended not in components")
	}
	if len(ext.AllOf) != 2 {
		t.Fatalf("expected allOf with 2 entries, got %d", len(ext.AllOf))
	}
	if ext.AllOf[0].Ref != "#/components/schemas/Base" {
		t.Errorf("expected $ref to Base, got %q", ext.AllOf[0].Ref)
	}
	if _, ok := comps["Base"]; !ok {
		t.Error("Base not registered in components")
	}
}

func TestBuildEmbeddedOnlySchema(t *testing.T) {
	// When there are only embedded fields and no own fields, allOf should not
	// include an extra empty object schema.
	types := map[string]*parser.TypeInfo{
		"A":    {Name: "A", Fields: []*parser.FieldInfo{{Name: "X", JSONName: "x", Type: &parser.TypeRef{Name: "string"}}}},
		"Wrap": {Name: "Wrap", Embedded: []string{"A"}},
	}
	b := schema.NewBuilder(types, nil)
	b.BuildSchema(&parser.TypeRef{Name: "Wrap"})

	wrap := b.Components()["Wrap"]
	if wrap == nil {
		t.Fatal("Wrap not in components")
	}
	if len(wrap.AllOf) != 1 {
		t.Errorf("expected allOf with 1 entry (no empty own-schema), got %d", len(wrap.AllOf))
	}
}

func TestExternalTypeMappings(t *testing.T) {
	b := schema.NewBuilder(nil, nil)
	cases := []struct{ name, wantType, wantFormat string }{
		{"uuid.UUID", "string", "uuid"},
		{"net.IP", "string", "ipv4"},
		{"sql.NullString", "string", ""},
		{"sql.NullInt64", "integer", "int64"},
		{"sql.NullBool", "boolean", ""},
		{"sql.NullTime", "string", "date-time"},
		{"url.URL", "string", "uri"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := b.BuildSchema(&parser.TypeRef{Name: c.name})
			if s.Type != c.wantType {
				t.Errorf("type: want %q got %q", c.wantType, s.Type)
			}
			if s.Format != c.wantFormat {
				t.Errorf("format: want %q got %q", c.wantFormat, s.Format)
			}
		})
	}
}

func TestBuildEnumField(t *testing.T) {
	types := map[string]*parser.TypeInfo{
		"Item": {
			Name: "Item",
			Fields: []*parser.FieldInfo{
				{Name: "Status", JSONName: "status", Type: &parser.TypeRef{Name: "ItemStatus"}},
			},
		},
	}
	enums := map[string]*parser.EnumInfo{
		"ItemStatus": {BaseType: "string", Values: []any{"active", "archived", "draft"}},
	}

	b := schema.NewBuilder(types, enums)
	b.BuildSchema(&parser.TypeRef{Name: "Item"})

	comps := b.Components()
	item, ok := comps["Item"]
	if !ok {
		t.Fatal("Item not in components")
	}
	statusProp := item.Properties["status"]
	if statusProp == nil {
		t.Fatal("missing 'status' property")
	}
	if statusProp.Type != "string" {
		t.Errorf("expected string type, got %q", statusProp.Type)
	}
	if len(statusProp.Enum) != 3 {
		t.Fatalf("expected 3 enum values, got %d", len(statusProp.Enum))
	}
	if statusProp.Enum[0] != "active" || statusProp.Enum[1] != "archived" || statusProp.Enum[2] != "draft" {
		t.Errorf("unexpected enum values: %v", statusProp.Enum)
	}
}

func TestEnsureErrorResponse(t *testing.T) {
	b := schema.NewBuilder(nil, nil)
	b.EnsureErrorResponse()
	comps := b.Components()
	errSchema, ok := comps["ErrorResponse"]
	if !ok {
		t.Fatal("ErrorResponse not in components")
	}
	if errSchema.Type != "object" {
		t.Errorf("expected object, got %s", errSchema.Type)
	}
	if _, ok := errSchema.Properties["error"]; !ok {
		t.Error("missing 'error' property in ErrorResponse")
	}
}
