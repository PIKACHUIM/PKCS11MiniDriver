// Package crypto 提供应用所需的加密工具函数。
package crypto

import (
	"golang.org/x/crypto/argon2"
)

// Argon2id 参数配置。
const (
	Argon2Time    = 3         // 迭代次数
	Argon2Memory  = 64 * 1024 // 内存使用量（KB），64MB
	Argon2Threads = 4         // 并行线程数
	Argon2KeyLen  = 32        // 输出密钥长度（字节）
)

// DeriveKeyArgon2id 使用 Argon2id 从密码和盐值派生 AES-256 密钥。
// 这是推荐的密钥派生方式，抗暴力破解能力远强于 HMAC-SHA256。
func DeriveKeyArgon2id(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)
}

// DeriveKeyWithAAD 使用 Argon2id 派生密钥，并将 AAD 信息混入盐值。
// aad 通常为 card_uuid + cert_uuid，用于绑定密钥到特定上下文。
func DeriveKeyWithAAD(password, salt, aad []byte) []byte {
	// 将 AAD 追加到盐值中，确保不同上下文产生不同密钥
	combinedSalt := make([]byte, len(salt)+len(aad))
	copy(combinedSalt, salt)
	copy(combinedSalt[len(salt):], aad)
	return argon2.IDKey(password, combinedSalt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)
}
