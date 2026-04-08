package handler

import (
	"context"
	"fmt"
	"net/http"
	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证控制器
type AuthHandler struct {
	AuthService    *service.AuthService
	UserService    *service.UserService
	CompanyService *service.CompanyService
}

// buildUserLoginData 构建用户登录数据（转换用户信息、获取管理的企业列表）
// 返回 currentUser 和 companies，如果出错则返回 error
func (h *AuthHandler) buildUserLoginData(ctx context.Context, user *entity.User) (*dto.UserResponse, []*dto.CompanyWithRoleResponse, error) {
	// 转换用户信息为响应格式
	currentUser, err := dto.ConvertToUserResponse(user)
	if err != nil {
		return nil, nil, err
	}

	// 通过关联表查询企业列表和角色
	companies, err := h.UserService.GetUserCompaniesWithRoles(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return currentUser, companies, nil
}

// @Summary 用户登录
// @Description 用户通过用户名和密码登录系统
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录请求参数"
// @Success 200 {object} dto.Response{data=dto.LoginResponse} "成功响应，包含 JWT Token"
// @Failure 400 {object} dto.Response "请求参数无效"
// @Failure 401 {object} dto.Response "用户名或密码错误"
// @Router /api/v1/public/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证
	if err := req.Validate(); err != nil {
		logger.Error(fmt.Sprintf("参数验证失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, err.Error(), nil))
		return
	}

	// 调用认证服务完成登录（直接返回 token 和用户信息）
	token, user, err := h.AuthService.Login(c.Request.Context(), req.Username, req.Password, c.ClientIP())
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建用户登录数据
	currentUser, companies, err := h.buildUserLoginData(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回成功响应
	response.Success(c, dto.LoginResponse{
		AccessToken: token,
		Info:        currentUser,
		Companies:   companies,
	})
}

// @Summary 小程序登录
// @Description 用户通过微信小程序 code 登录系统
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.MiniProgramLoginRequest true "小程序登录请求参数"
// @Success 200 {object} dto.Response{data=dto.MiniProgramLoginResponse} "成功响应，包含 JWT Token"
// @Failure 400 {object} dto.Response "请求参数无效"
// @Failure 401 {object} dto.Response "登录失败"
// @Router /api/v1/public/miniprogram/login [post]
func (h *AuthHandler) MiniProgramLogin(c *gin.Context) {
	var req dto.MiniProgramLoginRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证
	if err := req.Validate(); err != nil {
		logger.Error(fmt.Sprintf("参数验证失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, err.Error(), nil))
		return
	}

	// 调用认证服务完成小程序登录
	loginResult, err := h.AuthService.MiniProgramLogin(
		c.Request.Context(),
		req.Code,
		c.ClientIP(),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 如果用户未注册，直接返回需要补充信息
	if loginResult.NeedCompleteInfo {
		response.Success(c, dto.MiniProgramLoginResponse{
			AccessToken:      loginResult.Token,
			Info:             nil,
			Companies:        nil,
			NeedCompleteInfo: true,
		})
		return
	}

	// 用户已注册，使用登录结果中的用户信息
	user := loginResult.User
	if user == nil {
		response.Error(c, errx.New(constant.InternalServerError, "登录成功但用户信息为空", nil))
		return
	}

	// 构建用户登录数据
	currentUser, companies, err := h.buildUserLoginData(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回成功响应
	response.Success(c, dto.MiniProgramLoginResponse{
		AccessToken:      loginResult.Token,
		Info:             currentUser,
		Companies:        companies,
		NeedCompleteInfo: false,
	})
}

// @Summary 小程序补充用户信息（注册）
// @Description 小程序用户注册，必须提供用户信息和手机号。如果存在相同手机号的用户则更新，否则创建新用户
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.MiniProgramCompleteInfoRequest true "注册请求参数（必须包含用户信息和手机号）"
// @Success 200 {object} dto.Response{data=dto.MiniProgramLoginResponse} "成功响应，包含 JWT Token"
// @Failure 400 {object} dto.Response "请求参数无效或未授权用户信息/手机号"
// @Failure 401 {object} dto.Response "注册失败"
// @Router /api/v1/public/miniprogram/complete-info [post]
func (h *AuthHandler) MiniProgramCompleteInfo(c *gin.Context) {
	var req dto.MiniProgramCompleteInfoRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证（修改验证逻辑，要求必须提供用户信息和手机号）
	if strings.TrimSpace(req.Code) == "" {
		response.Error(c, errx.New(constant.BadRequest, "code不能为空", nil))
		return
	}
	if req.EncryptedData == "" || req.IV == "" {
		response.Error(c, errx.New(constant.BadRequest, "必须授权用户信息", nil))
		return
	}
	if req.PhoneEncryptedData == "" || req.PhoneIV == "" {
		response.Error(c, errx.New(constant.BadRequest, "必须授权手机号", nil))
		return
	}

	// 调用认证服务注册用户
	loginResult, err := h.AuthService.MiniProgramCompleteInfo(
		c.Request.Context(),
		req.Code,
		req.EncryptedData,
		req.IV,
		req.PhoneEncryptedData,
		req.PhoneIV,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 使用登录结果中的用户信息
	user := loginResult.User
	if user == nil {
		response.Error(c, errx.New(constant.InternalServerError, "注册成功但用户信息为空", nil))
		return
	}

	// 构建用户登录数据
	currentUser, companies, err := h.buildUserLoginData(c.Request.Context(), user)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回成功响应
	response.Success(c, dto.MiniProgramLoginResponse{
		AccessToken:      loginResult.Token,
		Info:             currentUser,
		Companies:        companies,
		NeedCompleteInfo: false,
	})
}

// @Summary 用户注销
// @Description 用户通过提供有效的 Token 注销登录
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response "成功响应"
// @Failure 400 {object} dto.Response "请求参数无效"
// @Failure 500 {object} dto.Response "注销失败"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// 目前注销功能为空实现，未来可以添加Token黑名单机制
	// 例如：将Token加入Redis黑名单

	// 返回成功响应
	response.Success(c, gin.H{"message": "注销成功"})
}

// @Summary 生成微信扫码登录二维码
// @Description 生成微信扫码登录二维码，返回二维码URL和临时token用于轮询查询
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=dto.WechatQRCodeResponse} "成功响应，包含二维码URL和token"
// @Failure 500 {object} dto.Response "生成二维码失败"
// @Router /api/v1/public/wechat/qrcode [get]
func (h *AuthHandler) GenerateWechatQRCode(c *gin.Context) {
	// 调用认证服务生成二维码
	qrCodeResp, err := h.AuthService.GenerateQRCode(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回成功响应
	response.Success(c, qrCodeResp)
}

// @Summary 查询微信扫码状态
// @Description 轮询查询微信扫码登录状态，返回扫码状态和登录token（如果已确认）
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body dto.WechatQRCodeStatusRequest true "查询状态请求参数"
// @Success 200 {object} dto.Response{data=dto.WechatQRCodeStatusResponse} "成功响应，包含扫码状态"
// @Failure 400 {object} dto.Response "请求参数无效"
// @Failure 500 {object} dto.Response "查询状态失败"
// @Router /api/v1/public/wechat/qrcode/status [post]
func (h *AuthHandler) GetWechatQRCodeStatus(c *gin.Context) {
	var req dto.WechatQRCodeStatusRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证
	if err := req.Validate(); err != nil {
		logger.Error(fmt.Sprintf("参数验证失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, err.Error(), nil))
		return
	}

	// 调用认证服务查询状态
	statusResp, err := h.AuthService.GetQRCodeStatus(c.Request.Context(), req.Token)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 如果已确认，获取用户信息和公司列表
	if statusResp.Status == dto.QRCodeStatusConfirmed && statusResp.AccessToken != "" {
		// 解析token获取用户ID
		userID, err := h.AuthService.ValidateToken(c.Request.Context(), statusResp.AccessToken)
		if err != nil {
			response.Error(c, err)
			return
		}

		// 获取当前用户实体（已通过连表查询包含企业信息）
		user, err := h.UserService.GetCurrentUserEntity(c.Request.Context(), userID)
		if err != nil {
			response.Error(c, err)
			return
		}

		// 构建用户登录数据
		currentUser, companies, err := h.buildUserLoginData(c.Request.Context(), user)
		if err != nil {
			response.Error(c, err)
			return
		}

		statusResp.Info = currentUser
		statusResp.Companies = companies
	}

	// 返回成功响应
	response.Success(c, statusResp)
}

// @Summary 微信扫码登录回调
// @Description 处理微信扫码登录回调，当用户扫码并确认后，微信会重定向到此接口
// @Tags Auth
// @Accept json
// @Produce json
// @Param code query string true "微信授权码"
// @Param state query string true "状态参数（临时token）"
// @Success 200 {string} string "成功响应，重定向到前端页面"
// @Failure 400 {object} dto.Response "请求参数无效"
// @Failure 500 {object} dto.Response "处理回调失败"
// @Router /api/v1/public/wechat/callback [get]
func (h *AuthHandler) WechatCallback(c *gin.Context) {
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 获取code和state参数
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		logger.Error(fmt.Sprintf("缺少必要参数：code=%s, state=%s", code, state))
		// 重定向到前端错误页面
		c.Redirect(http.StatusFound, "/login?error=invalid_params")
		return
	}

	// 调用认证服务处理回调
	if err := h.AuthService.HandleWechatCallback(c.Request.Context(), code, state); err != nil {
		logger.Error(fmt.Sprintf("处理微信回调失败：%v", err))
		// 重定向到前端错误页面
		c.Redirect(http.StatusFound, "/login?error=callback_failed")
		return
	}

	// 重定向到前端成功页面，带上state参数，前端可以通过state轮询获取token
	c.Redirect(http.StatusFound, fmt.Sprintf("/login?success=true&token=%s", state))
}
