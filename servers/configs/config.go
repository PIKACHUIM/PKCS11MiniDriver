// Package configs 提供 servers 的配置加载。
package configs

import (
	"fmt"
	"os"
	"strconv"
)

// Config 是 servers 的全局配置。
type Config struct {
	API            APIConfig
	Database       DatabaseConfig
	JWT            JWTConfig
	Log            LogConfig
	AllowedOrigins []string // CORS 允许的来源列表
	MasterKeyFile  string   // 主密钥文件路径（优先使用环境变量 SERVER_CARD_MASTER_KEY）
}

// APIConfig 是 HTTP API 配置。
type APIConfig struct {
	Host    string
	Port    int
	BaseURL string // 外部可访问的基础 URL（用于支付回调等），默认由 Host:Port 拼接
}

// Addr 返回监听地址。
func (c *APIConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// DatabaseConfig 是数据库配置。
type DatabaseConfig struct {
	// Path 是 SQLite 数据库文件路径（生产环境可替换为 PostgreSQL DSN）
	Path string
}

// JWTConfig 是 JWT 配置。
type JWTConfig struct {
	// Secret 是 JWT 签名密钥（生产环境应从环境变量或密钥管理服务获取）
	Secret string
	// ExpiryHours 是 Token 有效期（小时）
	ExpiryHours int
}

// LogConfig 是日志配置。
type LogConfig struct {
	Level string
}

// Load 从环境变量加载配置，未设置时使用默认值。
func Load() *Config {
	// 解析 CORS 允许来源列表
	allowedOrigins := parseAllowedOrigins(getEnv("SERVER_CARD_ALLOWED_ORIGINS", ""))

	cfg := &Config{
		API: APIConfig{
			Host:    getEnv("SERVER_CARD_HOST", "127.0.0.1"),
			Port:    getEnvInt("SERVER_CARD_PORT", 1027),
			BaseURL: getEnv("SERVER_CARD_BASE_URL", ""),
		},
		Database: DatabaseConfig{
			Path: getEnv("SERVER_CARD_DB_PATH", defaultDBPath()),
		},
		JWT: JWTConfig{
			Secret:      getEnv("SERVER_CARD_JWT_SECRET", "change-me-in-production-please!!"),
			ExpiryHours: getEnvInt("SERVER_CARD_JWT_EXPIRY_HOURS", 24),
		},
		Log: LogConfig{
			Level: getEnv("SERVER_CARD_LOG_LEVEL", "info"),
		},
		AllowedOrigins: allowedOrigins,
		MasterKeyFile:  getEnv("SERVER_CARD_MASTER_KEY_FILE", ""),
	}

	// BaseURL 未设置时，回退到 http://Host:Port
	if cfg.API.BaseURL == "" {
		cfg.API.BaseURL = fmt.Sprintf("http://%s:%d", cfg.API.Host, cfg.API.Port)
	}

	return cfg
}

// IsAllowedOrigin 检查请求来源是否在 CORS 白名单中。
// 默认只允许 localhost（任意端口）。
func (c *Config) IsAllowedOrigin(origin string) bool {
	if len(c.AllowedOrigins) == 0 {
		// 默认只允许 localhost
		return isLocalhost(origin)
	}
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// isLocalhost 检查 origin 是否为 localhost。
func isLocalhost(origin string) bool {
	return len(origin) > 0 &&
		(hasPrefix(origin, "http://localhost") ||
			hasPrefix(origin, "https://localhost") ||
			hasPrefix(origin, "http://127.0.0.1") ||
			hasPrefix(origin, "https://127.0.0.1"))
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// parseAllowedOrigins 解析逗号分隔的来源列表。
func parseAllowedOrigins(s string) []string {
	if s == "" {
		return nil
	}
	var origins []string
	for _, o := range splitComma(s) {
		o = trimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// defaultDBPath 返回默认数据库路径。
func defaultDBPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return home + "/.config/globaltrusts/servers/server.db"
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
