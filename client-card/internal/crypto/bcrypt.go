package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost 是 bcrypt 的计算代价，值越大越安全但越慢。
	BcryptCost = 12
)

// HashPassword 使用 bcrypt 对密码进行哈希。
// 用于本地/TPM2 用户密码存储。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt 哈希失败: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword 验证密码是否与 bcrypt 哈希匹配。
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
