package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/jwt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"crypto/md5"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MiniProgramTempUser 小程序临时用户信息（用于注册流程）
type MiniProgramTempUser struct {
	OpenID     string // 小程序OpenID
	UnionID    string // 微信UnionID
	SessionKey string // 会话密钥
}

// AuthService 认证服务
type AuthService struct {
	UserRepo            repository.UserRepository
	JwtTool             *jwt.JWT
	qrCodeStore         map[string]*QRCodeState         // 存储扫码状态
	qrCodeMutex         sync.RWMutex                    // 保护 qrCodeStore 的读写锁
	miniProgramTempUser map[string]*MiniProgramTempUser // 存储小程序临时用户信息（key为code）
	miniProgramMutex    sync.RWMutex                    // 保护 miniProgramTempUser 的读写锁
}

// NewAuthService 创建认证服务实例
func NewAuthService(userRepo repository.UserRepository, jwtTool *jwt.JWT) *AuthService {
	return &AuthService{
		UserRepo:            userRepo,
		JwtTool:             jwtTool,
		qrCodeStore:         make(map[string]*QRCodeState),
		miniProgramTempUser: make(map[string]*MiniProgramTempUser),
	}
}

// 超级管理员配置
var rootUser = config.Conf.Root

// Login 用户登录，返回 token 和用户信息
func (s *AuthService) Login(ctx context.Context, username, password, lastLoginIP string) (string, *entity.User, error) {
	// 输入验证
	if strings.TrimSpace(username) == "" {
		return "", nil, errx.New(constant.BadRequest, "用户名不能为空", nil)
	}
	if strings.TrimSpace(password) == "" {
		return "", nil, errx.New(constant.BadRequest, "密码不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var user *entity.User
	var err error

	// 检查是否为超级管理员
	if username == rootUser.UserName {
		userID, err := s.validateRootUser(password)
		if err != nil {
			return "", nil, err
		}
		// Root 用户不在数据库中，创建一个临时的 User 对象
		user = &entity.User{
			Model: gorm.Model{
				ID: userID,
			},
			Username: rootUser.UserName,
			Nickname: rootUser.Nickname,
		}
	} else {
		// 普通用户验证（已通过 Preload 加载 Company）
		user, err = s.validateRegularUser(ctx, username, password)
		if err != nil {
			return "", nil, err
		}
		// 更新最后登录信息
		if err := s.UserRepo.UpdateLastLogin(ctx, user.ID, time.Now(), lastLoginIP); err != nil {
			logger.Warn(fmt.Sprintf("更新登录信息失败: %v", err))
			// 不返回错误，因为登录已经成功
		}
	}

	// 生成 JWT Token
	token, err := s.JwtTool.GenerateToken(user.ID)
	if err != nil {
		logger.Error(fmt.Sprintf("Token生成失败: %v", err))
		return "", nil, errx.New(constant.InternalServerError, "Token生成失败", err)
	}

	return token, user, nil
}

// validateRootUser 验证超级管理员密码
func (s *AuthService) validateRootUser(password string) (uint, error) {
	hash := md5.Sum([]byte(rootUser.Password))
	rootPassword := hex.EncodeToString(hash[:])
	if rootPassword == password {
		return rootUser.ID, nil
	}
	return 0, errx.New(constant.Unauthorized, "用户名或密码错误", nil)
}

// validateRegularUser 验证普通用户
func (s *AuthService) validateRegularUser(ctx context.Context, username, password string) (*entity.User, error) {
	// 查找用户（支持通过用户名、邮箱或手机号登录）
	user, err := s.UserRepo.GetByUsernameOrEmailOrPhone(ctx, username)
	if err != nil {
		// 用户不存在或查询失败，统一返回认证错误
		return nil, errx.New(constant.Unauthorized, "用户名或密码错误", nil)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errx.New(constant.Unauthorized, "用户名或密码错误", nil)
	}

	// 检查用户状态
	if user.Status == entity.UserStatusDisabled {
		return nil, errx.New(constant.Unauthorized, "用户已被禁用", nil)
	}

	return user, nil
}

// ValidateToken 验证Token是否有效
func (s *AuthService) ValidateToken(ctx context.Context, token string) (uint, error) {
	if strings.TrimSpace(token) == "" {
		return 0, errx.New(constant.BadRequest, "Token不能为空", nil)
	}

	// 解析Token获取用户ID
	claims, err := s.JwtTool.ParseToken(token)
	if err != nil {
		return 0, errx.New(constant.Unauthorized, "Token无效或已过期", err)
	}

	return claims.UserID, nil
}

// MiniProgramLoginResult 小程序登录结果
type MiniProgramLoginResult struct {
	Token            string       // JWT Token
	NeedCompleteInfo bool         // 是否需要补充信息
	User             *entity.User // 用户信息（登录成功时返回，避免外部再查询一次）
}

// MiniProgramLogin 小程序登录（只处理 code，判断是否注册）
// 如果用户已注册，返回token和用户信息
// 如果用户未注册，缓存临时信息，返回需要注册
func (s *AuthService) MiniProgramLogin(ctx context.Context, code, lastLoginIP string) (*MiniProgramLoginResult, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证配置
	mpConfig := config.Conf.MiniProgram
	if mpConfig.AppID == "" || mpConfig.AppSecret == "" {
		logger.Error("小程序配置不完整")
		return nil, errx.New(constant.InternalServerError, "小程序配置不完整", nil)
	}

	// 调用微信 API 获取 openid 和 session_key
	sessionResp, err := s.code2Session(ctx, code, mpConfig.AppID, mpConfig.AppSecret)
	if err != nil {
		logger.Error(fmt.Sprintf("调用微信API失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "微信登录失败", err)
	}

	// 检查微信 API 返回的错误
	if sessionResp.ErrCode != 0 {
		logger.Error(fmt.Sprintf("微信API返回错误: %d, %s", sessionResp.ErrCode, sessionResp.ErrMsg))
		return nil, errx.New(constant.Unauthorized, fmt.Sprintf("微信登录失败: %s", sessionResp.ErrMsg), nil)
	}

	// 优先通过UnionID查找用户（与web扫码打通）
	var user *entity.User
	if sessionResp.UnionID != "" {
		user, err = s.UserRepo.GetByField(ctx, "wechat_union_id", sessionResp.UnionID)
		if err != nil {
			// UnionID查找失败，尝试通过小程序openid查找
			user, err = s.UserRepo.GetByField(ctx, "wechat_open_id", sessionResp.OpenID)
		}
	} else {
		// 没有UnionID，通过小程序openid查找
		user, err = s.UserRepo.GetByField(ctx, "wechat_open_id", sessionResp.OpenID)
	}

	if err != nil {
		// 用户不存在，缓存临时信息（使用code作为key），返回需要注册
		tempUser := &MiniProgramTempUser{
			OpenID:     sessionResp.OpenID,
			UnionID:    sessionResp.UnionID,
			SessionKey: sessionResp.SessionKey,
		}

		s.miniProgramMutex.Lock()
		s.miniProgramTempUser[code] = tempUser
		s.miniProgramMutex.Unlock()

		return &MiniProgramLoginResult{
			Token:            "", // 未注册，不返回token
			NeedCompleteInfo: true,
		}, nil
	}

	// 用户存在，更新 UnionID 和 OpenID（如果为空）
	updates := make(map[string]interface{})

	// 更新 UnionID（如果提供了且当前为空）
	if sessionResp.UnionID != "" && (user.WechatUnionID == nil || *user.WechatUnionID == "") {
		updates["wechat_union_id"] = sessionResp.UnionID
		user.WechatUnionID = &sessionResp.UnionID
	}

	// 更新小程序openid（如果为空）
	if user.WechatOpenID == nil || *user.WechatOpenID == "" {
		updates["wechat_open_id"] = sessionResp.OpenID
		user.WechatOpenID = &sessionResp.OpenID
	}

	// 合并为一次数据库更新（如果有需要更新的字段）
	if len(updates) > 0 {
		if err := s.UserRepo.Update(ctx, user.ID, updates); err != nil {
			logger.Warn(fmt.Sprintf("更新用户信息失败: %v", err))
		}
	}

	// 检查用户状态
	if user.Status == entity.UserStatusDisabled {
		return nil, errx.New(constant.Unauthorized, "用户已被禁用", nil)
	}

	// 生成 JWT Token
	token, err := s.JwtTool.GenerateToken(user.ID)
	if err != nil {
		logger.Error(fmt.Sprintf("Token生成失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "Token生成失败", err)
	}

	// 更新最后登录信息
	if err := s.UserRepo.UpdateLastLogin(ctx, user.ID, time.Now(), lastLoginIP); err != nil {
		logger.Warn(fmt.Sprintf("更新登录信息失败: %v", err))
		// 不返回错误，因为登录已经成功
	}

	return &MiniProgramLoginResult{
		Token:            token,
		NeedCompleteInfo: false, // 已注册，不需要补充信息
		User:             user,
	}, nil
}

// MiniProgramCompleteInfo 小程序补充用户信息（注册流程）
// 1. 从缓存获取临时用户信息
// 2. 解密用户信息和手机号
// 3. 检查是否存在相同手机号的用户，如果存在则更新，否则创建新用户
// 4. 账号使用手机号
func (s *AuthService) MiniProgramCompleteInfo(ctx context.Context, code, encryptedData, iv, phoneEncryptedData, phoneIV string) (*MiniProgramLoginResult, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 从缓存获取临时用户信息（使用code作为key）
	s.miniProgramMutex.RLock()
	tempUser, exists := s.miniProgramTempUser[code]
	s.miniProgramMutex.RUnlock()

	if !exists {
		return nil, errx.New(constant.BadRequest, "登录信息已过期，请重新登录", nil)
	}

	sessionKey := tempUser.SessionKey

	// 验证必须提供用户信息和手机号
	if encryptedData == "" || iv == "" {
		return nil, errx.New(constant.BadRequest, "必须授权用户信息", nil)
	}
	if phoneEncryptedData == "" || phoneIV == "" {
		return nil, errx.New(constant.BadRequest, "必须授权手机号", nil)
	}

	// 解密用户信息
	userInfo, err := s.decryptUserInfo(sessionKey, encryptedData, iv)
	if err != nil {
		logger.Error(fmt.Sprintf("解密用户信息失败: %v", err))
		return nil, errx.New(constant.BadRequest, "解密用户信息失败", err)
	}

	// 解密手机号信息
	phoneInfo, err := s.decryptPhoneInfo(sessionKey, phoneEncryptedData, phoneIV)
	if err != nil {
		logger.Error(fmt.Sprintf("解密手机号信息失败: %v", err))
		return nil, errx.New(constant.BadRequest, "解密手机号信息失败", err)
	}

	if phoneInfo.PurePhoneNumber == "" {
		return nil, errx.New(constant.BadRequest, "手机号不能为空", nil)
	}

	// 检查是否存在相同手机号的用户
	var user *entity.User
	user, err = s.UserRepo.GetByField(ctx, "phone", phoneInfo.PurePhoneNumber)
	if err != nil {
		// 不存在相同手机号的用户，创建新用户
		user, err = s.createMiniProgramUser(ctx, tempUser.OpenID, tempUser.UnionID, userInfo, phoneInfo.PurePhoneNumber)
		if err != nil {
			logger.Error(fmt.Sprintf("创建用户失败: %v", err))
			return nil, err
		}
	} else {
		// 存在相同手机号的用户，更新用户信息
		updates := make(map[string]interface{})

		// 更新微信相关信息
		if tempUser.UnionID != "" && (user.WechatUnionID == nil || *user.WechatUnionID == "") {
			updates["wechat_union_id"] = tempUser.UnionID
		}
		if user.WechatOpenID == nil || *user.WechatOpenID == "" {
			updates["wechat_open_id"] = tempUser.OpenID
		}

		// 更新用户信息（昵称、头像、性别），如果数据库存在则优先使用数据库里面的
		if user.Nickname == "" && userInfo.NickName != "" {
			updates["nickname"] = userInfo.NickName
		}
		if user.Avatar == "" && userInfo.AvatarURL != "" {
			updates["avatar"] = userInfo.AvatarURL
		}
		// 转换性别：0=未知, 1=男, 2=女，只有当数据库性别为未知时才更新
		if user.Gender == entity.GenderUnknown {
			switch userInfo.Gender {
			case 1:
				updates["gender"] = entity.GenderMale
			case 2:
				updates["gender"] = entity.GenderFemale
			}
		}

		if len(updates) > 0 {
			if err := s.UserRepo.Update(ctx, user.ID, updates); err != nil {
				logger.Error(fmt.Sprintf("更新用户信息失败: %v", err))
				return nil, errx.New(constant.InternalServerError, "更新用户信息失败", err)
			}
		}
	}

	// 检查用户状态
	if user.Status == entity.UserStatusDisabled {
		return nil, errx.New(constant.Unauthorized, "用户已被禁用", nil)
	}

	// 生成 JWT Token
	token, err := s.JwtTool.GenerateToken(user.ID)
	if err != nil {
		logger.Error(fmt.Sprintf("Token生成失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "Token生成失败", err)
	}

	// 更新最后登录信息
	if err := s.UserRepo.UpdateLastLogin(ctx, user.ID, time.Now(), ""); err != nil {
		logger.Warn(fmt.Sprintf("更新登录信息失败: %v", err))
	}

	// 删除临时用户信息（使用code作为key）
	s.miniProgramMutex.Lock()
	delete(s.miniProgramTempUser, code)
	s.miniProgramMutex.Unlock()

	return &MiniProgramLoginResult{
		Token:            token,
		NeedCompleteInfo: false,
		User:             user,
	}, nil
}

// createMiniProgramUser 创建小程序用户（使用手机号作为账号）
func (s *AuthService) createMiniProgramUser(ctx context.Context, openID, unionID string, userInfo *dto.WechatUserInfo, phone string) (*entity.User, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 生成默认密码（随机字符串，用户无法使用密码登录，只能通过小程序登录）
	defaultPassword := fmt.Sprintf("mp_%s_%d", openID, time.Now().UnixNano())
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(fmt.Sprintf("密码加密失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "创建用户失败", err)
	}

	// 转换性别：0=未知, 1=男, 2=女
	var gender entity.Gender
	switch userInfo.Gender {
	case 1:
		gender = entity.GenderMale
	case 2:
		gender = entity.GenderFemale
	default:
		gender = entity.GenderUnknown
	}

	newUser := &entity.User{
		Username:     phone, // 使用手机号作为账号
		Password:     string(hashedPassword),
		Nickname:     userInfo.NickName,
		Phone:        &phone,
		Status:       entity.UserStatusEnabled,
		Gender:       gender,
		Avatar:       userInfo.AvatarURL,
		WechatOpenID: &openID,
	}

	if unionID != "" {
		newUser.WechatUnionID = &unionID
	}

	if err := s.UserRepo.Create(ctx, newUser); err != nil {
		logger.Error(fmt.Sprintf("创建用户失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "创建用户失败", err)
	}

	return newUser, nil
}

// code2Session 调用微信 code2session 接口
func (s *AuthService) code2Session(ctx context.Context, code, appID, appSecret string) (*dto.WechatCode2SessionResponse, error) {
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appID, appSecret, code)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 创建自定义Transport跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sessionResp dto.WechatCode2SessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, err
	}

	return &sessionResp, nil
}

// decryptUserInfo 解密微信用户信息
func (s *AuthService) decryptUserInfo(sessionKey, encryptedData, iv string) (*dto.WechatUserInfo, error) {
	// Base64 解码
	key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("解码sessionKey失败: %v", err)
	}

	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("解码encryptedData失败: %v", err)
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, fmt.Errorf("解码iv失败: %v", err)
	}

	// AES-128-CBC 解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	if len(ivBytes) != block.BlockSize() {
		return nil, fmt.Errorf("IV长度不正确")
	}

	mode := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// PKCS7 去填充
	decrypted = s.pkcs7Unpadding(decrypted)

	// 解析 JSON
	var userInfo dto.WechatUserInfo
	if err := json.Unmarshal(decrypted, &userInfo); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %v", err)
	}

	return &userInfo, nil
}

// decryptPhoneInfo 解密手机号信息
func (s *AuthService) decryptPhoneInfo(sessionKey, encryptedData, iv string) (*dto.WechatPhoneInfo, error) {
	// Base64 解码
	key, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("解码sessionKey失败: %v", err)
	}

	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("解码encryptedData失败: %v", err)
	}

	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, fmt.Errorf("解码iv失败: %v", err)
	}

	// AES-128-CBC 解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	if len(ivBytes) != block.BlockSize() {
		return nil, fmt.Errorf("IV长度不正确")
	}

	mode := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	// PKCS7 去填充
	decrypted = s.pkcs7Unpadding(decrypted)

	// 解析 JSON
	var phoneInfo dto.WechatPhoneInfo
	if err := json.Unmarshal(decrypted, &phoneInfo); err != nil {
		return nil, fmt.Errorf("解析手机号信息失败: %v", err)
	}

	return &phoneInfo, nil
}

// pkcs7Unpadding PKCS7 去填充
func (s *AuthService) pkcs7Unpadding(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return data
	}
	return data[:(length - unpadding)]
}

