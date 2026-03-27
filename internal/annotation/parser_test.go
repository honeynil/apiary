package annotation_test

import (
	"testing"

	"github.com/honeynil/apiary/internal/annotation"
)

func TestParse_Full(t *testing.T) {
	lines := []string{
		"apiary:operation POST /api/v1/auth/telegram",
		"summary: Authenticate via Telegram",
		"description: Accepts Telegram WebApp initData",
		"tags: auth, users",
		"errors: 400,401,403,500",
	}

	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if op.Method != "POST" {
		t.Errorf("expected POST, got %s", op.Method)
	}
	if op.Path != "/api/v1/auth/telegram" {
		t.Errorf("expected /api/v1/auth/telegram, got %s", op.Path)
	}
	if op.Summary != "Authenticate via Telegram" {
		t.Errorf("unexpected summary: %q", op.Summary)
	}
	if op.Description != "Accepts Telegram WebApp initData" {
		t.Errorf("unexpected description: %q", op.Description)
	}
	if len(op.Tags) != 2 || op.Tags[0] != "auth" || op.Tags[1] != "users" {
		t.Errorf("unexpected tags: %v", op.Tags)
	}
	if len(op.Errors) != 4 {
		t.Errorf("expected 4 error codes, got %d: %v", len(op.Errors), op.Errors)
	}
}

func TestParse_NoMarker(t *testing.T) {
	lines := []string{
		"summary: Some function",
		"description: Does something",
	}
	_, ok := annotation.Parse(lines)
	if ok {
		t.Fatal("expected parse to fail without apiary:operation marker")
	}
}

func TestParse_MethodUppercased(t *testing.T) {
	lines := []string{"apiary:operation get /health"}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if op.Method != "GET" {
		t.Errorf("expected GET, got %s", op.Method)
	}
}

func TestParse_MinimalMarker(t *testing.T) {
	lines := []string{"apiary:operation DELETE /api/v1/users/{id}"}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if op.Method != "DELETE" {
		t.Errorf("expected DELETE, got %s", op.Method)
	}
	if op.Path != "/api/v1/users/{id}" {
		t.Errorf("unexpected path: %s", op.Path)
	}
	if op.Summary != "" || op.Description != "" || len(op.Tags) != 0 || len(op.Errors) != 0 {
		t.Error("expected empty optional fields")
	}
}

func TestParse_DescriptionWithColon(t *testing.T) {
	lines := []string{
		"apiary:operation POST /api/v1/foo",
		"description: Returns key: value pairs",
	}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if op.Description != "Returns key: value pairs" {
		t.Errorf("unexpected description: %q", op.Description)
	}
}

func TestParse_SecurityBearer(t *testing.T) {
	lines := []string{
		"apiary:operation GET /api/v1/me",
		"summary: Current user",
		"security: bearer",
	}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if len(op.Security) != 1 || op.Security[0] != "bearer" {
		t.Errorf("expected security [bearer], got %v", op.Security)
	}
}

func TestParse_SecurityNone(t *testing.T) {
	lines := []string{
		"apiary:operation POST /api/v1/auth/login",
		"security: none",
	}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if len(op.Security) != 1 || op.Security[0] != "none" {
		t.Errorf("expected security [none], got %v", op.Security)
	}
}

func TestParse_SecurityMultiple(t *testing.T) {
	lines := []string{
		"apiary:operation GET /api/v1/admin",
		"security: bearer, apikey",
	}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if len(op.Security) != 2 {
		t.Errorf("expected 2 security schemes, got %v", op.Security)
	}
}

func TestParse_SecurityNilWhenAbsent(t *testing.T) {
	lines := []string{
		"apiary:operation GET /api/v1/items",
		"summary: List items",
	}
	op, ok := annotation.Parse(lines)
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if op.Security != nil {
		t.Errorf("expected nil security (inherit global), got %v", op.Security)
	}
}
