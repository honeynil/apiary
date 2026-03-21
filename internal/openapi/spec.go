// Package openapi contains the types that model an OpenAPI 3.1 document and
// the builder that assembles them from parsed operations.
package openapi

import "github.com/honeynil/apiary/internal/schema"

// SecurityRequirement maps a scheme name to its required scopes (usually empty).
type SecurityRequirement map[string][]string

// OpenAPI is the root object of an OpenAPI 3.1 document.
type OpenAPI struct {
	OpenAPI    string              `yaml:"openapi"`
	Info       Info                `yaml:"info"`
	Paths      map[string]PathItem `yaml:"paths,omitempty"`
	Security   interface{}         `yaml:"security,omitempty"` // []SecurityRequirement
	Components *Components         `yaml:"components,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title       string `yaml:"title"`
	Version     string `yaml:"version"`
	Description string `yaml:"description,omitempty"`
}

// PathItem groups all operations for a single URL path.
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
	Patch  *Operation `yaml:"patch,omitempty"`
}

// Operation describes a single HTTP operation on a path.
type Operation struct {
	Summary     string               `yaml:"summary,omitempty"`
	Description string               `yaml:"description,omitempty"`
	Tags        []string             `yaml:"tags,omitempty"`
	Security    interface{}          `yaml:"security,omitempty"` // []SecurityRequirement or []
	Parameters  []Parameter          `yaml:"parameters,omitempty"`
	RequestBody *RequestBody         `yaml:"requestBody,omitempty"`
	Responses   map[string]*Response `yaml:"responses"`
}

// Parameter represents a path, query, header, or cookie parameter.
type Parameter struct {
	Name        string         `yaml:"name"`
	In          string         `yaml:"in"` // "path", "query", "header"
	Description string         `yaml:"description,omitempty"`
	Required    bool           `yaml:"required"`
	Schema      *schema.Schema `yaml:"schema"`
	Example     interface{}    `yaml:"example,omitempty"`
}

// RequestBody describes the body of a request.
type RequestBody struct {
	Description string                `yaml:"description,omitempty"`
	Required    bool                  `yaml:"required"`
	Content     map[string]*MediaType `yaml:"content"`
}

// MediaType holds the schema for a specific content type.
type MediaType struct {
	Schema *schema.Schema `yaml:"schema"`
}

// Response describes a single response variant.
type Response struct {
	Description string                `yaml:"description"`
	Content     map[string]*MediaType `yaml:"content,omitempty"`
}

// SecurityScheme describes an authentication/authorization mechanism.
type SecurityScheme struct {
	Type         string `yaml:"type"`
	Scheme       string `yaml:"scheme,omitempty"`       // for type: http
	BearerFormat string `yaml:"bearerFormat,omitempty"` // for scheme: bearer
	In           string `yaml:"in,omitempty"`           // for type: apiKey
	Name         string `yaml:"name,omitempty"`         // for type: apiKey
	Description  string `yaml:"description,omitempty"`
}

// Components holds reusable schema definitions.
type Components struct {
	Schemas         map[string]*schema.Schema  `yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `yaml:"securitySchemes,omitempty"`
}
