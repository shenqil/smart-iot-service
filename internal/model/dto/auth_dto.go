package dto

import (
	"errors"
	"strings"
)

// ============================================================================
// 普通登录（账号密码登录）
// ============================================================================

// LoginRequest 登录请求参数
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名
	Password string `json:"password" binding:"required"` // 密码
}

// Validate 验证登录请求
func (req *LoginRequest) Validate() error {
	if strings.TrimSpace(req.Username) == "" {
		return errors.New("用户名不能为空")
	}
	if strings.TrimSpace(req.Password) == "" {
		return errors.New("密码不能为空")
	}
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return errors.New("用户名长度必须在3-50个字符之间")
	}
	return nil
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken string                    `json:"accessToken"` // JWT Token
	Info        *UserResponse             `json:"info"`        // 当前用户信息
	Companies   []*CompanyWithRoleResponse `json:"companies"`   // 公司列表（包含角色）
}

// ============================================================================
// 小程序登录相关
// ============================================================================

// MiniProgramLoginRequest 小程序登录请求参数
type MiniProgramLoginRequest struct {
	Code string `json:"code" binding:"required"` // 微信小程序登录code，通过wx.login()获取
}

// Validate 验证小程序登录请求
func (req *MiniProgramLoginRequest) Validate() error {
	if strings.TrimSpace(req.Code) == "" {
		return errors.New("code不能为空")
	}
	return nil
}

// MiniProgramLoginResponse 小程序登录响应
type MiniProgramLoginResponse struct {
	AccessToken      string                    `json:"accessToken"`      // JWT Token
	Info             *UserResponse             `json:"info"`             // 当前用户信息
	Companies        []*CompanyWithRoleResponse `json:"companies"`        // 公司列表（包含角色）
	NeedCompleteInfo bool                      `json:"needCompleteInfo"` // 是否需要补充信息（昵称、头像、手机号）
}

// MiniProgramCompleteInfoRequest 小程序补充信息请求参数（注册流程）
type MiniProgramCompleteInfoRequest struct {
	Code               string `json:"code" binding:"required"`      // 微信小程序登录code，通过wx.login()获取（用于获取session_key）
	EncryptedData      string `json:"encryptedData,omitempty"`      // 加密的用户信息（可选），通过wx.getUserProfile()获取
	IV                 string `json:"iv,omitempty"`                 // 初始向量（可选），用于解密用户信息
	PhoneIV            string `json:"phoneIV,omitempty"`            // 手机号解密初始向量（可选），通过getPhoneNumber获取
	PhoneEncryptedData string `json:"phoneEncryptedData,omitempty"` // 加密的手机号信息（可选），通过getPhoneNumber获取
}

// Validate 验证补充信息请求
func (req *MiniProgramCompleteInfoRequest) Validate() error {
	if strings.TrimSpace(req.Code) == "" {
		return errors.New("code不能为空")
	}
	// 至少需要提供一种加密数据
	if req.EncryptedData == "" && req.PhoneEncryptedData == "" {
		return errors.New("至少需要提供用户信息或手机号加密数据")
	}
	if req.EncryptedData != "" && req.IV == "" {
		return errors.New("提供用户信息时必须提供IV")
	}
	if req.PhoneEncryptedData != "" && req.PhoneIV == "" {
		return errors.New("提供手机号信息时必须提供PhoneIV")
	}
	return nil
}

// ============================================================================
// 微信扫码登录相关
// ============================================================================

// WechatQRCodeResponse 微信扫码登录二维码响应
type WechatQRCodeResponse struct {
	QRCodeURL string `json:"qrCodeUrl"` // 二维码URL
	Token     string `json:"token"`     // 临时token，用于轮询查询扫码状态
	ExpiresIn int    `json:"expiresIn"` // 过期时间（秒）
}

// WechatQRCodeStatusRequest 查询扫码状态请求
type WechatQRCodeStatusRequest struct {
	Token string `json:"token" binding:"required"` // 临时token
}

// Validate 验证查询扫码状态请求
func (req *WechatQRCodeStatusRequest) Validate() error {
	if strings.TrimSpace(req.Token) == "" {
		return errors.New("token不能为空")
	}
	return nil
}

// QRCodeStatus 扫码状态枚举
type QRCodeStatus string

const (
	QRCodeStatusWaiting   QRCodeStatus = "waiting"   // 等待扫码
	QRCodeStatusScanned   QRCodeStatus = "scanned"   // 已扫码，等待确认
	QRCodeStatusConfirmed QRCodeStatus = "confirmed" // 已确认，可以获取token
	QRCodeStatusCancelled QRCodeStatus = "cancelled" // 已取消
	QRCodeStatusExpired   QRCodeStatus = "expired"   // 已过期
)

// WechatQRCodeStatusResponse 扫码状态响应
type WechatQRCodeStatusResponse struct {
	Status      QRCodeStatus                `json:"status"`      // 扫码状态
	AccessToken string                      `json:"accessToken"` // JWT Token（仅在status为confirmed时返回）
	Info        *UserResponse               `json:"info"`        // 当前用户信息（仅在status为confirmed时返回）
	Companies   []*CompanyWithRoleResponse  `json:"companies"`   // 公司列表（包含角色，仅在status为confirmed时返回）
}

// ============================================================================
// 小程序解密后的数据（内部使用）
// ============================================================================

// WechatUserInfo 解密后的微信用户信息（小程序）
type WechatUserInfo struct {
	NickName  string `json:"nickName"`  // 用户昵称
	AvatarURL string `json:"avatarUrl"` // 用户头像
	Gender    int    `json:"gender"`    // 性别 0：未知、1：男、2：女
	Country   string `json:"country"`   // 国家
	Province  string `json:"province"`  // 省份
	City      string `json:"city"`      // 城市
	Language  string `json:"language"`  // 语言
}

// WechatPhoneInfo 解密后的手机号信息（小程序）
type WechatPhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`     // 用户绑定的手机号（国外手机号会有区号）
	PurePhoneNumber string `json:"purePhoneNumber"` // 没有区号的手机号
	CountryCode     string `json:"countryCode"`     // 区号
}

// ============================================================================
// 微信扫码登录解密后的数据（内部使用）
// ============================================================================

// WechatWebUserInfo 微信网页用户信息（用于扫码登录）
type WechatWebUserInfo struct {
	OpenID  string `json:"openid"`  // 用户唯一标识
	UnionID string `json:"unionid"` // 用户在开放平台的唯一标识符（可选）
}

// ============================================================================
// 微信 API 响应（内部使用）
// ============================================================================

// WechatCode2SessionResponse 微信 code2session 接口响应
type WechatCode2SessionResponse struct {
	OpenID     string `json:"openid"`      // 用户唯一标识
	SessionKey string `json:"session_key"` // 会话密钥
	UnionID    string `json:"unionid"`     // 用户在开放平台的唯一标识符（可选）
	ErrCode    int    `json:"errcode"`     // 错误码
	ErrMsg     string `json:"errmsg"`      // 错误信息
}
