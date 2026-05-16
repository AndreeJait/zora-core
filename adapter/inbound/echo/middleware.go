package echo

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// APIKeyMiddleware returns an Echo middleware that validates the X-API-Key header
// against the provided key. If apiKey is empty, the middleware is a no-op (all requests pass).
func APIKeyMiddleware(apiKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// No API key configured — skip auth
			if apiKey == "" {
				return next(c)
			}

			providedKey := c.Request().Header.Get("X-API-Key")
			if providedKey != apiKey {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error": map[string]any{
						"code":    "UNAUTHENTICATED",
						"message": "invalid or missing API key",
					},
				})
			}

			return next(c)
		}
	}
}