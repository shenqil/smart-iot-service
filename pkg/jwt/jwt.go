package jwt

import (
	"errors"
	"rbac-admin/pkg/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT 自定义结构体
type JWT struct {
	signingMethod jwt.SigningMethod // 签名算法
	signingKey    []byte            // 签名密钥
	expireHours   int               // 有效期（小时）
}

// Claims 自定义声明
type Claims struct {
	UserID uint `json:"user_id"` // 用户 ID
	jwt.RegisteredClaims
}

var (
	ErrUnsupportedSigningMethod = errors.New("unsupported signing method")
	ErrInvalidSigningMethod     = errors.New("invalid signing method")
	ErrInvalidToken             = errors.New("invalid token")
	ErrTokenExpired             = errors.New("expired token")
)

// NewJWT 初始化 JWT 工具
func NewJWT() (*JWT, error) {
	confJWTAuth := config.Conf.JWTAuth

	var signingMethod jwt.SigningMethod
	switch confJWTAuth.SigningMethod {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		return nil, ErrUnsupportedSigningMethod
	}

	return &JWT{
		signingMethod: signingMethod,
		signingKey:    []byte(confJWTAuth.SigningKey),
		expireHours:   confJWTAuth.Expired,
	}, nil
}

// GenerateToken 生成 JWT Token
func (j *JWT) GenerateToken(userID uint) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "rbac-admin",
		},
	}

	token := jwt.NewWithClaims(j.signingMethod, claims)
	return token.SignedString(j.signingKey)
}

// ParseToken 解析 JWT Token
func (j *JWT) ParseToken(tokenString string) (*Claims, error) {
	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != j.signingMethod {
			return nil, ErrInvalidSigningMethod
		}
		return j.signingKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	// 验证令牌
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
