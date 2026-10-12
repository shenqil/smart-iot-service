package config

import (
	"log"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	App         AppConfig
	HTTP        HTTPConfig
	Log         LogConfig
	Root        RootConfig
	Redis       RedisConfig
	JWTAuth     JWTAuthConfig
	Gorm        GormConfig
	MySQL       MySQLConfig
	Postgres    PostgresConfig
	AliyunOSS   AliyunOSSConfig
	Collector   CollectorConfig
	InfluxDB    InfluxDBConfig
	Pubsub      PubsubConfig
	MiniProgram MiniProgramConfig
	WechatWeb   WechatWebConfig
	Deploy      DeployConfig
}

// 全局配置
var Conf Config

// AppConfig 应用全局配置
type AppConfig struct {
	RunMode       string `mapstructure:"RunMode"`
	Swagger       bool   `mapstructure:"Swagger"`
	WeatherAPIKey string `mapstructure:"WeatherAPIKey"`
}

// HTTPConfig HTTP 服务配置
type HTTPConfig struct {
	Host             string `mapstructure:"Host"`
	Port             int    `mapstructure:"Port"`
	MaxContentLength int    `mapstructure:"MaxContentLength"`
	MaxLoggerLength  int    `mapstructure:"MaxLoggerLength"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"Level"`      // 日志级别：debug, info, warn, error
	Output     string `mapstructure:"Output"`     // 日志输出目标：stdout / stderr / file
	OutputFile string `mapstructure:"OutputFile"` // 日志目录（当 Output 为 file）
}

// RootConfig 超级管理员账户配置
type RootConfig struct {
	ID       uint   `mapstructure:"ID"`
	UserName string `mapstructure:"UserName"`
	Password string `mapstructure:"Password"`
	Nickname string `mapstructure:"Nickname"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"Addr"`
	Password string `mapstructure:"Password"`
	DB       int    `mapstructure:"DB"`
}

// JWTAuthConfig JWT 认证配置
type JWTAuthConfig struct {
	Enable        bool   `mapstructure:"Enable"`
	SigningMethod string `mapstructure:"SigningMethod"`
	SigningKey    string `mapstructure:"SigningKey"`
	Expired       int    `mapstructure:"Expired"`
	RedisDB       int    `mapstructure:"RedisDB"`
	RedisPrefix   string `mapstructure:"RedisPrefix"`
}

// GormConfig Gorm 数据库配置
type GormConfig struct {
	Debug             bool   `mapstructure:"Debug"`
	DBType            string `mapstructure:"DBType"`
	MaxLifetime       int    `mapstructure:"MaxLifetime"`
	MaxOpenConns      int    `mapstructure:"MaxOpenConns"`
	MaxIdleConns      int    `mapstructure:"MaxIdleConns"`
	TablePrefix       string `mapstructure:"TablePrefix"`
	EnableAutoMigrate bool   `mapstructure:"EnableAutoMigrate"`
}

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host       string `mapstructure:"Host"`
	Port       int    `mapstructure:"Port"`
	User       string `mapstructure:"User"`
	Password   string `mapstructure:"Password"`
	DBName     string `mapstructure:"DBName"`
	Parameters string `mapstructure:"Parameters"`
}

// PostgresConfig PostgreSQL 数据库配置
type PostgresConfig struct {
	Host     string `mapstructure:"Host"`
	Port     int    `mapstructure:"Port"`
	User     string `mapstructure:"User"`
	Password string `mapstructure:"Password"`
	DBName   string `mapstructure:"DBName"`
	SSLMode  string `mapstructure:"SSLMode"`
}

// AliyunOSSConfig 阿里云OSS配置
type AliyunOSSConfig struct {
	AccessKeyId     string `mapstructure:"AccessKeyId"`
	AccessKeySecret string `mapstructure:"AccessKeySecret"`
	Endpoint        string `mapstructure:"Endpoint"`
	BucketName      string `mapstructure:"BucketName"`
	UploadDir       string `mapstructure:"UploadDir"`
	ExpireTime      int64  `mapstructure:"ExpireTime"`
}

// CollectorConfig
type CollectorConfig struct {
	Addr1           string `mapstructure:"Addr1"`
	Addr2           string `mapstructure:"Addr2"`
	Addr3           string `mapstructure:"Addr3"`
	Addr4           string `mapstructure:"Addr4"`
	KeepaliveMinute int    `mapstructure:"KeepaliveMinute"`
	RecvBuffer      int    `mapstructure:"RecvBuffer"`
	SendBuffer      int    `mapstructure:"SendBuffer"`
	ChanSize        int    `mapstructure:"ChanSize"`
	DB              int    `mapstructure:"DB"`
}

// InfluxDBConfig
type InfluxDBConfig struct {
	Host     string `mapstructure:"Host"`
	Token    string `mapstructure:"Token"`
	Database string `mapstructure:"Database"`
}

// PubsubConfig
type PubsubConfig struct {
	DB int `mapstructure:"DB"`
}

// MiniProgramConfig 微信小程序配置
type MiniProgramConfig struct {
	AppID     string `mapstructure:"AppID"`     // 小程序AppID
	AppSecret string `mapstructure:"AppSecret"` // 小程序AppSecret
}

// WechatWebConfig 微信开放平台网站应用配置
type WechatWebConfig struct {
	AppID       string `mapstructure:"AppID"`       // 网站应用AppID
	AppSecret   string `mapstructure:"AppSecret"`   // 网站应用AppSecret
	RedirectURI string `mapstructure:"RedirectURI"` // 回调地址
}

// DeployConfig 部署配置
type DeployConfig struct {
	DeployDir string `mapstructure:"DeployDir"` // 部署目录
}

// LoadConfig 加载配置文件
func init() {
	// 设置配置文件名和类型
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	// 配置文件查找路径，可以设置多个
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	// 反序列化配置到结构体
	if err := viper.Unmarshal(&Conf); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}
}
