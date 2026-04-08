package main

//go:generate go run -mod=mod github.com/swaggo/swag/cmd/swag@latest init --parseDependency --parseInternal

import (
	"fmt"
	"log"
	"rbac-admin/internal/app"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/logger"
)

// @title RBAC 系统 API 文档
// @version 1.0
// @description 基于 Gin 和 RBAC 的权限管理系统 API 文档。
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @BasePath /
// @schemes http https
func main() {

	fmt.Printf("Run Mode: %s\n", config.Conf.App.RunMode)

	logger.InitLogger()

	injector, cleanup, err := app.InitializeApp()
	if err != nil {
		log.Fatalf("failed to InitializeApp: %v", err)
	}
	defer cleanup()

	go injector.Collector.Start()
	go injector.DeviceSimulatorManager.Start()
	injector.AppScheduler.Start()
	defer injector.AppScheduler.Stop()

	app.StartServerWithConfig(injector.Engine)

}
