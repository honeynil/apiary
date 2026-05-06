package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/honeynil/apiary/internal/parser"
)

const sampleCode = `package sample

import "context"

// apiary:operation POST /api/v1/hello
// summary: Hello world
// tags: test
// errors: 400,500
func (h *Handler) Hello(ctx context.Context, req HelloRequest) (HelloResponse, error) {
	return HelloResponse{}, nil
}

type Handler struct{}

type HelloRequest struct {
	Name string ` + "`" + `json:"name" validate:"required" doc:"Your name" example:"Alice"` + "`" + `
}

type HelloResponse struct {
	Message string ` + "`" + `json:"message" doc:"Greeting message"` + "`" + `
}
`

func writeTempFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseDir_HappyPath(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "handler.go", sampleCode)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}

	op := ops[0]
	if op.Annotation.Method != "POST" {
		t.Errorf("expected POST, got %s", op.Annotation.Method)
	}
	if op.Annotation.Path != "/api/v1/hello" {
		t.Errorf("unexpected path: %s", op.Annotation.Path)
	}
	if op.Annotation.Summary != "Hello world" {
		t.Errorf("unexpected summary: %s", op.Annotation.Summary)
	}
	if op.RequestType == nil || op.RequestType.Name != "HelloRequest" {
		t.Errorf("expected HelloRequest, got %v", op.RequestType)
	}
	if op.ResponseType == nil || op.ResponseType.Name != "HelloResponse" {
		t.Errorf("expected HelloResponse, got %v", op.ResponseType)
	}
	if len(op.Annotation.Errors) != 2 {
		t.Errorf("expected 2 error codes, got %d", len(op.Annotation.Errors))
	}

	types := p.Types()
	req, ok := types["HelloRequest"]
	if !ok {
		t.Fatal("HelloRequest not found in types")
	}
	if len(req.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(req.Fields))
	}
	f := req.Fields[0]
	if f.JSONName != "name" {
		t.Errorf("expected json name 'name', got %q", f.JSONName)
	}
	if !f.Required {
		t.Error("expected field to be required")
	}
	if f.Doc != "Your name" {
		t.Errorf("unexpected doc: %q", f.Doc)
	}
	if f.Example != "Alice" {
		t.Errorf("unexpected example: %q", f.Example)
	}
}

func TestParseDir_SkipsInvalidSignature(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

// apiary:operation GET /api/v1/bad
// summary: Bad signature
func badFunc() error {
	return nil
}
`
	writeTempFile(t, dir, "bad.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	if len(p.Operations()) != 0 {
		t.Fatalf("expected 0 operations for invalid signature, got %d", len(p.Operations()))
	}
}

func TestParseDir_SkipsNoAnnotation(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "context"

// This function has no apiary marker
func (h *Handler) NoOp(ctx context.Context, req struct{}) (struct{}, error) {
	return struct{}{}, nil
}

type Handler struct{}
`
	writeTempFile(t, dir, "noop.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	if len(p.Operations()) != 0 {
		t.Fatalf("expected 0 operations, got %d", len(p.Operations()))
	}
}

func TestParseDir_NoCtxSignature(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

// apiary:operation GET /health
// summary: Health check
// security: none
func (h *Handler) Health(req HealthRequest) (HealthResponse, error) {
	return HealthResponse{}, nil
}

type Handler struct{}
type HealthRequest struct{}
type HealthResponse struct {
	Status string ` + "`" + `json:"status"` + "`" + `
}
`
	writeTempFile(t, dir, "health.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation for no-ctx signature, got %d", len(ops))
	}
	if ops[0].RequestType == nil || ops[0].RequestType.Name != "HealthRequest" {
		t.Errorf("expected HealthRequest, got %v", ops[0].RequestType)
	}
	if ops[0].Annotation.Security[0] != "none" {
		t.Errorf("expected security none, got %v", ops[0].Annotation.Security)
	}
}

func TestParseDir_NoParamsSignature(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

// apiary:operation GET /ping
// summary: Ping
func (h *Handler) Ping() (PingResponse, error) {
	return PingResponse{}, nil
}

type Handler struct{}
type PingResponse struct {
	OK bool ` + "`" + `json:"ok"` + "`" + `
}
`
	writeTempFile(t, dir, "ping.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].RequestType != nil {
		t.Errorf("expected nil RequestType, got %v", ops[0].RequestType)
	}
	if ops[0].ResponseType == nil || ops[0].ResponseType.Name != "PingResponse" {
		t.Errorf("expected PingResponse, got %v", ops[0].ResponseType)
	}
}

