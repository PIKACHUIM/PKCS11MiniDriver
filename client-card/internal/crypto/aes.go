// Package crypto 提供应用所需的加密工具函数。
// 包含 AES-256-GCM 加密、HMAC-SHA256 和 bcrypt 密码哈希。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	// KeySize 是 AES-256 密钥长度（字节）。
	KeySize = 32
	// SaltSize 是随机盐值长度（字节）。
	SaltSize = 32
	// NonceSize 是 GCM nonce 长度（字节）。
	NonceSize = 12
)

// EncryptAES256GCM 使用 AES-256-GCM 加密数据。
// 返回格式：nonce(12B) + ciphertext + tag(16B)
func EncryptAES256GCM(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("密钥长度必须为 %d 字节，实际为 %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES256GCM 使用 AES-256-GCM 解密数据。
// 输入格式：nonce(12B) + ciphertext + tag(16B)
func DecryptAES256GCM(key, ciphertext []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("密钥长度必须为 %d 字节，实际为 %d", KeySize, len(key))
	}

	if len(ciphertext) < NonceSize {
		return nil, fmt.Errorf("密文长度不足，无法提取 nonce")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonce := ciphertext[:NonceSize]
	data := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 解密失败（密钥或数据错误）: %w", err)
	}

	return plaintext, nil
}

// GenerateRandomBytes 生成指定长度的随机字节。
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("生成随机字节失败: %w", err)
	}
	return b, nil
}

// GenerateKey 生成 32 字节随机 AES-256 密钥。
func GenerateKey() ([]byte, error) {
	return GenerateRandomBytes(KeySize)
}

// GenerateSalt 生成 32 字节随机盐值。
func GenerateSalt() ([]byte, error) {
	return GenerateRandomBytes(SaltSize)
}
