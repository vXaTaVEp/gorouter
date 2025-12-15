package gorouter

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"live-streaming-server/config"
	"live-streaming-server/l"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT claims结构
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func GenerateToken(username string, userID int64) (string, error) {
	jwtConfig := config.JWT()
	expirationTime := time.Now().Add(jwtConfig.GetExpiration())

	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtConfig.GetSecretKey()))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken 验证并解析JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	jwtConfig := config.JWT()

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(jwtConfig.GetSecretKey()), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ExtractTokenFromRequest 从请求头提取token
// 支持从 Authorization: Bearer <token> 格式提取
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	// 检查Bearer格式
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid Authorization header format, should be: Bearer <token>")
	}

	return parts[1], nil
}

// GetUserFromRequest 从请求中提取用户信息（用于中间件）
func GetUserFromRequest(r *http.Request) (*Claims, error) {
	tokenString, err := ExtractTokenFromRequest(r)
	if err != nil {
		return nil, err
	}

	claims, err := ValidateToken(tokenString)
	if err != nil {
		l.Warnf("JWT validation failed: %v", err)
		return nil, err
	}

	return claims, nil
}
