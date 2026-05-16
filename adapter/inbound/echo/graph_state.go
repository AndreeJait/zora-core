package echo

import (
	"github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/labstack/echo/v5"
)

// RegisterGraphStateRoutes registers graph state inspection and manipulation routes.
func RegisterGraphStateRoutes(r RouteRegistrar, agentUC agent.UseCase) {
	r.GET("/api/graph/state/:threadID", echow.Bind(getGraphState(agentUC)))
	r.GET("/api/graph/state/:threadID/history", echow.Bind(getGraphStateHistory(agentUC)))
	r.POST("/api/graph/state/:threadID/redirect", echow.Bind(redirectGraphState(agentUC)))
	r.POST("/api/graph/state/:threadID/revert", echow.Bind(revertGraphState(agentUC)))
}

func getGraphState(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		threadID := c.Param("threadID")
		snapshot, err := agentUC.GetState(threadID)
		if err != nil {
			return nil, err
		}
		return responsew.Success(snapshot, "Graph state retrieved"), nil
	}
}

func getGraphStateHistory(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		threadID := c.Param("threadID")
		checkpoints, err := agentUC.ListCheckpoints(threadID)
		if err != nil {
			return nil, err
		}
		return responsew.Success(checkpoints, "Checkpoint history retrieved"), nil
	}
}

// redirectRequest is the request body for redirecting graph execution.
type redirectRequest struct {
	NextNodes []string `json:"next_nodes"`
}

func redirectGraphState(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		threadID := c.Param("threadID")
		var req redirectRequest
		if err := c.Bind(&req); err != nil {
			return nil, err
		}
		if len(req.NextNodes) == 0 {
			return nil, statusw.InvalidReqParam.WithCustomMessage("next_nodes is required")
		}
		if err := agentUC.Redirect(threadID, req.NextNodes); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Graph redirected"), nil
	}
}

// revertRequest is the request body for reverting to a previous checkpoint.
type revertRequest struct {
	CheckpointID string `json:"checkpoint_id"`
}

func revertGraphState(agentUC agent.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		threadID := c.Param("threadID")
		var req revertRequest
		if err := c.Bind(&req); err != nil {
			return nil, err
		}
		if req.CheckpointID == "" {
			return nil, statusw.InvalidReqParam.WithCustomMessage("checkpoint_id is required")
		}
		if err := agentUC.RevertTo(threadID, req.CheckpointID); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Graph reverted"), nil
	}
}