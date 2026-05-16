package mux

import (
	"net/http"

	"github.com/AndreeJait/zora-core/port/inbound/health"
	httpw "github.com/AndreeJait/go-utility/v2/httpw/muxw"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/gorilla/mux"
)

// RegisterRoutes registers all HTTP routes on the Gorilla Mux router.
// Public routes (health) are registered directly. Protected routes use a subrouter with AuthMiddleware.
func RegisterRoutes(r *mux.Router, healthUC health.UseCase) {
	r.HandleFunc("/health", httpw.Bind(func(r *http.Request) (any, error) {
		health := healthUC.Check(r.Context())
		return responsew.Success(health, "Service is healthy"), nil
	})).Methods(http.MethodGet)

	// Protected routes — auth middleware applied at subrouter level
}

// kyan:service:start
// kyan:service:end
