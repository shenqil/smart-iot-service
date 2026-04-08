package database

import (
	"fmt"
	"log"
	"rbac-admin/internal/model/entity"

	"gorm.io/gorm"
)

// Migrate 自动迁移数据库表结构
func Migrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&entity.Company{},
		&entity.User{},
		&entity.UserCompany{},
		&entity.DeviceType{},
		&entity.StandardSensor{},
		&entity.StandardRelay{},
		&entity.Device{},
		&entity.Site{},
		&entity.SiteDevice{},
		&entity.DeviceScreenshot{},
		&entity.DeviceAlarm{},
		&entity.DeviceRelayRecord{},
		&entity.MiniProgramMenu{},
		&entity.DeviceSimulator{},
		&entity.WaterFertilizerTemplate{},
		&entity.UserSite{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	fmt.Println("Database migration completed!")
}
