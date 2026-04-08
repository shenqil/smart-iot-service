// weather_handler.go
package handler

import (
	"strconv"

	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"github.com/gin-gonic/gin"
	"log/slog"
	"fmt"
)

type WeatherHandler struct {
	WeatherSer *service.WeatherService
}

// GetWeather 获取天气信息
// @Summary 获取天气信息
// @Tags 天气服务
// @Produce json
// @Param lat query string true "纬度"
// @Param lon query string true "经度"
// @Success 200 {object} dto.Response{data=dto.TencentRealtime}
// @Router /api/v1/auth/weather [get]
func (h *WeatherHandler) GetWeather(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("纬度格式错误: %v", err))
		response.Error(c, errx.New(constant.BadRequest, "纬度格式错误", err))
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("经度格式错误: %v", err))
		response.Error(c, errx.New(constant.BadRequest, "经度格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Float64("lat", lat),
		slog.Float64("lon", lon))

	weather, err := h.WeatherSer.GetWeather(ctx, lat, lon)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, weather)
}
