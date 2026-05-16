package echo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/AndreeJait/go-utility/v2/graphw"
	httpw "github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/labstack/echo/v5"
)

// RegisterAgentRoutes registers agent-related routes.
func RegisterAgentRoutes(r RouteRegistrar, agentUC agent.UseCase) {
	r.POST("/api/v1/agent/execute", httpw.Bind(executeAgent(agentUC)))
	r.GET("/api/v1/agent/stream", httpw.Bind(streamAgentHandler(agentUC)))
	r.POST("/api/v1/agent/test", httpw.Bind(testMessage(agentUC)))
}

// @Summary      Execute agent synchronously
// @Description  Run the agent with a task and receive the final result
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        body  body  agent.ExecuteInput  true  "Agent execution input"
// @Success      200  {object}  agent.ExecuteOutput
// @Failure      400  {object}  responsew.BaseResponse
// @Failure      500  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/agent/execute [post]
func executeAgent(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		var input agent.ExecuteInput
		if err := json.NewDecoder((*c).Request().Body).Decode(&input); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
		output, err := agentUC.Execute(c.Request().Context(), input)
		if err != nil {
			return nil, err
		}
		return responsew.Success(output, "Agent execution completed"), nil
	}
}

// @Summary      Stream agent execution
// @Description  Run the agent with a task and stream intermediate steps via Server-Sent Events
// @Tags         agent
// @Produce      text/event-stream
// @Param        task        query  string  true  "Task description"
// @Param        session_id  query  string  false  "Session ID for resuming"
// @Param        source      query  string  false  "Source context (e.g. waha)"
// @Param        chat_id     query  string  false  "Chat ID (for WhatsApp)"
// @Param        message_id  query  string  false  "Message ID (for WhatsApp)"
// @Success      200  {string}  string  "SSE stream"
// @Failure      400  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/agent/stream [get]
func streamAgentHandler(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		task := c.QueryParam("task")
		sessionID := c.QueryParam("session_id")
		source := c.QueryParam("source")
		chatID := c.QueryParam("chat_id")
		messageID := c.QueryParam("message_id")
		userID := c.QueryParam("user_id")
		isAdmin := c.QueryParam("is_admin") == "true"
		if task == "" {
			return nil, fmt.Errorf("task query parameter is required")
		}

		input := agent.ExecuteInput{
			Task:      task,
			SessionID: sessionID,
			Source:    source,
			ChatID:    chatID,
			MessageID: messageID,
			UserID:    userID,
			IsAdmin:   isAdmin,
		}

		w := c.Response()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			return nil, fmt.Errorf("SSE not supported")
		}

		w.WriteHeader(http.StatusOK)

		for step, err := range agentUC.Stream(c.Request().Context(), input) {
			if err != nil {
				data, _ := json.Marshal(map[string]any{"error": err.Error()})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return nil, nil
			}

			stepData, _ := json.Marshal(stepToJSON(step))
			fmt.Fprintf(w, "data: %s\n\n", stepData)
			flusher.Flush()
		}

		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return nil, nil
	}
}

// stepToJSON converts a graphw.Step[entity.ZoraState] to a serializable map.
func stepToJSON(step graphw.Step[entity.ZoraState]) map[string]any {
	return map[string]any{
		"node":  step.Node,
		"nodes": step.Nodes,
		"state": step.State,
	}
}

// testMessageRequest represents the test API request body.
type testMessageRequest struct {
	Message string `json:"message"`
}

// testMessageResponse represents the response for the test API.
type testMessageResponse struct {
	Message  string `json:"message"`
	Response string `json:"response"`
}

// @Summary      Test agent with a message
// @Description  Send a message prefixed with !zora to test agent execution
// @Tags         agent
// @Accept       json
// @Produce      json
// @Param        body  body  testMessageRequest  true  "Test message"
// @Success      200  {object}  testMessageResponse
// @Failure      400  {object}  responsew.BaseResponse
// @Failure      500  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/agent/test [post]
func testMessage(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		var req testMessageRequest
		if err := json.NewDecoder((*c).Request().Body).Decode(&req); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}

		body := strings.TrimSpace(req.Message)
		if !strings.HasPrefix(strings.ToLower(body), "!zora") {
			return responsew.Success(testMessageResponse{
				Message:  req.Message,
				Response: "Not a !zora command. Messages must start with !zora.",
			}, "ok"), nil
		}

		task := strings.TrimSpace(body[5:])
		if task == "" {
			return responsew.Success(testMessageResponse{
				Message:  req.Message,
				Response: "Hi! Ask me anything by typing !zora followed by your question.",
			}, "ok"), nil
		}

		input := agent.ExecuteInput{Task: task}
		output, err := agentUC.Execute(c.Request().Context(), input)
		if err != nil {
			return nil, err
		}

		return responsew.Success(testMessageResponse{
			Message:  req.Message,
			Response: output.Resolution,
		}, "Agent execution completed"), nil
	}
}