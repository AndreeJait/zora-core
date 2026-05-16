package echo

import (
	"encoding/json"
	"net/http"

	"github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	"github.com/labstack/echo/v5"
)

// RegisterWhitelistRoutes registers REST routes for whitelist management.
func RegisterWhitelistRoutes(r RouteRegistrar, whitelistUC whitelist.UseCase) {
	r.GET("/api/v1/whitelist", echow.Bind(listWhitelist(whitelistUC)))
	r.POST("/api/v1/whitelist", echow.Bind(addWhitelist(whitelistUC)))
	r.DELETE("/api/v1/whitelist/:phone", echow.Bind(removeWhitelist(whitelistUC)))
}

// @Summary      List whitelist entries
// @Description  Retrieve all whitelisted users
// @Tags         whitelist
// @Produce      json
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/whitelist [get]
func listWhitelist(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		entries, err := whitelistUC.ListWhitelist(c.Request().Context())
		if err != nil {
			return nil, err
		}
		return responsew.Success(entries, "Whitelist entries retrieved"), nil
	}
}

type addWhitelistRequest struct {
	Phone         string   `json:"phone"`
	Name          string   `json:"name"`
	LID           string   `json:"lid"`
	TokensPerHour int      `json:"tokens_per_hour"` // 0 = unlimited
	Scope         string   `json:"scope"`           // "personal", "group", "both" (default: "both")
	ChatIDs       []string `json:"chat_ids"`        // optional: restrict to these group IDs
}

// @Summary      Add whitelist entry
// @Description  Create or update a whitelisted user
// @Tags         whitelist
// @Accept       json
// @Produce      json
// @Param        body  body  addWhitelistRequest  true  "Whitelist entry details"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/whitelist [post]
func addWhitelist(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		var req addWhitelistRequest
		if err := json.NewDecoder((*c).Request().Body).Decode(&req); err != nil {
			return nil, err
		}
		if req.Phone == "" {
			return nil, statusw.InvalidReqParam.WithCustomMessage("phone is required")
		}
		if req.Name == "" {
			req.Name = req.Phone
		}
		if err := whitelistUC.AddWhitelist(c.Request().Context(), req.Phone, req.Name, req.TokensPerHour, req.Scope, req.ChatIDs); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Whitelist entry added"), nil
	}
}

// @Summary      Remove whitelist entry
// @Description  Remove a whitelisted user by phone number
// @Tags         whitelist
// @Produce      json
// @Param        phone  path  string  true  "Phone number"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/whitelist/{phone} [delete]
func removeWhitelist(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		phone := c.Param("phone")
		if err := whitelistUC.RemoveWhitelist(c.Request().Context(), phone); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Whitelist entry removed"), nil
	}
}

var _ = http.StatusOK