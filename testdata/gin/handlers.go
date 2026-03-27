//go:build ignore

// Package gin — пример того же task-manager, но с gin-хендлерами.
//
// Запуск:
//
//	apiary -security bearer -title "Task Manager API (gin)" -version "1.0.0" -out docs/tasks_gin.yaml ./testdata/gin
package gin

import "github.com/gin-gonic/gin"

// apiary:operation POST /api/v1/auth/login
// summary: Войти
// description: Возвращает JWT по логину и паролю.
// tags: auth
// security: none
// request: LoginRequest
// response: LoginResponse
// errors: 400,401,500
func Login(c *gin.Context) {}

// apiary:operation POST /api/v1/auth/refresh
// summary: Обновить токен
// tags: auth
// security: bearer
// request: RefreshRequest
// response: LoginResponse
// errors: 401,500
func Refresh(c *gin.Context) {}

// apiary:operation GET /api/v1/tasks
// summary: Список задач
// description: Поддерживает фильтрацию по статусу и приоритету, пагинацию.
// tags: tasks
// request: ListTasksRequest
// response: ListTasksResponse
// errors: 400,401,500
func ListTasks(c *gin.Context) {}

// apiary:operation GET /api/v1/tasks/{id}
// summary: Задача по ID
// tags: tasks
// request: GetTaskRequest
// response: TaskDTO
// errors: 401,404,500
func GetTask(c *gin.Context) {}

// apiary:operation POST /api/v1/tasks
// summary: Создать задачу
// tags: tasks
// request: CreateTaskRequest
// response: TaskDTO
// errors: 400,401,422,500
func CreateTask(c *gin.Context) {}

// apiary:operation PUT /api/v1/tasks/{id}
// summary: Обновить задачу
// tags: tasks
// request: UpdateTaskRequest
// response: TaskDTO
// errors: 400,401,403,404,422,500
func UpdateTask(c *gin.Context) {}

// apiary:operation DELETE /api/v1/tasks/{id}
// summary: Удалить задачу
// description: Только создатель задачи или администратор.
// tags: tasks
// request: DeleteTaskRequest
// response: DeleteTaskResponse
// errors: 401,403,404,500
func DeleteTask(c *gin.Context) {}

// apiary:operation GET /api/v1/tasks/{task_id}/comments
// summary: Комментарии к задаче
// tags: comments
// request: ListCommentsRequest
// response: ListCommentsResponse
// errors: 401,404,500
func ListComments(c *gin.Context) {}

// apiary:operation POST /api/v1/tasks/{task_id}/comments
// summary: Добавить комментарий
// tags: comments
// request: AddCommentRequest
// response: CommentDTO
// errors: 400,401,404,500
func AddComment(c *gin.Context) {}
