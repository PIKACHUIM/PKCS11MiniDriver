// Package auth 提供 JWT 认证功能。
package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 JWT 载荷。
type Claims struct {
	UserUUID string `json:"user_uuid"`
	Username string `json:"username"`
	Role     string `json:"role"` // admin/user/readonly
	jwt.RegisteredClaims
}

// Manager 提供 JWT 签发、验证、黑名单和 Token 轮换。
type Manager struct {
	secret      []byte
	expiryHours int

	// JWT 黑名单（logout 失效）
	blacklist   map[string]time.Time // tokenID -> 过期时间
	blacklistMu sync.RWMutex
}

// NewManager 创建 JWT Manager。
// secret 长度必须 >= 32 字节（256 位），否则返回错误。
func NewManager(secret string, expiryHours int) (*Manager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT 密钥长度不足：需要 >= 32 字节（256 位），当前 %d 字节", len(secret))
	}
	m := &Manager{
		secret:      []byte(secret),
		expiryHours: expiryHours,
		blacklist:   make(map[string]time.Time),
	}
	go m.cleanupBlacklist()
	return m, nil
}

// Sign 签发 JWT Token，返回 token 字符串和 token ID。
func (m *Manager) Sign(userUUID, username, role string) (string, string, error) {
	if role == "" {
		role = "user"
	}
	tokenID := fmt.Sprintf("%s-%d", userUUID, time.Now().UnixNano())
	claims := &Claims{
		UserUUID: userUUID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(m.expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "opencert-platform",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secret)
	if err != nil {
		return "", "", err
	}
	return tokenStr, tokenID, nil
}

// Verify 验证 JWT Token，返回 Claims。
// 同时检查 Token 是否在黑名单中。
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

	// 检查黑名单
	if m.IsBlacklisted(claims.ID) {
		return nil, fmt.Errorf("Token 已失效（已登出）")
	}

	return claims, nil
}

// Revoke 将 Token 加入黑名单（logout 时调用）。
func (m *Manager) Revoke(tokenID string, expiresAt time.Time) {
	m.blacklistMu.Lock()
	defer m.blacklistMu.Unlock()
	m.blacklist[tokenID] = expiresAt
}

// IsBlacklisted 检查 Token 是否在黑名单中。
func (m *Manager) IsBlacklisted(tokenID string) bool {
	m.blacklistMu.RLock()
	defer m.blacklistMu.RUnlock()
	_, ok := m.blacklist[tokenID]
	return ok
}

// NeedsRefresh 检查 Token 是否即将过期（剩余时间 < 总时间的 20%）。
// 返回 true 时，应在响应头中设置 X-Token-Refresh: true。
func (m *Manager) NeedsRefresh(claims *Claims) bool {
	if claims.ExpiresAt == nil || claims.IssuedAt == nil {
		return false
	}
	totalDuration := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	remaining := time.Until(claims.ExpiresAt.Time)
	return remaining < totalDuration/5
}

// Refresh 刷新 Token（Token 轮换）。
// 签发新 Token 并将旧 Token 加入黑名单。
func (m *Manager) Refresh(oldClaims *Claims) (string, string, error) {
	// 将旧 Token 加入黑名单
	if oldClaims.ExpiresAt != nil {
		m.Revoke(oldClaims.ID, oldClaims.ExpiresAt.Time)
	}

	// 签发新 Token
	return m.Sign(oldClaims.UserUUID, oldClaims.Username, oldClaims.Role)
}

// cleanupBlacklist 定期清理过期的黑名单条目。
func (m *Manager) cleanupBlacklist() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.blacklistMu.Lock()
		now := time.Now()
		for id, expiresAt := range m.blacklist {
			if now.After(expiresAt) {
				delete(m.blacklist, id)
			}
		}
		m.blacklistMu.Unlock()
	}
}
