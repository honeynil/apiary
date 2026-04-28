package sample

type TelegramAuthRequest struct {
	InitData string `json:"init_data" validate:"required" doc:"initData из Telegram WebApp" example:"query_id=AAH4pFf..."`
}

type TelegramAuthResponse struct {
	User      UserDTO `json:"user"`
	ExpiresIn int     `json:"expires_in" doc:"TTL токена в секундах" example:"3600"`
	IsNewUser bool    `json:"is_new_user"`
}

type GetUserRequest struct {
	ID int64 `path:"id" validate:"required" example:"42"`
}

type UserResponse struct {
	User UserDTO `json:"user"`
}

type CreateUserRequest struct {
	Username  string `json:"username"   validate:"required" example:"larry_somik"`
	Email     string `json:"email"      validate:"required" example:"larry@example.com"`
	FirstName string `json:"first_name" example:"Ларри"`
	LastName  string `json:"last_name"  example:"Сомик"`
}

type ListProductsRequest struct {
	Currency string `header:"X-Currency" doc:"Валюта цен (ISO 4217)" example:"RUB"`
	Page     int    `query:"page"        default:"1"  example:"1"`
	PageSize int    `query:"page_size"   default:"20" example:"20"`
	Search   string `query:"search"      example:"ноутбук"`
}

type ListProductsResponse struct {
	Products []ProductDTO `json:"products"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

type HealthRequest struct{}

type HealthResponse struct {
	Status  string `json:"status"  example:"ok"`
	Version string `json:"version" example:"1.2.3"`
}

type UserDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type ProductDTO struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Price       float64         `json:"price"    example:"1990.00"`
	InStock     bool            `json:"in_stock"`
	Category    ProductCategory `json:"category" doc:"Категория товара"`
}

type ProductCategory string

const (
	CategoryElectronics ProductCategory = "electronics"
	CategoryClothing    ProductCategory = "clothing"
	CategoryFood        ProductCategory = "food"
	CategoryBooks       ProductCategory = "books"
)
