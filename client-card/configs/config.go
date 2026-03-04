// Package config 负责加载和管理应用配置。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config 是应用的顶层配置结构。
type Config struct {
	IPC      IPCConfig      `yaml:"ipc"`
	API      APIConfig      `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
	Tray     TrayConfig     `yaml:"tray"`
}

// IPCConfig 是与 pkcs11-mock 通信的 IPC 配置。
type IPCConfig struct {
	PipeName string `yaml:"pipe_name"`
	Timeout  int    `yaml:"timeout"`
}

// APIConfig 是 REST API 服务配置。
type APIConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	TLSEnabled     bool   `yaml:"tls_enabled"`
	TLSCert        string `yaml:"tls_cert"`
	TLSKey         string `yaml:"tls_key"`
	JWTSecret      string `yaml:"jwt_secret"`
	JWTExpireHours int    `yaml:"jwt_expire_hours"`
}

// DatabaseConfig 是 SQLite 数据库配置。
type DatabaseConfig struct {
	Path         string `yaml:"path"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// LogConfig 是日志配置。
type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// TrayConfig 是系统托盘配置。
type TrayConfig struct {
	Enabled bool   `yaml:"enabled"`
	Icon    string `yaml:"icon"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		IPC: IPCConfig{
			PipeName: "client-card",
			Timeout:  30,
		},
		API: APIConfig{
			Host:           "127.0.0.1",
			Port:           1026,
			TLSEnabled:     false,
			JWTExpireHours: 24,
		},
		Database: DatabaseConfig{
			Path:         defaultDBPath(),
			MaxOpenConns: 5,
			MaxIdleConns: 2,
		},
		Log: LogConfig{
			Level: "info",
		},
		Tray: TrayConfig{
			Enabled: true,
		},
	}
}

// Load 从指定路径加载配置文件，若文件不存在则使用默认配置。
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = defaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认配置
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 补全空值
	if cfg.Database.Path == "" {
		cfg.Database.Path = defaultDBPath()
	}

	return cfg, nil
}

// Addr 返回 API 监听地址字符串。
func (c *APIConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IPCPath 返回平台对应的 IPC 路径。
func (c *IPCConfig) IPCPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + c.PipeName
	}
	return filepath.Join(os.TempDir(), c.PipeName+".sock")
}

// defaultConfigPath 返回默认配置文件路径。
func defaultConfigPath() string {
	return filepath.Join(userDataDir(), "config.yaml")
}

// defaultDBPath 返回默认数据库文件路径。
func defaultDBPath() string {
	return filepath.Join(userDataDir(), "client-card.db")
}

// userDataDir 返回用户数据目录。
func userDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "client-card")
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "Application Support", "client-card")
		}
	default:
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return filepath.Join(dir, "client-card")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "client-card")
		}
	}
	return "."
}
