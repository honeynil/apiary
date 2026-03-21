package router

import "time"

type TaskDTO struct {
	ID          int64      `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"      doc:"todo | in_progress | done"`
	Priority    int        `json:"priority"    doc:"1 — низкий, 3 — высокий"`
	AssignedTo  *UserBrief `json:"assigned_to"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UserBrief struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type CommentDTO struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Author    UserBrief `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required" example:"larry"`
	Password string `json:"password" validate:"required" example:"nosomword"`
}

type RefreshRequest struct {
	Token string `header:"Authorization" validate:"required" doc:"Bearer <токен>"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in" example:"3600"`
}

type ListTasksRequest struct {
	Status   string `query:"status"    doc:"todo | in_progress | done" example:"todo"`
	Priority int    `query:"priority"  doc:"1-3"`
	Page     int    `query:"page"      default:"1"  example:"1"`
	PageSize int    `query:"page_size" default:"20" example:"20"`
}

type ListTasksResponse struct {
	Tasks    []TaskDTO `json:"tasks"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

type GetTaskRequest struct {
	ID int64 `path:"id" validate:"required" example:"42"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"       validate:"required" example:"Упал прод"`
	Description string `json:"description" example:"OOM killer убил сервис около 03:00"`
	Priority    int    `json:"priority"    doc:"1-3" example:"3"`
	AssigneeID  int64  `json:"assignee_id"`
}

type UpdateTaskRequest struct {
	ID          int64  `path:"id" validate:"required"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"      example:"in_progress"`
	Priority    int    `json:"priority"`
	AssigneeID  int64  `json:"assignee_id"`
}

type DeleteTaskRequest struct {
	ID int64 `path:"id" validate:"required" example:"42"`
}

type DeleteTaskResponse struct {
	ID      int64 `json:"id"`
	Success bool  `json:"success"`
}

type ListCommentsRequest struct {
	TaskID int64 `path:"task_id" validate:"required" example:"42"`
}

type ListCommentsResponse struct {
	Comments []CommentDTO `json:"comments"`
	Total    int          `json:"total"`
}

type AddCommentRequest struct {
	TaskID int64  `path:"task_id" validate:"required" example:"42"`
	Body   string `json:"body"    validate:"required" example:"Смотрел логи — это регрессия из #1337"`
}
