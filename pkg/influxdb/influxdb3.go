package influxdb

import (
	"fmt"
	"rbac-admin/pkg/config"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// NewInfluxdb3 创建InfluxDB客户端
func NewInfluxdb3() (*influxdb3.Client, error) {
	confInflux := config.Conf.InfluxDB

	client, err := influxdb3.New(influxdb3.ClientConfig{
		Host:     confInflux.Host,
		Token:    confInflux.Token,
		Database: confInflux.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("创建InfluxDB客户端失败: %v", err)
	}

	return client, nil
}
