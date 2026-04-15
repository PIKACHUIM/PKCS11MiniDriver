// Package tpm - Mock 实现，用于测试。
package tpm

import (
	"crypto/rand"
	"fmt"
)

// MockProvider 是 TPM 的 Mock 实现，使用内存 AES 密钥模拟 Seal/Unseal。
// 仅用于测试，不提供真实的硬件安全保证。
type MockProvider struct {
	key [32]byte // 固定 AES 密钥，模拟 TPM 内部密钥
}

// NewMock 创建一个 Mock TPM Provider（使用随机密钥）。
func NewMock() *MockProvider {
	p := &MockProvider{}
	if _, err := rand.Read(p.key[:]); err != nil {
		panic(fmt.Sprintf("MockProvider 初始化失败: %v", err))
	}
	return p
}

// NewMockWithKey 创建一个使用固定密钥的 Mock Provider（用于确定性测试）。
func NewMockWithKey(key [32]byte) *MockProvider {
	return &MockProvider{key: key}
}

// Available 始终返回 true。
func (m *MockProvider) Available() bool {
	return true
}

// PlatformName 返回 mock 标识。
func (m *MockProvider) PlatformName() string {
	return "mock"
}

// Seal 使用 AES-256-GCM 加密数据，模拟 TPM Seal。
func (m *MockProvider) Seal(data []byte) ([]byte, error) {
	return sealWithAES(m.key[:], data)
}

// Unseal 解密 Seal 的数据。
func (m *MockProvider) Unseal(blob []byte) ([]byte, error) {
	return unsealWithAES(m.key[:], blob)
}