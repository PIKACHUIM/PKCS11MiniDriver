package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost 是 bcrypt 的计算代价，值越大越安全但越慢。
	// 从 12 升级到 13，符合当前安全标准。
	BcryptCost = 13
	// BcryptOldCost 是旧版本使用的 cost 值，用于检测是否需要迁移。
	BcryptOldCost = 12
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

// VerifyAndUpgrade 验证密码并检查是否需要升级哈希。
// 如果密码正确且哈希使用旧 cost，返回新的哈希值。
// 返回值：(验证通过, 新哈希（需要升级时非空）, 错误)
func VerifyAndUpgrade(password, hash string) (bool, string, error) {
	if !VerifyPassword(password, hash) {
		return false, "", nil
	}

	// 检查当前哈希的 cost
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true, "", nil // 验证通过但无法检测 cost，不升级
	}

	// 如果 cost 低于当前标准，生成新哈希
	if cost < BcryptCost {
		newHash, err := HashPassword(password)
		if err != nil {
			return true, "", fmt.Errorf("升级哈希失败: %w", err)
		}
		return true, newHash, nil
	}

	return true, "", nil
}
