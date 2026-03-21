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
	if op.RequestType != "HelloRequest" {
		t.Errorf("expected HelloRequest, got %s", op.RequestType)
	}
	if op.ResponseType != "HelloResponse" {
		t.Errorf("expected HelloResponse, got %s", op.ResponseType)
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
