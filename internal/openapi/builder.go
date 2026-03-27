package openapi

import (
	"log"
	"strconv"
	"strings"

	"github.com/honeynil/apiary/internal/parser"
	"github.com/honeynil/apiary/internal/schema"
)

// builtinSchemes maps short names to their SecurityScheme definitions.
var builtinSchemes = map[string]*SecurityScheme{
	"bearer": {
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT Bearer token — pass as: Authorization: Bearer <token>",
	},
	"basic": {
		Type:        "http",
		Scheme:      "basic",
		Description: "HTTP Basic authentication",
	},
	"apikey": {
		Type:        "apiKey",
		In:          "header",
		Name:        "X-API-Key",
		Description: "API key passed in the X-API-Key header",
	},
}

type securityEntry struct {
	name    string // key used in the document (like "adminAuth")
	builtin string // builtin type: "bearer", "basic", "apikey"
}

// Builder assembles an OpenAPI 3.1 document from parsed operations.
type Builder struct {
	title       string
	version     string
	description string
	security    []securityEntry
}

func NewBuilder(title, version string) *Builder {
	return &Builder{title: title, version: version}
}

func (b *Builder) WithDescription(desc string) *Builder {
	b.description = desc
	return b
}

func (b *Builder) WithSecurity(scheme string) *Builder {
	name, builtin, found := strings.Cut(scheme, ":")
	if !found {
		builtin = name
	}
	b.security = append(b.security, securityEntry{name: name, builtin: builtin})
	return b
}

func (b *Builder) Build(operations []*parser.OperationInfo, types map[string]*parser.TypeInfo) (*OpenAPI, error) {
	sb := schema.NewBuilder(types)
	sb.EnsureErrorResponse()

	spec := &OpenAPI{
		OpenAPI: "3.1.0",
		Info:    Info{Title: b.title, Version: b.version, Description: b.description},
		Paths:   make(map[string]PathItem),
	}

	for _, opInfo := range operations {
		op, err := b.buildOperation(opInfo, sb, types)
		if err != nil {
			return nil, err
		}
		item := spec.Paths[opInfo.Annotation.Path]
		switch strings.ToUpper(opInfo.Annotation.Method) {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		case "PUT":
			item.Put = op
		case "DELETE":
			item.Delete = op
		case "PATCH":
			item.Patch = op
		}
		spec.Paths[opInfo.Annotation.Path] = item
	}

	// Build components
	components := &Components{Schemas: sb.Components()}

	// Warn about cross-package types that were not resolved.
	for _, unknown := range sb.UnknownTypes() {
		log.Printf("apiary: warning: type %q not found — add its package to the scan pattern", unknown)
	}

	// Register security schemes.
	if len(b.security) > 0 {
		components.SecuritySchemes = make(map[string]*SecurityScheme)
		var globalReqs []SecurityRequirement
		for _, entry := range b.security {
			scheme, ok := builtinSchemes[strings.ToLower(entry.builtin)]
			if !ok {
				log.Printf("apiary: warning: unknown security scheme type %q (supported: bearer, basic, apikey)", entry.builtin)
				continue
			}
			components.SecuritySchemes[entry.name] = scheme
			globalReqs = append(globalReqs, SecurityRequirement{entry.name: {}})
		}
		if len(globalReqs) > 0 {
			spec.Security = globalReqs
		}
	}

	if len(components.Schemas) > 0 || len(components.SecuritySchemes) > 0 {
		spec.Components = components
	}
	return spec, nil
}

