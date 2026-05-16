package echo

import (
	"net/http"
	"strconv"

	"github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/task"
	"github.com/labstack/echo/v5"
)

var _ entity.Task

// RegisterTaskRoutes registers REST routes for task management.
func RegisterTaskRoutes(r RouteRegistrar, taskUC task.UseCase) {
	r.GET("/api/v1/tasks/:id", echow.Bind(getTask(taskUC)))
	r.GET("/api/v1/tasks/:id/graph", echow.Bind(renderTaskGraph(taskUC)))
	r.GET("/api/v1/tasks", echow.Bind(listTasks(taskUC)))
}

// @Summary      Get a task by ID
// @Description  Retrieve a task by its UUID
// @Tags         tasks
// @Produce      json
// @Param        id  path  string  true  "Task ID"
// @Success      200  {object}  entity.Task
// @Failure      404  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks/{id} [get]
func getTask(taskUC task.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		id := c.Param("id")
		result, err := taskUC.Get(c.Request().Context(), id)
		if err != nil {
			return nil, err
		}
		return responsew.Success(result, "Task found"), nil
	}
}

// @Summary      Render task graph
// @Description  Get the Mermaid graph diagram for a task. Use format=presigned to upload to MinIO and get a URL.
// @Tags         tasks
// @Produce      json
// @Param        id      path  string  true  "Task ID"
// @Param        format  query string  false "Response format: mmd (Mermaid text) or presigned (MinIO URL)" default(mmd)
// @Success      200  {object}  responsew.BaseResponse
// @Failure      404  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks/{id}/graph [get]
func renderTaskGraph(taskUC task.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		id := c.Param("id")
		format := c.QueryParam("format")
		if format == "" {
			format = "mmd"
		}
		result, err := taskUC.RenderGraph(c.Request().Context(), id, format)
		if err != nil {
			return nil, err
		}
		if format == "presigned" {
			return responsew.Success(map[string]string{"url": result}, "Graph URL generated"), nil
		}
		return responsew.Success(map[string]string{"mermaid": result}, "Graph rendered"), nil
	}
}

// @Summary      List tasks
// @Description  Retrieve a paginated list of tasks, optionally filtered by status or chat_id
// @Tags         tasks
// @Produce      json
// @Param        status   query  string  false  "Filter by status"
// @Param        chat_id  query  string  false  "Filter by chat ID"
// @Param        page     query  int     false  "Page number (default 1)"
// @Param        per_page query  int     false  "Items per page (default 10)"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks [get]
func listTasks(taskUC task.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
		status := c.QueryParam("status")
		chatID := c.QueryParam("chat_id")

		if chatID != "" {
			items, total, err := taskUC.GetByChatID(c.Request().Context(), chatID, page, perPage)
			if err != nil {
				return nil, err
			}
			return responsew.SuccessPaginated(items, total, page, perPage, "Tasks retrieved"), nil
		}

		items, total, err := taskUC.List(c.Request().Context(), status, page, perPage)
		if err != nil {
			return nil, err
		}
		return responsew.SuccessPaginated(items, total, page, perPage, "Tasks retrieved"), nil
	}
}

var _ = http.StatusOK