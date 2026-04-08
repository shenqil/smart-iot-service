package middleware

import (
	"log/slog"
	"rbac-admin/internal/cache"
	"rbac-admin/internal/handler/response"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/jwt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	CompanyIDHeader     = "X-Company-ID"
)

// JWTTokenMiddleware 创建JWT Token验证中间件（第一个中间件）
// 只负责解析和验证token，存储用户ID
func JWTTokenMiddleware(jwtService *jwt.JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		tokenString := extractTokenFromHeader(c)
		if tokenString == "" {
			response.Error(c, errx.New(constant.Unauthorized, "请求未包含授权令牌", nil))
			return
		}

		// 解析和验证token
		claims, err := jwtService.ParseToken(tokenString)
		if err != nil {
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("token解析失败", slog.Any("error", err))
			handleJWTError(c, err)
			return
		}

		// 设置用户ID到上下文
		contextx.SetUserIDToRequestContext(c, claims.UserID)

		// 将用户ID添加到日志
		contextx.SetLoggerToRequestContext(c, slog.Uint64("userID", uint64(claims.UserID)))

		c.Next()
	}
}

// UserPermissionMiddleware 创建用户权限中间件（第二个中间件）
// 通过用户ID获取缓存的用户信息，得到用户角色和所管理的企业列表
// skipPaths: 需要跳过的路径列表（可选），仅支持精确匹配
func UserPermissionMiddleware(userRoleCache *cache.UserRoleCache, skipPaths ...string) gin.HandlerFunc {
	skipMap := make(map[string]bool)
	for _, path := range skipPaths {
		skipMap[path] = true
	}

	return func(c *gin.Context) {
		// 检查是否需要跳过（精确匹配）
		if skipMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 从上下文获取用户ID
		userID, exists := contextx.GetUserIDFromRequestContext(c)
		if !exists {
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("用户ID未设置，请先执行JWTTokenMiddleware")
			response.Error(c, errx.New(constant.Unauthorized, "用户ID未设置", nil))
			return
		}

		// 从请求头获取公司ID
		companyIDStr := c.GetHeader(CompanyIDHeader)
		if companyIDStr == "" {
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("请求未包含公司ID")
			response.Error(c, errx.New(constant.BadRequest, "请求必须包含公司ID", nil))
			return
		}

		// 解析公司ID
		companyID, err := strconv.ParseUint(companyIDStr, 10, 32)
		if err != nil {
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("公司ID格式错误", slog.String("companyID", companyIDStr), slog.Any("error", err))
			response.Error(c, errx.New(constant.BadRequest, "公司ID格式错误", nil))
			return
		}

		companyIDUint := uint(companyID)

		// 通过UserCacheService获取角色
		if userRoleCache == nil { // Changed from userCache
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("用户缓存服务未初始化")
			response.Error(c, errx.New(constant.InternalServerError, "系统错误", nil))
			return
		}

		// 调用GetRole获取角色
		role, err := userRoleCache.GetRole(c.Request.Context(), userID, companyIDUint) // Changed from userCache
		if err != nil {
			logger := contextx.GetLoggerFromContext(c.Request.Context())
			logger.Error("获取用户角色失败", slog.Uint64("userID", uint64(userID)), slog.Uint64("companyID", uint64(companyIDUint)), slog.Any("error", err))

			// 如果是权限错误，返回Forbidden
			if errxErr, ok := err.(*errx.Error); ok && errxErr.Code == constant.Forbidden {
				response.Error(c, err)
				return
			}
			// 其他错误返回Unauthorized
			response.Error(c, errx.New(constant.Unauthorized, "权限验证失败", err))
			return
		}

		// 设置公司ID和角色到上下文
		contextx.SetCompanyIDToRequestContext(c, &companyIDUint)
		contextx.SetUserRoleToRequestContext(c, &role)

		c.Next()
	}
}

// extractTokenFromHeader 从请求头提取token
func extractTokenFromHeader(c *gin.Context) string {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return ""
	}

	if !strings.HasPrefix(authHeader, BearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(authHeader, BearerPrefix)
}

// handleJWTError 处理JWT验证错误
func handleJWTError(c *gin.Context, err error) {
	switch err {
	case jwt.ErrTokenExpired:
		response.Error(c, errx.New(constant.Unauthorized, "令牌已过期", nil))

	case jwt.ErrInvalidToken:
		response.Error(c, errx.New(constant.Unauthorized, "无效的令牌", nil))

	default:
		response.Error(c, errx.New(constant.Unauthorized, "认证失败", nil))
	}
}
