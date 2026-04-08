package app

import (
	"fmt"
	"net/http"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/logger"

	"github.com/gin-gonic/gin"
)

// StartServerWithConfig 启动服务
func StartServerWithConfig(r *gin.Engine) {

	confHTTP := config.Conf.HTTP

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", confHTTP.Host, confHTTP.Port),
		Handler: r,
	}

	logger.Logger.Info(fmt.Sprintf("Server starting on %s:%d", confHTTP.Host, confHTTP.Port))

	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Logger.Error(fmt.Sprintf("Server error: %v", err))
		panic(fmt.Sprintf("Server error: %v", err))
	}
}
