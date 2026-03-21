package schema

import "github.com/honeynil/apiary/internal/parser"

// ResolveAll ensures every type in names — and all types they transitively
// reference — is present in the components map.
func (b *Builder) ResolveAll(names []string) {
	for _, name := range names {
		b.ensureComponent(name)
	}
}

// Dependencies returns all struct-type names that typeRef directly or
// transitively references, in dependency order. Primitive and unknown types
// are skipped. The visited map prevents infinite loops on recursive types.
func Dependencies(typeRef *parser.TypeRef, types map[string]*parser.TypeInfo, visited map[string]bool) []string {
	if typeRef == nil {
		return nil
	}
	if typeRef.IsSlice || typeRef.IsMap {
		return Dependencies(typeRef.Elem, types, visited)
	}

	name := typeRef.Name
	if primitiveSchema(name) != nil {
		return nil
	}
	if visited[name] {
		return nil
	}
	visited[name] = true

	deps := []string{name}
	typeInfo, ok := types[name]
	if !ok {
		return deps
	}
	for _, field := range typeInfo.Fields {
		deps = append(deps, Dependencies(field.Type, types, visited)...)
	}
	return deps
}
