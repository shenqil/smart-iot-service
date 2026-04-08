package dto

// PolicyToken 请求参数
type PolicyTokenRequest struct {
	SubDir string `form:"subDir,omitempty"` // 可选子目录（如按用户ID细分）
}

// PolicyToken 结构体用于存储生成的表单数据
type PolicyTokenResponse struct {
	AccessKeyId string `json:"ossAccessKeyId"`
	Host        string `json:"host"`
	Signature   string `json:"signature"`
	Policy      string `json:"policy"`
	Directory   string `json:"dir"`
}