// ============================================================================
// 微信扫码登录相关方法
// ============================================================================

// QRCodeState 扫码状态存储结构
type QRCodeState struct {
	Status    dto.QRCodeStatus `json:"status"`     // 扫码状态
	UserID    uint             `json:"user_id"`    // 用户ID（确认后设置）
	OpenID    string           `json:"open_id"`    // 微信OpenID
	UnionID   string           `json:"union_id"`   // 微信UnionID
	CreatedAt int64            `json:"created_at"` // 创建时间戳
}

const (
	qrCodeExpire = 5 * time.Minute // 二维码过期时间
)

// GenerateQRCode 生成微信扫码登录二维码
// 使用微信开放平台的OAuth2.0授权码模式
func (s *AuthService) GenerateQRCode(ctx context.Context) (*dto.WechatQRCodeResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证配置
	webConfig := config.Conf.WechatWeb
	if webConfig.AppID == "" || webConfig.AppSecret == "" || webConfig.RedirectURI == "" {
		logger.Error("微信开放平台配置不完整")
		return nil, errx.New(constant.InternalServerError, "微信开放平台配置不完整", nil)
	}

	// 生成临时token（作为state参数）
	token := uuid.New().String()

	// 构建微信授权URL（扫码登录）
	// 微信开放平台网站应用扫码登录使用OAuth2.0授权码模式
	authURL := fmt.Sprintf("https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		webConfig.AppID,
		url.QueryEscape(webConfig.RedirectURI),
		token)

	// 将状态存储到内存map
	state := &QRCodeState{
		Status:    dto.QRCodeStatusWaiting,
		CreatedAt: time.Now().Unix(),
	}

	s.qrCodeMutex.Lock()
	s.qrCodeStore[token] = state
	s.qrCodeMutex.Unlock()

	// 异步清理过期数据
	go s.cleanupExpiredQRCodeStates()

	return &dto.WechatQRCodeResponse{
		QRCodeURL: authURL, // 这是授权URL，前端需要将其转换为二维码图片
		Token:     token,
		ExpiresIn: int(qrCodeExpire.Seconds()),
	}, nil
}

