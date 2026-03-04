//go:build darwin && !cgo

// Package tpm - macOS 纯 Go fallback 实现（CGO 不可用时）。
// 使用文件系统存储绑定密钥（不依赖 Keychain）。
// 注意：此实现安全性低于 CGO 版本，仅用于测试/CI 环境。
package tpm

import (
	"crypto/rand"
	"fmt"
	"os"
)

// DarwinNoCGOProvider 是 macOS 纯 Go 实现（CGO 不可用时的 fallback）。
type DarwinNoCGOProvider struct {
	bindKey []byte
}

// newPlatformProvider 在 CGO 不可用时使用文件系统存储绑定密钥。
func newPlatformProvider() (Provider, error) {
	p := &DarwinNoCGOProvider{}
	if err := p.initBindKey(); err != nil {
		return nil, fmt.Errorf("初始化绑定密钥失败: %w", err)
	}
	return p, nil
}

// Available 始终返回 true。
func (p *DarwinNoCGOProvider) Available() bool { return true }

// PlatformName 返回平台标识。
func (p *DarwinNoCGOProvider) PlatformName() string { return string(TPMPlatformAppleT2) }

// Seal 使用绑定密钥加密数据。
func (p *DarwinNoCGOProvider) Seal(data []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("绑定密钥未初始化")
	}
	return sealWithAES(p.bindKey, data)
}

// Unseal 解密数据。
func (p *DarwinNoCGOProvider) Unseal(blob []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("绑定密钥未初始化")
	}
	return unsealWithAES(p.bindKey, blob)
}

// initBindKey 从文件系统加载或生成绑定密钥。
func (p *DarwinNoCGOProvider) initBindKey() error {
	home, _ := os.UserHomeDir()
	dir := home + "/.config/globaltrusts/client-card/tpm"
	keyPath := dir + "/bind.key"

	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		p.bindKey = data
		return nil
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("生成绑定密钥失败: %w", err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("创建密钥目录失败: %w", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return fmt.Errorf("保存绑定密钥失败: %w", err)
	}
	p.bindKey = key
	return nil
}
