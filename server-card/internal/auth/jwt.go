// Package auth 提供 JWT 认证功能。
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 JWT 载荷。
type Claims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Manager 提供 JWT 签发和验证。
type Manager struct {
	secret      []byte
	expiryHours int
}

// NewManager 创建 JWT Manager。
func NewManager(secret string, expiryHours int) *Manager {
	return &Manager{
		secret:      []byte(secret),
		expiryHours: expiryHours,
	}
}

// Sign 签发 JWT Token。
func (m *Manager) Sign(userUUID, username string) (string, error) {
	claims := &Claims{
		UserUUID: userUUID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "server-card",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Verify 验证 JWT Token，返回 Claims。
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("Token 验证失败: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("无效的 Token")
	}
	return claims, nil
}
