// Package sample показывает как аннотировать хендлеры для apiary.
//
//	apiary -security bearer -title "Sample API" -version "0.1.0" -out docs/sample.yaml ./testdata/sample
package sample

import "context"

type AuthHandler struct{}
type UserHandler struct{}
type ProductHandler struct{}
type HealthHandler struct{}

// apiary:operation POST /api/v1/auth/telegram
// summary: Аутентификация через Telegram
// description: Принимает initData из Telegram WebApp, верифицирует HMAC-SHA256 подпись.
// tags: auth
// security: none
// errors: 400,401,500
func (h *AuthHandler) TelegramAuth(ctx context.Context, req TelegramAuthRequest) (TelegramAuthResponse, error) {
	return TelegramAuthResponse{}, nil
}

// apiary:operation GET /api/v1/users/{id}
// summary: Профиль пользователя
// tags: users
// errors: 401,403,404,500
func (h *UserHandler) GetUser(ctx context.Context, req GetUserRequest) (UserResponse, error) {
	return UserResponse{}, nil
}

// apiary:operation POST /api/v1/users
// summary: Создать пользователя
// tags: users
// errors: 400,409,500
func (h *UserHandler) CreateUser(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	return UserResponse{}, nil
}

// apiary:operation GET /api/v1/products
// summary: Список товаров
// description: Поддерживает пагинацию и полнотекстовый поиск. Цены возвращаются в валюте из заголовка X-Currency.
// tags: products
// errors: 400,500
func (h *ProductHandler) ListProducts(ctx context.Context, req ListProductsRequest) (ListProductsResponse, error) {
	return ListProductsResponse{}, nil
}

// apiary:operation GET /health
// summary: Healthcheck
// tags: infra
// security: none
func (h *HealthHandler) Health(req HealthRequest) (HealthResponse, error) {
	return HealthResponse{Status: "ok"}, nil
}
