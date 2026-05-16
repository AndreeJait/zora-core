package gin

import (
	"github.com/AndreeJait/zora-core/port/inbound/health"
	httpw "github.com/AndreeJait/go-utility/v2/httpw/ginw"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all HTTP routes on the Gin engine.
// Public routes (health) are registered directly. Protected routes use a group with AuthMiddleware.
func RegisterRoutes(r *gin.Engine, healthUC health.UseCase) {
	r.GET("/health", httpw.Bind(func(c *gin.Context) (any, error) {
		health := healthUC.Check(c.Request.Context())
		return responsew.Success(health, "Service is healthy"), nil
	}))

	// Protected routes — auth middleware applied at group level

	// Protected routes — auth middleware applied at group level
}

// kyan:service:start
// kyan:service:end
