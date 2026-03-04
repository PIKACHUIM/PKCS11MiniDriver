package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
)

// HMACSHA256 计算 HMAC-SHA256，返回 32 字节摘要。
// key: 密钥（如用户密码的哈希）
// data: 待计算数据（如随机盐值）
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// DeriveKey 从密码和盐值派生 AES-256 密钥。
// 使用 HMAC-SHA256(password, salt) 作为密钥派生函数。
// 注意：生产环境建议使用 PBKDF2 或 Argon2，此处用于快速派生。
func DeriveKey(password, salt []byte) []byte {
	return HMACSHA256(password, salt)
}
