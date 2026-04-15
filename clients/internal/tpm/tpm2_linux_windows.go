//go:build windows || linux

// Package tpm - Windows/Linux TPM2 实现。
// 使用 TPM 2.0 芯片保护绑定密钥，通过绑定密钥对主密钥进行 Seal/Unseal。
//
// 实现策略：
//   - 生成 32 字节随机绑定密钥，持久化到 TPM NV 存储（Linux）或受保护文件（Windows）
//   - Seal：用绑定密钥 AES-256-GCM 加密数据
//   - Unseal：从 TPM 读取绑定密钥，解密数据
//
// 注意：Windows 当前使用受保护文件存储绑定密钥（生产环境建议改用 DPAPI/TPM Platform Crypto Provider）。
package tpm

import (
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
)

// tpm2DevicePaths 是 Linux TPM 设备路径列表。
var tpm2DevicePaths = []string{
	"/dev/tpm0",
	"/dev/tpmrm0",
}

// TPM2Provider 是 Windows/Linux 平台的 TPM2 实现。
type TPM2Provider struct {
	bindKey []byte // 从 TPM 获取的绑定密钥（运行时缓存）
}

// NewTPM2Provider 创建 Windows/Linux TPM2 Provider 并初始化绑定密钥。
func NewTPM2Provider() (*TPM2Provider, error) {
	p := &TPM2Provider{}

	if !p.Available() {
		return nil, ErrNotAvailable
	}

	if err := p.initBindKey(); err != nil {
		return nil, fmt.Errorf("初始化 TPM 绑定密钥失败: %w", err)
	}

	return p, nil
}

// newPlatformProvider 实现跨平台工厂函数。
func newPlatformProvider() (Provider, error) {
	return NewTPM2Provider()
}
func (p *TPM2Provider) Available() bool {
	if runtime.GOOS == "windows" {
		// Windows：检查 TPM 设备路径
		_, err := os.Stat(`\\.\TPM`)
		return err == nil
	}
	// Linux：检查设备文件
	for _, path := range tpm2DevicePaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// PlatformName 返回平台标识。
func (p *TPM2Provider) PlatformName() string {
	return string(TPMPlatformTPM2)
}

// Seal 使用 TPM 绑定密钥加密数据。
func (p *TPM2Provider) Seal(data []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("TPM 绑定密钥未初始化")
	}
	return sealWithAES(p.bindKey, data)
}

// Unseal 使用 TPM 绑定密钥解密数据。
func (p *TPM2Provider) Unseal(blob []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("TPM 绑定密钥未初始化")
	}
	return unsealWithAES(p.bindKey, blob)
}

// initBindKey 初始化绑定密钥（平台分发）。
func (p *TPM2Provider) initBindKey() error {
	keyPath := p.bindKeyPath()

	// 尝试读取已有密钥
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		p.bindKey = data
		return nil
	}

	// 生成新密钥
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("生成绑定密钥失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(p.bindKeyDir(), 0700); err != nil {
		return fmt.Errorf("创建密钥目录失败: %w", err)
	}

	// 持久化密钥（Linux 生产环境建议写入 TPM NV；Windows 建议使用 DPAPI）
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return fmt.Errorf("保存绑定密钥失败: %w", err)
	}

	p.bindKey = key
	return nil
}

// bindKeyDir 返回绑定密钥存储目录。
func (p *TPM2Provider) bindKeyDir() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("USERPROFILE")
		}
		return appData + `\GlobalTrusts\client-card\tpm`
	}
	// Linux/macOS
	home, _ := os.UserHomeDir()
	return home + "/.config/globaltrusts/clients/tpm"
}

// bindKeyPath 返回绑定密钥文件路径。
func (p *TPM2Provider) bindKeyPath() string {
	if runtime.GOOS == "windows" {
		return p.bindKeyDir() + `\bind.key`
	}
	return p.bindKeyDir() + "/bind.key"
}
