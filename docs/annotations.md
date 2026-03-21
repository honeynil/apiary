# Apiary annotation format

Apiary discovers API operations by looking for the `// apiary:operation` marker
comment placed **directly above** a Go function. All other comments in the same
block are treated as metadata fields.

## Marker line

```
// apiary:operation METHOD /path
```

| Field    | Description                                |
|----------|--------------------------------------------|
| `METHOD` | HTTP verb — `GET POST PUT PATCH DELETE`    |
| `/path`  | URL path, may contain `{param}` segments   |

## Metadata fields

Each metadata field must appear on its own comment line, directly adjacent to
the marker (no blank lines between them):

```
// apiary:operation POST /api/v1/users
// summary: Create user
// description: Creates a new user account. The username must be unique.
// tags: users, admin
// security: bearer
// errors: 400,409,500
```

| Key           | Value format              | Description                                                       |
|---------------|---------------------------|-------------------------------------------------------------------|
| `summary`     | free text                 | Short one-line description                                        |
| `description` | free text                 | Longer description; may contain colons                            |
| `tags`        | comma-separated strings   | OpenAPI tags used for grouping in UIs                             |
| `errors`      | comma-separated integers  | HTTP status codes that produce an error body                      |
| `security`    | comma-separated names, or `none` | Override global security for this operation (see below)  |

## Supported function signatures

Apiary accepts all of the following shapes. The return list must always be
`(ResponseType, error)`.

```go
// Standard — context + typed request
func (h *T) Method(ctx context.Context, req RequestType) (ResponseType, error)

// No context
func (h *T) Method(req RequestType) (ResponseType, error)

// No request body (ctx only)
func (h *T) Method(ctx context.Context) (ResponseType, error)

// No context, no request body (e.g. health check)
func (h *T) Method() (ResponseType, error)
```

Functions that do not match any of these shapes are silently skipped.

## Struct field tags

Apiary reads the following struct tags when building JSON Schemas and OpenAPI
parameter lists:

| Tag                   | Effect                                                          |
|-----------------------|-----------------------------------------------------------------|
| `json:"name"`         | JSON field name. Use `"-"` to exclude the field.               |
| `doc:"text"`          | `description` in the JSON Schema / parameter                   |
| `example:"val"`       | `example` in the JSON Schema / parameter                       |
| `default:"val"`       | `default` in the JSON Schema                                   |
| `validate:"required"` | Marks the field as `required`                                  |
| `path:"name"`         | Path parameter — matches `{name}` in the URL (always required) |
| `query:"name"`        | Query parameter                                                |
| `header:"name"`       | Header parameter (e.g. `X-API-Key`, `Authorization`)          |

### Parameter routing rules

| Tag / condition | OpenAPI location |
|---|---|
| `path:"name"` | `parameters[in=path]` — always required |
| `query:"name"` | `parameters[in=query]` |
| `header:"name"` | `parameters[in=header]` |
| Remaining fields on `GET`/`DELETE` | implicit `parameters[in=query]` |
| Remaining fields on `POST`/`PUT`/`PATCH` | `requestBody` (JSON) |

## Security

### Global security (CLI flag)

Pass `-security` to define a scheme globally. Every operation inherits it
unless it overrides with `// security:`.

```bash
apiary -security bearer ./...          # JWT Bearer everywhere
apiary -security bearer,apikey ./...   # two schemes, both required
```

Supported built-in names:

| Name     | Type    | Details                                    |
|----------|---------|--------------------------------------------|
| `bearer` | `http`  | `scheme: bearer`, `bearerFormat: JWT`      |
| `basic`  | `http`  | `scheme: basic`                            |
| `apikey` | `apiKey`| `in: header`, `name: X-API-Key`            |

### Per-operation override (`// security:`)

```go
// apiary:operation POST /api/v1/auth/login
// security: none         ← public endpoint, opt out of global security
func (h *AuthHandler) Login(...)

// apiary:operation GET /api/v1/admin/stats
// security: bearer,apikey  ← require both schemes for this endpoint
func (h *AdminHandler) Stats(...)
```

## Error responses

`errors: 400,401,500` generates a response object for each code. All error
responses share the same `ErrorResponse` schema:

```json
{ "error": "string" }
```

## Cross-package types

Apiary resolves struct types from all directories it scans. If you see a
warning like:

```
apiary: warning: type "OrderDTO" not found — add its package to the scan pattern
```

broaden the pattern to include the package that defines the type:

```bash
# Instead of just scanning handlers:
apiary ./internal/handler/...

# Also include the types package:
apiary ./internal/handler/... ./internal/dto/...

# Or simply scan everything:
apiary ./...
```

## Full example

```go
// apiary:operation POST /api/v1/auth/telegram
// summary: Authenticate via Telegram
// description: Accepts initData from Telegram WebApp, verifies HMAC-SHA256.
// tags: auth
// security: none
// errors: 400,401,500
func (h *AuthHandler) TelegramAuth(ctx context.Context, req TelegramAuthRequest) (TelegramAuthResponse, error) {
    // ...
}

type TelegramAuthRequest struct {
    InitData string `json:"init_data" doc:"Telegram WebApp initData" validate:"required" example:"query_id=AAH..."`
}

type TelegramAuthResponse struct {
    User      UserDTO `json:"user"`
    ExpiresIn int     `json:"expires_in" doc:"Token TTL in seconds" example:"3600"`
    IsNewUser bool    `json:"is_new_user"`
}

// Header param example
type ListProductsRequest struct {
    Currency string `header:"X-Currency" doc:"ISO 4217 currency code" example:"EUR"`
    Page     int    `query:"page"        doc:"Page number"            default:"1"`
}

// No-ctx handler example
// apiary:operation GET /health
// summary: Health check
// security: none
func (h *HealthHandler) Health(req HealthRequest) (HealthResponse, error) { ... }
```
