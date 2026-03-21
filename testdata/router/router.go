// Package router — пример task-manager сервиса с аннотациями apiary.
//
//	apiary -security bearer -title "Task Manager API" -version "1.0.0" -out docs/tasks.yaml ./testdata/router
package router

import "context"

type AuthHandler struct{}
type TaskHandler struct{}
type CommentHandler struct{}

// apiary:operation POST /api/v1/auth/login
// summary: Войти
// description: Возвращает JWT по логину и паролю.
// tags: auth
// security: none
// errors: 400,401,500
func (h *AuthHandler) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

// apiary:operation POST /api/v1/auth/refresh
// summary: Обновить токен
// tags: auth
// security: bearer
// errors: 401,500
func (h *AuthHandler) Refresh(ctx context.Context, req RefreshRequest) (LoginResponse, error) {
	return LoginResponse{}, nil
}

// apiary:operation GET /api/v1/tasks
// summary: Список задач
// description: Поддерживает фильтрацию по статусу и приоритету, пагинацию.
// tags: tasks
// errors: 400,401,500
func (h *TaskHandler) List(ctx context.Context, req ListTasksRequest) (ListTasksResponse, error) {
	return ListTasksResponse{}, nil
}

// apiary:operation GET /api/v1/tasks/{id}
// summary: Задача по ID
// tags: tasks
// errors: 401,404,500
func (h *TaskHandler) Get(ctx context.Context, req GetTaskRequest) (TaskDTO, error) {
	return TaskDTO{}, nil
}

// apiary:operation POST /api/v1/tasks
// summary: Создать задачу
// tags: tasks
// errors: 400,401,422,500
func (h *TaskHandler) Create(ctx context.Context, req CreateTaskRequest) (TaskDTO, error) {
	return TaskDTO{}, nil
}

// apiary:operation PUT /api/v1/tasks/{id}
// summary: Обновить задачу
// tags: tasks
// errors: 400,401,403,404,422,500
func (h *TaskHandler) Update(ctx context.Context, req UpdateTaskRequest) (TaskDTO, error) {
	return TaskDTO{}, nil
}

// apiary:operation DELETE /api/v1/tasks/{id}
// summary: Удалить задачу
// description: Только создатель задачи или администратор.
// tags: tasks
// errors: 401,403,404,500
func (h *TaskHandler) Delete(ctx context.Context, req DeleteTaskRequest) (DeleteTaskResponse, error) {
	return DeleteTaskResponse{}, nil
}

// apiary:operation GET /api/v1/tasks/{task_id}/comments
// summary: Комментарии к задаче
// tags: comments
// errors: 401,404,500
func (h *CommentHandler) List(ctx context.Context, req ListCommentsRequest) (ListCommentsResponse, error) {
	return ListCommentsResponse{}, nil
}

// apiary:operation POST /api/v1/tasks/{task_id}/comments
// summary: Добавить комментарий
// tags: comments
// errors: 400,401,404,500
func (h *CommentHandler) Add(ctx context.Context, req AddCommentRequest) (CommentDTO, error) {
	return CommentDTO{}, nil
}
