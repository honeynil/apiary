package openapi_test

import (
	"testing"

	"github.com/honeynil/apiary/internal/annotation"
	"github.com/honeynil/apiary/internal/openapi"
	"github.com/honeynil/apiary/internal/parser"
)

func TestBuild_BasicOperation(t *testing.T) {
	ops := []*parser.OperationInfo{
		{
			Annotation: &annotation.Operation{
				Method:  "POST",
				Path:    "/api/v1/auth",
				Summary: "Authenticate",
				Tags:    []string{"auth"},
				Errors:  []annotation.ErrorSpec{{Code: 400}, {Code: 401}},
			},
			RequestType:  &parser.TypeRef{Name: "AuthRequest"},
			ResponseType: &parser.TypeRef{Name: "AuthResponse"},
		},
	}

	types := map[string]*parser.TypeInfo{
		"AuthRequest": {
			Name: "AuthRequest",
			Fields: []*parser.FieldInfo{
				{
					Name:     "Token",
					JSONName: "token",
					Type:     &parser.TypeRef{Name: "string"},
					Required: true,
					Doc:      "Bearer token",
				},
			},
		},
		"AuthResponse": {
			Name: "AuthResponse",
			Fields: []*parser.FieldInfo{
				{
					Name:     "UserID",
					JSONName: "user_id",
					Type:     &parser.TypeRef{Name: "int64"},
				},
			},
		},
	}

	b := openapi.NewBuilder("Test API", "1.0.0")
	spec, err := b.Build(ops, types)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("expected openapi 3.1.0, got %s", spec.OpenAPI)
	}
	if spec.Info.Title != "Test API" {
		t.Errorf("unexpected title: %s", spec.Info.Title)
	}

	item, ok := spec.Paths["/api/v1/auth"]
	if !ok {
		t.Fatal("path /api/v1/auth not found")
	}
	if item.Post == nil {
		t.Fatal("POST operation missing")
	}
	if item.Post.Summary != "Authenticate" {
		t.Errorf("unexpected summary: %s", item.Post.Summary)
	}
	if len(item.Post.Tags) != 1 || item.Post.Tags[0] != "auth" {
		t.Errorf("unexpected tags: %v", item.Post.Tags)
	}

	// 200 + 400 + 401
	if len(item.Post.Responses) != 3 {
		t.Errorf("expected 3 responses, got %d", len(item.Post.Responses))
	}
	if _, ok := item.Post.Responses["200"]; !ok {
		t.Error("missing 200 response")
	}
	if _, ok := item.Post.Responses["400"]; !ok {
		t.Error("missing 400 response")
	}

	if spec.Components == nil {
		t.Fatal("components is nil")
	}
	if _, ok := spec.Components.Schemas["AuthRequest"]; !ok {
		t.Error("AuthRequest not in components")
	}
	if _, ok := spec.Components.Schemas["AuthResponse"]; !ok {
		t.Error("AuthResponse not in components")
	}
	if _, ok := spec.Components.Schemas["ErrorResponse"]; !ok {
		t.Error("ErrorResponse not in components")
	}
}

func TestBuild_GetWithQueryParams(t *testing.T) {
	ops := []*parser.OperationInfo{
		{
			Annotation: &annotation.Operation{
				Method:  "GET",
				Path:    "/api/v1/users",
				Summary: "List users",
			},
			RequestType:  &parser.TypeRef{Name: "ListRequest"},
			ResponseType: &parser.TypeRef{Name: "ListResponse"},
		},
	}

	types := map[string]*parser.TypeInfo{
		"ListRequest": {
			Name: "ListRequest",
			Fields: []*parser.FieldInfo{
				{
					Name:       "Page",
					JSONName:   "page",
					QueryParam: "page",
					Type:       &parser.TypeRef{Name: "int"},
					Default:    "1",
				},
				{
					Name:       "Search",
					JSONName:   "search",
					QueryParam: "search",
					Type:       &parser.TypeRef{Name: "string"},
				},
			},
		},
		"ListResponse": {
			Name:   "ListResponse",
			Fields: []*parser.FieldInfo{},
		},
	}

	b := openapi.NewBuilder("API", "0.1.0")
	spec, err := b.Build(ops, types)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	item := spec.Paths["/api/v1/users"]
	if item.Get == nil {
		t.Fatal("GET operation missing")
	}
	if item.Get.RequestBody != nil {
		t.Error("GET should not have a request body")
	}
	if len(item.Get.Parameters) != 2 {
		t.Errorf("expected 2 query parameters, got %d", len(item.Get.Parameters))
	}
	for _, p := range item.Get.Parameters {
		if p.In != "query" {
			t.Errorf("expected query parameter, got %s", p.In)
		}
	}
}

func TestBuild_PathParameters(t *testing.T) {
	ops := []*parser.OperationInfo{
		{
			Annotation: &annotation.Operation{
				Method:  "GET",
				Path:    "/api/v1/users/{id}",
				Summary: "Get user",
				Errors:  []annotation.ErrorSpec{{Code: 404}},
			},
			RequestType:  &parser.TypeRef{Name: "GetUserRequest"},
			ResponseType: &parser.TypeRef{Name: "UserResponse"},
		},
	}

	types := map[string]*parser.TypeInfo{
		"GetUserRequest": {
			Name: "GetUserRequest",
			Fields: []*parser.FieldInfo{
				{
					Name:      "ID",
					JSONName:  "id",
					PathParam: "id",
					Type:      &parser.TypeRef{Name: "int64"},
					Required:  true,
				},
			},
		},
		"UserResponse": {
			Name: "UserResponse",
			Fields: []*parser.FieldInfo{
				{Name: "ID", JSONName: "id", Type: &parser.TypeRef{Name: "int64"}},
			},
		},
	}

	b := openapi.NewBuilder("API", "0.1.0")
	spec, err := b.Build(ops, types)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	item := spec.Paths["/api/v1/users/{id}"]
	if item.Get == nil {
		t.Fatal("GET operation missing")
	}
	if len(item.Get.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(item.Get.Parameters))
	}
	p := item.Get.Parameters[0]
	if p.In != "path" {
		t.Errorf("expected path parameter, got %s", p.In)
	}
	if p.Name != "id" {
		t.Errorf("expected parameter name 'id', got %q", p.Name)
	}
	if !p.Required {
		t.Error("path parameter must be required")
	}
}

func TestBuild_MultipleOperations(t *testing.T) {
	ops := []*parser.OperationInfo{
		{
			Annotation:   &annotation.Operation{Method: "GET", Path: "/foo"},
			RequestType:  &parser.TypeRef{Name: "FooReq"},
			ResponseType: &parser.TypeRef{Name: "FooResp"},
		},
		{
			Annotation:   &annotation.Operation{Method: "POST", Path: "/bar"},
			RequestType:  &parser.TypeRef{Name: "BarReq"},
			ResponseType: &parser.TypeRef{Name: "BarResp"},
		},
	}

	types := make(map[string]*parser.TypeInfo)
	for _, name := range []string{"FooReq", "FooResp", "BarReq", "BarResp"} {
		types[name] = &parser.TypeInfo{Name: name}
	}

	spec, err := openapi.NewBuilder("Multi", "1.0.0").Build(ops, types)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if len(spec.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(spec.Paths))
	}
}
