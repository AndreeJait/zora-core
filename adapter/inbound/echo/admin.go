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

// RegisterAdminRoutes registers REST routes for admin management.
func RegisterAdminRoutes(r RouteRegistrar, whitelistUC whitelist.UseCase) {
	r.GET("/api/v1/admins", echow.Bind(listAdmins(whitelistUC)))
	r.POST("/api/v1/admins", echow.Bind(addAdmin(whitelistUC)))
	r.DELETE("/api/v1/admins/:phone", echow.Bind(removeAdmin(whitelistUC)))
}

// @Summary      List admins
// @Description  Retrieve all admin users
// @Tags         admins
// @Produce      json
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/admins [get]
func listAdmins(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		admins, err := whitelistUC.ListAdmins(c.Request().Context())
		if err != nil {
			return nil, err
		}
		return responsew.Success(admins, "Admins retrieved"), nil
	}
}

type addAdminRequest struct {
	Phone   string   `json:"phone"`
	Name    string   `json:"name"`
	LID     string   `json:"lid"`
	Scope   string   `json:"scope"`     // "personal", "group", "both" (default: "both")
	ChatIDs []string `json:"chat_ids"`  // optional: restrict to these group IDs
}

// @Summary      Add admin
// @Description  Create or update an admin user
// @Tags         admins
// @Accept       json
// @Produce      json
// @Param        body  body  addAdminRequest  true  "Admin details"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/admins [post]
func addAdmin(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		var req addAdminRequest
		if err := json.NewDecoder((*c).Request().Body).Decode(&req); err != nil {
			return nil, err
		}
		if req.Phone == "" {
			return nil, statusw.InvalidReqParam.WithCustomMessage("phone is required")
		}
		if req.Name == "" {
			req.Name = req.Phone
		}
		if err := whitelistUC.AddAdmin(c.Request().Context(), req.Phone, req.Name, req.Scope, req.ChatIDs); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Admin added"), nil
	}
}

// @Summary      Remove admin
// @Description  Remove an admin user by phone number
// @Tags         admins
// @Produce      json
// @Param        phone  path  string  true  "Phone number"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/admins/{phone} [delete]
func removeAdmin(whitelistUC whitelist.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		phone := c.Param("phone")
		if err := whitelistUC.RemoveAdmin(c.Request().Context(), phone); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Admin removed"), nil
	}
}

var _ = http.StatusOK