// GetQRCodeStatus 查询扫码状态
func (s *AuthService) GetQRCodeStatus(ctx context.Context, token string) (*dto.WechatQRCodeStatusResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 从内存map获取状态
	s.qrCodeMutex.RLock()
	state, exists := s.qrCodeStore[token]
	s.qrCodeMutex.RUnlock()

	if !exists {
		return &dto.WechatQRCodeStatusResponse{
			Status: dto.QRCodeStatusExpired,
		}, nil
	}

	// 检查是否过期
	if time.Now().Unix()-state.CreatedAt > int64(qrCodeExpire.Seconds()) {
		// 删除过期数据
		s.qrCodeMutex.Lock()
		delete(s.qrCodeStore, token)
		s.qrCodeMutex.Unlock()
		return &dto.WechatQRCodeStatusResponse{
			Status: dto.QRCodeStatusExpired,
		}, nil
	}

	response := &dto.WechatQRCodeStatusResponse{
		Status: state.Status,
	}

	// 如果已确认，返回token和用户信息
	if state.Status == dto.QRCodeStatusConfirmed && state.UserID > 0 {
		// 生成JWT Token
		jwtToken, err := s.JwtTool.GenerateToken(state.UserID)
		if err != nil {
			logger.Error(fmt.Sprintf("生成Token失败: %v", err))
			return nil, errx.New(constant.InternalServerError, "生成Token失败", err)
		}

		response.AccessToken = jwtToken
	}

	return response, nil
}

