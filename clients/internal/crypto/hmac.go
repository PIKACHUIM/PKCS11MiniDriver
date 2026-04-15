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
// 使用 Argon2id 作为密钥派生函数（推荐）。
func DeriveKey(password, salt []byte) []byte {
	return DeriveKeyArgon2id(password, salt)
}

// DeriveKeyLegacy 使用旧的 HMAC-SHA256 派生密钥（仅用于向后兼容）。
func DeriveKeyLegacy(password, salt []byte) []byte {
	return HMACSHA256(password, salt)
}
