package dto

// TencentWeatherResponse 腾讯地图天气API响应结构
type TencentWeatherResponse struct {
	Status  int    `json:"status"`  // 状态码，0表示成功
	Message string `json:"message"` // 状态信息
	Result  struct {
		Realtime []TencentRealtime `json:"realtime"` // 实时天气信息
	} `json:"result"` // 天气数据结果
}

// TencentRealtime 腾讯地图实时天气信息
type TencentRealtime struct {
	Province   string `json:"province"`    // 省份
	City       string `json:"city"`        // 城市
	District   string `json:"district"`    // 区县
	UpdateTime string `json:"update_time"` // 更新时间
	Infos      struct {
		Weather       string `json:"weather"`        // 天气状况
		Temperature   int    `json:"temperature"`    // 温度(℃)
		WindDirection string `json:"wind_direction"` // 风向
		WindPower     string `json:"wind_power"`     // 风力
		Humidity      int    `json:"humidity"`       // 湿度(%)
	} `json:"infos"` // 详细信息
}