// cleanupExpiredQRCodeStates 清理过期的扫码状态
func (s *AuthService) cleanupExpiredQRCodeStates() {
	s.qrCodeMutex.Lock()
	defer s.qrCodeMutex.Unlock()

	now := time.Now().Unix()
	for token, state := range s.qrCodeStore {
		if now-state.CreatedAt > int64(qrCodeExpire.Seconds()) {
			delete(s.qrCodeStore, token)
		}
	}
}

// HandleWechatCallback 处理微信回调（当用户扫码并确认后，微信会调用此接口）
func (s *AuthService) HandleWechatCallback(ctx context.Context, code, state string) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证配置
	webConfig := config.Conf.WechatWeb
	if webConfig.AppID == "" || webConfig.AppSecret == "" {
		return errx.New(constant.InternalServerError, "微信开放平台配置不完整", nil)
	}

	// 使用code获取access_token和openid
	userInfo, err := s.getWechatUserInfo(ctx, code, webConfig.AppID, webConfig.AppSecret)
	if err != nil {
		logger.Error(fmt.Sprintf("获取微信用户信息失败: %v", err))
		return errx.New(constant.InternalServerError, "获取微信用户信息失败", err)
	}

	// 从内存map获取扫码状态（state就是token）
	s.qrCodeMutex.Lock()
	qrState, exists := s.qrCodeStore[state]
	s.qrCodeMutex.Unlock()

	if !exists {
		return errx.New(constant.BadRequest, "二维码已过期或不存在", nil)
	}

	// 检查是否过期
	if time.Now().Unix()-qrState.CreatedAt > int64(qrCodeExpire.Seconds()) {
		s.qrCodeMutex.Lock()
		delete(s.qrCodeStore, state)
		s.qrCodeMutex.Unlock()
		return errx.New(constant.BadRequest, "二维码已过期", nil)
	}

	// 检查状态
	if qrState.Status != dto.QRCodeStatusWaiting && qrState.Status != dto.QRCodeStatusScanned {
		return errx.New(constant.BadRequest, "二维码状态无效", nil)
	}

	// 查找或创建用户
	// 优先使用UnionID查找（跨应用统一），这是关联小程序和网页登录的关键
	// WechatOpenID字段存储小程序的openid，WechatWebOpenID字段存储网页的openid
	// 两个openid不同，但UnionID相同，用于关联同一用户
	var user *entity.User
	if userInfo.UnionID != "" {
		// 优先通过UnionID查找用户（这样可以关联小程序和网页登录的同一用户）
		user, err = s.UserRepo.GetByField(ctx, "wechat_union_id", userInfo.UnionID)
		if err != nil {
			// UnionID查找失败，尝试通过网页openid查找（备用方案）
			user, err = s.UserRepo.GetByField(ctx, "wechat_web_open_id", userInfo.OpenID)
			if err != nil {
				// 都查找失败，创建新用户
				user, err = s.createWechatWebUser(ctx, userInfo.OpenID, userInfo.UnionID)
				if err != nil {
					logger.Error(fmt.Sprintf("创建用户失败: %v", err))
					return err
				}
			} else {
				// 通过网页openid找到了用户，更新UnionID（如果为空）
				if userInfo.UnionID != "" && (user.WechatUnionID == nil || *user.WechatUnionID == "") {
					updates := map[string]interface{}{
						"wechat_union_id": userInfo.UnionID,
					}
					if err := s.UserRepo.Update(ctx, user.ID, updates); err != nil {
						logger.Warn(fmt.Sprintf("更新用户UnionID失败: %v", err))
					} else {
						user.WechatUnionID = &userInfo.UnionID
					}
				}
			}
		} else {
			// 通过UnionID找到了用户，更新网页openid（如果为空）
			updates := make(map[string]interface{})
			needUpdate := false

			if user.WechatWebOpenID == nil || *user.WechatWebOpenID == "" {
				updates["wechat_web_open_id"] = userInfo.OpenID
				needUpdate = true
			}

			// 更新UnionID（如果提供了且当前为空，虽然理论上不应该为空）
			if userInfo.UnionID != "" && (user.WechatUnionID == nil || *user.WechatUnionID == "") {
				updates["wechat_union_id"] = userInfo.UnionID
				needUpdate = true
			}

			if needUpdate {
				if err := s.UserRepo.Update(ctx, user.ID, updates); err != nil {
					logger.Warn(fmt.Sprintf("更新用户信息失败: %v", err))
				} else {
					if webOpenID, ok := updates["wechat_web_open_id"]; ok {
						webOpenIDStr := webOpenID.(string)
						user.WechatWebOpenID = &webOpenIDStr
					}
					if unionID, ok := updates["wechat_union_id"]; ok {
						unionIDStr := unionID.(string)
						user.WechatUnionID = &unionIDStr
					}
				}
			}
		}
	} else {
		// 没有UnionID，尝试通过网页openid查找
		user, err = s.UserRepo.GetByField(ctx, "wechat_web_open_id", userInfo.OpenID)
		if err != nil {
			// 查找失败，创建新用户
			// 注意：这种情况应该很少见，因为微信开放平台通常都会返回UnionID
			user, err = s.createWechatWebUser(ctx, userInfo.OpenID, userInfo.UnionID)
			if err != nil {
				logger.Error(fmt.Sprintf("创建用户失败: %v", err))
				return err
			}
		}
		// 如果通过openid找到了用户，但用户没有UnionID，这种情况理论上不应该发生
		// 因为如果用户之前通过小程序登录过，应该有UnionID
	}

	// 检查用户状态
	if user.Status == entity.UserStatusDisabled {
		return errx.New(constant.Unauthorized, "用户已被禁用", nil)
	}

	// 更新扫码状态为已确认
	s.qrCodeMutex.Lock()
	qrState.Status = dto.QRCodeStatusConfirmed
	qrState.UserID = user.ID
	qrState.OpenID = userInfo.OpenID
	qrState.UnionID = userInfo.UnionID
	s.qrCodeMutex.Unlock()

	// 更新最后登录信息
	if err := s.UserRepo.UpdateLastLogin(ctx, user.ID, time.Now(), ""); err != nil {
		logger.Warn(fmt.Sprintf("更新登录信息失败: %v", err))
	}

	return nil
}

