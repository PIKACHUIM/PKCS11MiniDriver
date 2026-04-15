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
	return EncryptAES256GCMWithAAD(key, plaintext, nil)
}

// EncryptAES256GCMWithAAD 使用 AES-256-GCM 加密数据，带附加认证数据。
// aad 通常为 card_uuid + cert_uuid，防止密文被替换到其他上下文。
func EncryptAES256GCMWithAAD(key, plaintext, aad []byte) ([]byte, error) {
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, aad)
	return ciphertext, nil
}

// DecryptAES256GCM 使用 AES-256-GCM 解密数据。
// 输入格式：nonce(12B) + ciphertext + tag(16B)
func DecryptAES256GCM(key, ciphertext []byte) ([]byte, error) {
	return DecryptAES256GCMWithAAD(key, ciphertext, nil)
}

// DecryptAES256GCMWithAAD 使用 AES-256-GCM 解密数据，带附加认证数据。
func DecryptAES256GCMWithAAD(key, ciphertext, aad []byte) ([]byte, error) {
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

	plaintext, err := gcm.Open(nil, nonce, data, aad)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM 解密失败（密钥或数据错误）: %w", err)
	}

	return plaintext, nil
}

// DecryptWithFallback 尝试使用新算法（Argon2id + AAD）解密，
// 失败后回退到旧的 HMAC-SHA256 方案。
// 成功解密后返回明文和是否需要迁移的标志。
func DecryptWithFallback(password, salt, aad, ciphertext []byte) (plaintext []byte, needsMigration bool, err error) {
	// 先尝试新算法：Argon2id 派生密钥 + AAD
	newKey := DeriveKeyArgon2id(password, salt)
	plaintext, err = DecryptAES256GCMWithAAD(newKey, ciphertext, aad)
	ZeroBytes(newKey)
	if err == nil {
		return plaintext, false, nil
	}

	// 回退到旧算法：HMAC-SHA256 派生密钥，无 AAD
	oldKey := HMACSHA256(password, salt)
	plaintext, err = DecryptAES256GCM(oldKey, ciphertext)
	ZeroBytes(oldKey)
	if err == nil {
		return plaintext, true, nil // 需要迁移到新算法
	}

	return nil, false, fmt.Errorf("新旧算法均解密失败: %w", err)
}

// ZeroBytes 将字节切片清零，用于内存中的密钥清理。
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
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