func TestParseDir_HeaderTag(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "context"

// apiary:operation GET /api/v1/products
// summary: List products
func (h *Handler) ListProducts(ctx context.Context, req ProductsRequest) (struct{}, error) {
	return struct{}{}, nil
}

type Handler struct{}
type ProductsRequest struct {
	Currency string ` + "`" + `header:"X-Currency" doc:"ISO 4217 currency" example:"RUB"` + "`" + `
	Page     int    ` + "`" + `query:"page" default:"1"` + "`" + `
}
`
	writeTempFile(t, dir, "products.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	types := p.Types()
	req, ok := types["ProductsRequest"]
	if !ok {
		t.Fatal("ProductsRequest not found")
	}
	var headerField, queryField *parser.FieldInfo
	for _, f := range req.Fields {
		if f.Header != "" {
			headerField = f
		}
		if f.QueryParam != "" {
			queryField = f
		}
	}
	if headerField == nil {
		t.Fatal("expected a field with header tag")
	}
	if headerField.Header != "X-Currency" {
		t.Errorf("expected X-Currency, got %q", headerField.Header)
	}
	if queryField == nil {
		t.Fatal("expected a field with query tag")
	}
}

func TestParseDir_GinHandler(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "github.com/gin-gonic/gin"

// apiary:operation GET /api/v1/users
// summary: List users
// request: ListUsersRequest
// response: []UserDTO
func ListUsers(c *gin.Context) {}

type ListUsersRequest struct {
	Page int ` + "`" + `query:"page"` + "`" + `
}
type UserDTO struct {
	ID   int64  ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
}
`
	writeTempFile(t, dir, "users.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	op := ops[0]
	if op.Annotation.Method != "GET" {
		t.Errorf("expected GET, got %s", op.Annotation.Method)
	}
	if op.RequestType == nil || op.RequestType.Name != "ListUsersRequest" {
		t.Errorf("expected ListUsersRequest, got %v", op.RequestType)
	}
	if op.ResponseType == nil || !op.ResponseType.IsSlice {
		t.Errorf("expected slice response, got %v", op.ResponseType)
	}
	if op.ResponseType.Elem == nil || op.ResponseType.Elem.Name != "UserDTO" {
		t.Errorf("expected UserDTO elem, got %v", op.ResponseType.Elem)
	}
}

func TestParseDir_StdlibHTTPHandler_FreeFunc(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "net/http"

// apiary:operation POST /api/v1/auth/login
// summary: Login
// request: LoginRequest
// response: LoginResponse
func Login(w http.ResponseWriter, r *http.Request) {}

type LoginRequest struct {
	User string ` + "`" + `json:"user"` + "`" + `
}
type LoginResponse struct {
	Token string ` + "`" + `json:"token"` + "`" + `
}
`
	writeTempFile(t, dir, "login.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	op := ops[0]
	if op.RequestType == nil || op.RequestType.Name != "LoginRequest" {
		t.Errorf("expected LoginRequest, got %v", op.RequestType)
	}
	if op.ResponseType == nil || op.ResponseType.Name != "LoginResponse" {
		t.Errorf("expected LoginResponse, got %v", op.ResponseType)
	}
}

func TestParseDir_StdlibHTTPHandler_Method(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "net/http"

type AuthHandler struct{}

// apiary:operation POST /api/v1/auth/login
// summary: Login
// request: LoginRequest
// response: LoginResponse
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {}

type LoginRequest struct {
	User string ` + "`" + `json:"user"` + "`" + `
}
type LoginResponse struct {
	Token string ` + "`" + `json:"token"` + "`" + `
}
`
	writeTempFile(t, dir, "auth.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	op := ops[0]
	if op.RequestType == nil || op.RequestType.Name != "LoginRequest" {
		t.Errorf("expected LoginRequest, got %v", op.RequestType)
	}
	if op.ResponseType == nil || op.ResponseType.Name != "LoginResponse" {
		t.Errorf("expected LoginResponse, got %v", op.ResponseType)
	}
}

func TestParseDir_StdlibHTTPHandler_NoRequestAnnotation(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "net/http"

// apiary:operation POST /api/v1/ping
// summary: Ping
// response: PingResponse
func Ping(w http.ResponseWriter, r *http.Request) {}

type PingResponse struct {
	OK bool ` + "`" + `json:"ok"` + "`" + `
}
`
	writeTempFile(t, dir, "ping.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	if ops[0].RequestType != nil {
		t.Errorf("expected nil RequestType, got %v", ops[0].RequestType)
	}
}

func TestParseDir_StdlibHTTPHandler_WrongShapes(t *testing.T) {
	cases := map[string]string{
		"returns_error": `package sample
import "net/http"
// apiary:operation GET /a
// summary: x
// response: R
func F(w http.ResponseWriter, r *http.Request) error { return nil }
type R struct{}
`,
		"writer_ptr": `package sample
import "net/http"
// apiary:operation GET /a
// summary: x
// response: R
func F(w *http.ResponseWriter, r *http.Request) {}
type R struct{}
`,
		"one_param": `package sample
import "net/http"
// apiary:operation GET /a
// summary: x
// response: R
func F(w http.ResponseWriter) {}
type R struct{}
`,
	}
	for name, code := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeTempFile(t, dir, "f.go", code)
			p := parser.New()
			if err := p.ParseDir(dir); err != nil {
				t.Fatal(err)
			}
			if ops := p.Operations(); len(ops) != 0 {
				t.Errorf("expected 0 operations, got %d", len(ops))
			}
		})
	}
}

func TestParseDir_SliceResponse(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

import "context"

// apiary:operation GET /api/v1/items
// summary: List items
func (h *Handler) ListItems(ctx context.Context) ([]ItemDTO, error) {
	return nil, nil
}

type Handler struct{}
type ItemDTO struct {
	ID int64 ` + "`" + `json:"id"` + "`" + `
}
`
	writeTempFile(t, dir, "items.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	ops := p.Operations()
	if len(ops) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(ops))
	}
	resp := ops[0].ResponseType
	if resp == nil || !resp.IsSlice {
		t.Fatalf("expected slice ResponseType, got %v", resp)
	}
	if resp.Elem == nil || resp.Elem.Name != "ItemDTO" {
		t.Errorf("expected ItemDTO elem, got %v", resp.Elem)
	}
}

func TestParseDir_EnumConsts(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

type Status string

const (
	StatusActive  Status = "active"
	StatusPending Status = "pending"
	StatusClosed  Status = "closed"
)

type Item struct {
	Name   string ` + "`" + `json:"name"` + "`" + `
	Status Status ` + "`" + `json:"status"` + "`" + `
}
`
	writeTempFile(t, dir, "enums.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	enums := p.Enums()
	info, ok := enums["Status"]
	if !ok {
		t.Fatal("Status enum not found")
	}
	if info.BaseType != "string" {
		t.Errorf("expected base type string, got %q", info.BaseType)
	}
	if len(info.Values) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(info.Values), info.Values)
	}
	if info.Values[0] != "active" || info.Values[1] != "pending" || info.Values[2] != "closed" {
		t.Errorf("unexpected enum values: %v", info.Values)
	}
}

func TestParseDir_EnumIota(t *testing.T) {
	dir := t.TempDir()
	code := `package sample

type Role int

const (
	RoleAdmin Role = iota
	RoleUser
	RoleModerator
)
`
	writeTempFile(t, dir, "role.go", code)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}

	enums := p.Enums()
	info, ok := enums["Role"]
	if !ok {
		t.Fatal("Role enum not found")
	}
	if info.BaseType != "int" {
		t.Errorf("expected base type int, got %q", info.BaseType)
	}
	if len(info.Values) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(info.Values), info.Values)
	}
	if info.Values[0] != 0 || info.Values[1] != 1 || info.Values[2] != 2 {
		t.Errorf("unexpected iota values: %v", info.Values)
	}
}

func TestParseDir_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "handler.go", sampleCode)

	code2 := `package sample

import "context"

// apiary:operation GET /api/v1/health
// summary: Health check
func (h *Handler) Health(ctx context.Context, req HealthRequest) (HealthResponse, error) {
	return HealthResponse{}, nil
}

type HealthRequest struct{}
type HealthResponse struct {
	Status string ` + "`" + `json:"status"` + "`" + `
}
`
	writeTempFile(t, dir, "health.go", code2)

	p := parser.New()
	if err := p.ParseDir(dir); err != nil {
		t.Fatal(err)
	}
	if len(p.Operations()) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(p.Operations()))
	}
}