// getWechatUserInfo 通过code获取微信用户信息
func (s *AuthService) getWechatUserInfo(ctx context.Context, code, appID, appSecret string) (*dto.WechatWebUserInfo, error) {
	// 调用微信API通过code获取access_token和openid
	tokenURL := fmt.Sprintf("https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		appID, appSecret, code)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return nil, err
	}

	// 创建自定义Transport跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		OpenID       string `json:"openid"`
		Scope        string `json:"scope"`
		UnionID      string `json:"unionid"`
		ErrCode      int    `json:"errcode"`
		ErrMsg       string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("微信API返回错误: %d, %s", result.ErrCode, result.ErrMsg)
	}

	return &dto.WechatWebUserInfo{
		OpenID:  result.OpenID,
		UnionID: result.UnionID,
	}, nil
}

// createWechatWebUser 创建微信网页用户
// WechatOpenID字段存储小程序的openid，WechatWebOpenID字段存储网页的openid
// UnionID用于关联小程序和网页登录的同一用户
// 注意：网页扫码登录没有手机号，使用openID或unionID作为用户名
func (s *AuthService) createWechatWebUser(ctx context.Context, openID, unionID string) (*entity.User, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 使用UnionID或openID作为用户名（网页扫码登录没有手机号）
	uniqueID := unionID
	if uniqueID == "" {
		uniqueID = openID
	}
	// 使用简单格式：web_ + 唯一标识的前16位
	usernamePrefix := uniqueID
	if len(uniqueID) > 16 {
		usernamePrefix = uniqueID[:16]
	}
	defaultUsername := fmt.Sprintf("web_%s", usernamePrefix)

	// 生成默认密码
	defaultPassword := fmt.Sprintf("web_%s_%d", uniqueID, time.Now().UnixNano())
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(fmt.Sprintf("密码加密失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "创建用户失败", err)
	}

	newUser := &entity.User{
		Username:        defaultUsername,
		Password:        string(hashedPassword),
		Nickname:        fmt.Sprintf("微信用户_%s", uniqueID[:min(8, len(uniqueID))]),
		Status:          entity.UserStatusEnabled,
		Gender:          entity.GenderUnknown,
		WechatWebOpenID: &openID, // 存储网页的openid
	}

	if unionID != "" {
		newUser.WechatUnionID = &unionID
	}

	if err := s.UserRepo.Create(ctx, newUser); err != nil {
		logger.Error(fmt.Sprintf("创建用户失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "创建用户失败", err)
	}

	return newUser, nil
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
