package database

import (
	"fmt"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/logger"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitDatabase 初始化数据库连接
func InitDatabase() (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	confGorm := config.Conf.Gorm
	confMySQL := config.Conf.MySQL
	confPostgres := config.Conf.Postgres

	switch confGorm.DBType {
	case "mysql":
		// 创建不指定数据库的连接
		adminDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?%s",
			confMySQL.User,
			confMySQL.Password,
			confMySQL.Host,
			confMySQL.Port,
			confMySQL.Parameters,
		)
		adminDB, err := gorm.Open(mysql.Open(adminDSN), &gorm.Config{})
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Failed to connect to database: %v", err))
			return nil, err
		}
		adminSQLDB, err := adminDB.DB()
		if err != nil {
			logger.Logger.Warn(fmt.Sprintf("Failed to get admin connection pool: %v", err))
			return nil, err
		} else {
			defer adminSQLDB.Close()
		}

		// 使用 IF NOT EXISTS 创建数据库(如果不存在)
		createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", confMySQL.DBName)
		if err := adminDB.Exec(createDBSQL).Error; err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to create database: %v", err))
			panic(fmt.Sprintf("Failed to create database: %v", err))
		}

		// 创建正式数据库连接（带数据库名）
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
			confMySQL.User,
			confMySQL.Password,
			confMySQL.Host,
			confMySQL.Port,
			confMySQL.DBName,
			confMySQL.Parameters,
		)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			confPostgres.Host,
			confPostgres.User,
			confPostgres.Password,
			confPostgres.DBName,
			confPostgres.Port,
			confPostgres.SSLMode,
		)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	default:
		return nil, fmt.Errorf("unsupported database type: %s", confGorm.DBType)

	}

	if err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(confGorm.MaxIdleConns)
	sqlDB.SetMaxOpenConns(confGorm.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(confGorm.MaxLifetime) * time.Second)

	if confGorm.EnableAutoMigrate {
		Migrate(db)
	}

	return db, nil
}