// buildOperation converts a single OperationInfo into an OpenAPI Operation.
func (b *Builder) buildOperation(
	opInfo *parser.OperationInfo,
	sb *schema.Builder,
	types map[string]*parser.TypeInfo,
) (*Operation, error) {
	ann := opInfo.Annotation
	method := strings.ToUpper(ann.Method)

	op := &Operation{
		Summary:     ann.Summary,
		Description: ann.Description,
		Tags:        ann.Tags,
		Responses:   make(map[string]*Response),
	}

	// Per-operation security override.
	if len(ann.Security) > 0 {
		if len(ann.Security) == 1 && strings.ToLower(ann.Security[0]) == "none" {
			// Explicitly opt out of global security for this endpoint.
			op.Security = []SecurityRequirement{}
		} else {
			reqs := make([]SecurityRequirement, 0, len(ann.Security))
			for _, s := range ann.Security {
				reqs = append(reqs, SecurityRequirement{s: {}})
			}
			op.Security = reqs
		}
	}

	if opInfo.RequestType != nil {
		reqRef := opInfo.RequestType

		if reqRef.IsSlice || reqRef.IsMap {
			op.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]*MediaType{
					"application/json": {Schema: sb.BuildSchema(reqRef)},
				},
			}
		} else {
			typeInfo := types[reqRef.Name]

			var pathFields, queryFields, headerFields, bodyFields []*parser.FieldInfo
			if typeInfo != nil {
				for _, f := range typeInfo.Fields {
					switch {
					case f.PathParam != "":
						pathFields = append(pathFields, f)
					case f.QueryParam != "":
						queryFields = append(queryFields, f)
					case f.Header != "":
						headerFields = append(headerFields, f)
					default:
						bodyFields = append(bodyFields, f)
					}
				}
			}

			for _, f := range pathFields {
				op.Parameters = append(op.Parameters, Parameter{
					Name:        f.PathParam,
					In:          "path",
					Description: f.Doc,
					Required:    true,
					Schema:      sb.BuildSchema(f.Type),
					Example:     nilIfEmpty(f.Example),
				})
			}

			for _, f := range headerFields {
				op.Parameters = append(op.Parameters, Parameter{
					Name:        f.Header,
					In:          "header",
					Description: f.Doc,
					Required:    f.Required,
					Schema:      sb.BuildSchema(f.Type),
					Example:     nilIfEmpty(f.Example),
				})
			}

			for _, f := range queryFields {
				op.Parameters = append(op.Parameters, Parameter{
					Name:        f.QueryParam,
					In:          "query",
					Description: f.Doc,
					Required:    f.Required,
					Schema:      sb.BuildSchema(f.Type),
					Example:     nilIfEmpty(f.Example),
				})
			}

			// For GET/DELETE the remaining fields become implicit query parameters.
			// For POST/PUT/PATCH they go into the JSON request body.
			if method == "GET" || method == "DELETE" {
				for _, f := range bodyFields {
					jsonName := f.JSONName
					if jsonName == "" {
						jsonName = strings.ToLower(f.Name[:1]) + f.Name[1:]
					}
					op.Parameters = append(op.Parameters, Parameter{
						Name:        jsonName,
						In:          "query",
						Description: f.Doc,
						Required:    f.Required,
						Schema:      sb.BuildSchema(f.Type),
						Example:     nilIfEmpty(f.Example),
					})
				}
			} else if len(bodyFields) > 0 || (typeInfo != nil && len(typeInfo.Fields) == len(pathFields)) {
				op.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]*MediaType{
						"application/json": {Schema: sb.BuildSchema(reqRef)},
					},
				}
			} else if len(bodyFields) == 0 && typeInfo != nil {
				// POST/PUT/PATCH with no body fields (all are path/query params) — skip body.
			} else {
				op.RequestBody = &RequestBody{
					Required: true,
					Content: map[string]*MediaType{
						"application/json": {Schema: sb.BuildSchema(reqRef)},
					},
				}
			}
		}
	}

	if opInfo.ResponseType != nil {
		op.Responses["200"] = &Response{
			Description: "OK",
			Content: map[string]*MediaType{
				"application/json": {Schema: sb.BuildSchema(opInfo.ResponseType)},
			},
		}
	} else {
		op.Responses["200"] = &Response{Description: "OK"}
	}

	for _, code := range ann.Errors {
		op.Responses[strconv.Itoa(code)] = &Response{
			Description: httpStatusText(code),
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &schema.Schema{Ref: "#/components/schemas/ErrorResponse"},
				},
			},
		}
	}

	return op, nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func httpStatusText(code int) string {
	texts := map[int]string{
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		409: "Conflict",
		422: "Unprocessable Entity",
		429: "Too Many Requests",
		500: "Internal Server Error",
		502: "Bad Gateway",
		503: "Service Unavailable",
	}
	if t, ok := texts[code]; ok {
		return t
	}
	return "Error"
}
