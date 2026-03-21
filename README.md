# apiary

**apiary** generates an [OpenAPI 3.1](https://spec.openapis.org/oas/v3.1.0) YAML
document from annotated Go source code — no annotation-heavy comments, no code
generation framework, no Swagger v2.

Think of it as **sqlc for OpenAPI**: you write ordinary Go code, add a minimal
marker comment, and the tool does the rest.

---

### Before (swaggo)

```go
// TelegramAuth godoc
// @Summary      Authenticate via Telegram
// @Description  Accepts initData from Telegram WebApp, verifies HMAC signature
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body TelegramAuthRequest true "Request body"
// @Success      200 {object} TelegramAuthResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /api/v1/auth/telegram [post]
func (h *AuthHandler) TelegramAuth(w http.ResponseWriter, r *http.Request) { ... }
```

### After (apiary)

```go
// apiary:operation POST /api/v1/auth/telegram
// summary: Authenticate via Telegram
// description: Accepts initData from Telegram WebApp, verifies HMAC signature
// tags: auth
// security: none
// errors: 400,401,500
func (h *AuthHandler) TelegramAuth(ctx context.Context, req TelegramAuthRequest) (TelegramAuthResponse, error) {
    // business logic — apiary never touches this
}
```

---

## Installation

```bash
go install github.com/honeynil/apiary/cmd/apiary@latest
```

---

## Usage

```bash
# Scan the current module, write to openapi.yaml
apiary ./...

# With JWT security applied globally
apiary -security bearer -title "My API" -version "1.0.0" -out docs/openapi.yaml ./...

# Scan specific package trees
apiary -out docs/openapi.yaml ./internal/handler/... ./internal/dto/...
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-out` | `openapi.yaml` | Output file path. Use `-` for stdout. |
| `-title` | `API` | Value of `info.title` in the spec |
| `-version` | `0.0.1` | Value of `info.version` in the spec |
| `-security` | _(none)_ | Comma-separated global security schemes: `bearer`, `basic`, `apikey` |

---

## Annotation format

Place the marker **directly above** the function, with no blank lines:

```
// apiary:operation METHOD /path
// summary: One-line summary
// description: Longer description (may contain colons)
// tags: tag1, tag2
// security: bearer          ← optional, overrides global
// errors: 400,401,403,500
```

### Supported function signatures

All of the following shapes are accepted:

```go
func (h *T) A(ctx context.Context, req MyRequest) (MyResponse, error) // standard
func (h *T) B(req MyRequest) (MyResponse, error)                       // no ctx
func (h *T) C(ctx context.Context) (MyResponse, error)                 // no request body
func (h *T) D() (MyResponse, error)                                    // health-check style
```

---

## Struct tags

| Tag | Effect |
|-----|--------|
| `json:"name"` | JSON field name. Use `"-"` to exclude the field. |
| `doc:"text"` | `description` in the JSON Schema / parameter |
| `example:"val"` | `example` in the JSON Schema / parameter |
| `default:"val"` | `default` in the JSON Schema |
| `validate:"required"` | Marks the field as `required` |
| `path:"name"` | Path parameter — matches `{name}` in the URL |
| `query:"name"` | Query parameter |
| `header:"name"` | Header parameter (e.g. `X-Currency`, `Authorization`) |

### Parameter routing

| Tag / condition | OpenAPI location |
|---|---|
| `path:"name"` | `parameters[in=path]` — always required |
| `query:"name"` | `parameters[in=query]` |
| `header:"name"` | `parameters[in=header]` |
| Remaining fields on `GET`/`DELETE` | implicit query parameters |
| Remaining fields on `POST`/`PUT`/`PATCH` | JSON request body |

---

## Security

```bash
# Define JWT Bearer as the global default
apiary -security bearer ./...
```

This adds `BearerAuth` to `components/securitySchemes` and sets it as the
global `security` requirement. Individual operations can override it:

```go
// apiary:operation POST /api/v1/auth/login
// security: none        ← public, no token required
func (h *AuthHandler) Login(...)

// apiary:operation GET /api/v1/admin/report
// security: bearer      ← explicit (same as global, self-documenting)
func (h *AdminHandler) Report(...)
```

Built-in scheme names:

| Name | Type | Details |
|------|------|---------|
| `bearer` | `http` | `scheme: bearer`, `bearerFormat: JWT` |
| `basic` | `http` | `scheme: basic` |
| `apikey` | `apiKey` | `in: header`, `name: X-API-Key` |

---

## Error responses

`errors: 400,401,500` adds a response entry for each code. All error responses
share the `ErrorResponse` schema:

```yaml
ErrorResponse:
  type: object
  properties:
    error:
      type: string
      description: Human-readable error message
  required: [error]
```

---

## Cross-package types

Apiary resolves types from every directory it scans. If a type lives in a
different package, include that package in the pattern:

```bash
apiary ./internal/handler/... ./internal/dto/...
# or just
apiary ./...
```

If a type cannot be resolved, apiary prints a warning and emits
`{type: object}` as a placeholder — the YAML is still valid.

---

## Example

See [testdata/sample/](testdata/sample/) for a complete annotated service with
path params, query params, header params, security, and a no-ctx handler.

```bash
apiary -security bearer -title "Sample API" -version "0.1.0" \
       -out docs/sample.yaml ./testdata/sample
```

---

## Go type -> JSON Schema mapping

| Go type | JSON Schema |
|---------|-------------|
| `string` | `{type: string}` |
| `bool` | `{type: boolean}` |
| `int`, `int32` | `{type: integer, format: int32}` |
| `int64` | `{type: integer, format: int64}` |
| `float32` | `{type: number, format: float}` |
| `float64` | `{type: number, format: double}` |
| `time.Time` | `{type: string, format: date-time}` |
| `[]T` | `{type: array, items: ...}` |
| `map[K]V` | `{type: object, additionalProperties: ...}` |
| Struct | `{$ref: '#/components/schemas/TypeName'}` |
| `interface{}` / `any` | `{}` (any) |
