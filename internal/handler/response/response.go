package response

import (
	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/errx"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, dto.Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

func Paginate(c *gin.Context, records interface{}, total int64) {
	c.JSON(200, dto.Response{
		Code:    200,
		Message: "success",
		Data: dto.PaginationResponse{
			Records:    records,
			TotalCount: total,
		},
	})
}

func Error(c *gin.Context, err error) {
	if customErr, ok := err.(*errx.Error); ok {
		c.JSON(200, dto.Response{
			Code:    customErr.Code,
			Message: customErr.Message,
		})
		c.Abort()
		return
	}

	c.JSON(200, dto.Response{
		Code:    500,
		Message: "未知错误",
	})
	c.Abort()
}
