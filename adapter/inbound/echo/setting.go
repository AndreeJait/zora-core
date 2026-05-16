package echo

import (
	"encoding/json"
	"net/http"

	"github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/zora-core/port/inbound/setting"
	"github.com/labstack/echo/v5"
)

// RegisterSettingRoutes registers REST routes for runtime settings.
func RegisterSettingRoutes(r RouteRegistrar, settingUC setting.UseCase) {
	r.GET("/api/v1/settings", echow.Bind(listSettings(settingUC)))
	r.GET("/api/v1/settings/:key", echow.Bind(getSetting(settingUC)))
	r.PUT("/api/v1/settings/:key", echow.Bind(updateSetting(settingUC)))
}

// @Summary      List all settings
// @Description  Retrieve all runtime settings as key-value pairs
// @Tags         settings
// @Produce      json
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/settings [get]
func listSettings(settingUC setting.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		settings, err := settingUC.GetAll(c.Request().Context())
		if err != nil {
			return nil, err
		}
		return responsew.Success(settings, "Settings retrieved"), nil
	}
}

// @Summary      Get a setting
// @Description  Retrieve a single setting by key
// @Tags         settings
// @Produce      json
// @Param        key  path  string  true  "Setting key"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/settings/{key} [get]
func getSetting(settingUC setting.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		key := c.Param("key")
		value, err := settingUC.Get(c.Request().Context(), key)
		if err != nil {
			return nil, err
		}
		return responsew.Success(map[string]string{key: value}, "Setting found"), nil
	}
}

type updateSettingRequest struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// @Summary      Update a setting
// @Description  Create or update a runtime setting by key
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        key   path  string                true  "Setting key"
// @Param        body  body  updateSettingRequest  true  "Setting value and optional description"
// @Success      200  {object}  responsew.BaseResponse
// @Security     ApiKeyAuth
// @Router       /api/v1/settings/{key} [put]
func updateSetting(settingUC setting.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		key := c.Param("key")
		var req updateSettingRequest
		if err := json.NewDecoder((*c).Request().Body).Decode(&req); err != nil {
			return nil, err
		}
		if err := settingUC.Set(c.Request().Context(), key, req.Value, req.Description); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Setting updated"), nil
	}
}

var _ = http.StatusOK