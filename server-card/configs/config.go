// Package configs 提供 server-card 的配置加载。
package configs

import (
	"fmt"
	"os"
	"strconv"
)

// Config 是 server-card 的全局配置。
type Config struct {
	API      APIConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Log      LogConfig
}

// APIConfig 是 HTTP API 配置。
type APIConfig struct {
	Host string
	Port int
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
	return &Config{
		API: APIConfig{
			Host: getEnv("SERVER_CARD_HOST", "127.0.0.1"),
			Port: getEnvInt("SERVER_CARD_PORT", 1027),
		},
		Database: DatabaseConfig{
			Path: getEnv("SERVER_CARD_DB_PATH", defaultDBPath()),
		},
		JWT: JWTConfig{
			Secret:      getEnv("SERVER_CARD_JWT_SECRET", "change-me-in-production"),
			ExpiryHours: getEnvInt("SERVER_CARD_JWT_EXPIRY_HOURS", 24),
		},
		Log: LogConfig{
			Level: getEnv("SERVER_CARD_LOG_LEVEL", "info"),
		},
	}
}

// defaultDBPath 返回默认数据库路径。
func defaultDBPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "."
	}
	return home + "/.config/globaltrusts/server-card/server.db"
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
