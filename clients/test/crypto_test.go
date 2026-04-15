// Package storage_test 提供 clients 加密工具的单元测试。
package storage_test

import (
	"bytes"
	"testing"

	"github.com/globaltrusts/client-card/internal/crypto"
)

// ---- Argon2id 密钥派生测试 ----

func TestDeriveKeyArgon2id(t *testing.T) {
	t.Parallel()

	password := []byte("test-password-123")
	salt := []byte("test-salt-32bytes-padding-here!!")

	key := crypto.DeriveKeyArgon2id(password, salt)

	if len(key) != crypto.KeySize {
		t.Errorf("派生密钥长度 = %d，期望 %d", len(key), crypto.KeySize)
	}

	// 相同输入应产生相同密钥（确定性）
	key2 := crypto.DeriveKeyArgon2id(password, salt)
	if !bytes.Equal(key, key2) {
		t.Error("相同输入应产生相同密钥")
	}

	// 不同盐值应产生不同密钥
	salt2 := []byte("different-salt-32bytes-padding!!!")
	key3 := crypto.DeriveKeyArgon2id(password, salt2)
	if bytes.Equal(key, key3) {
		t.Error("不同盐值应产生不同密钥")
	}

	// 不同密码应产生不同密钥
	key4 := crypto.DeriveKeyArgon2id([]byte("other-password"), salt)
	if bytes.Equal(key, key4) {
		t.Error("不同密码应产生不同密钥")
	}
}

func TestDeriveKeyWithAAD(t *testing.T) {
	t.Parallel()

	password := []byte("password")
	salt := []byte("salt-32bytes-padding-here-test!!")
	aad1 := []byte("card-uuid-1:cert-uuid-1")
	aad2 := []byte("card-uuid-2:cert-uuid-2")

	key1 := crypto.DeriveKeyWithAAD(password, salt, aad1)
	key2 := crypto.DeriveKeyWithAAD(password, salt, aad2)

	if len(key1) != crypto.KeySize {
		t.Errorf("派生密钥长度 = %d，期望 %d", len(key1), crypto.KeySize)
	}

	// 不同 AAD 应产生不同密钥
	if bytes.Equal(key1, key2) {
		t.Error("不同 AAD 应产生不同密钥")
	}

	// 空 AAD 与有 AAD 应产生不同密钥
	keyNoAAD := crypto.DeriveKeyWithAAD(password, salt, nil)
	if bytes.Equal(key1, keyNoAAD) {
		t.Error("有 AAD 与无 AAD 应产生不同密钥")
	}
}

// ---- AES-256-GCM 加解密测试 ----

func TestEncryptDecryptAES256GCM(t *testing.T) {
	t.Parallel()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() 失败: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"空明文", []byte{}},
		{"短明文", []byte("hello")},
		{"标准明文", []byte("这是一段测试明文，包含中文字符")},
		{"长明文", bytes.Repeat([]byte("abcdefgh"), 1024)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ciphertext, err := crypto.EncryptAES256GCM(key, tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptAES256GCM() 失败: %v", err)
			}

			// 密文长度应大于明文（nonce + tag）
			if len(tt.plaintext) > 0 && len(ciphertext) <= len(tt.plaintext) {
				t.Error("密文长度应大于明文长度")
			}

			// 解密应还原明文
			got, err := crypto.DecryptAES256GCM(key, ciphertext)
			if err != nil {
				t.Fatalf("DecryptAES256GCM() 失败: %v", err)
			}
			if !bytes.Equal(got, tt.plaintext) {
				t.Errorf("解密结果与原文不匹配")
			}
		})
	}
}

func TestEncryptDecryptAES256GCMWithAAD(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()
	plaintext := []byte("敏感私钥数据")
	aad := []byte("card-uuid:cert-uuid")

	// 加密（带 AAD）
	ciphertext, err := crypto.EncryptAES256GCMWithAAD(key, plaintext, aad)
	if err != nil {
		t.Fatalf("EncryptAES256GCMWithAAD() 失败: %v", err)
	}

	// 使用正确 AAD 解密应成功
	got, err := crypto.DecryptAES256GCMWithAAD(key, ciphertext, aad)
	if err != nil {
		t.Fatalf("DecryptAES256GCMWithAAD() 失败: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Error("解密结果与原文不匹配")
	}

	// 使用错误 AAD 解密应失败（防止密文被移植到其他上下文）
	_, err = crypto.DecryptAES256GCMWithAAD(key, ciphertext, []byte("wrong-aad"))
	if err == nil {
		t.Error("错误 AAD 解密应失败")
	}

	// 使用空 AAD 解密应失败
	_, err = crypto.DecryptAES256GCMWithAAD(key, ciphertext, nil)
	if err == nil {
		t.Error("空 AAD 解密应失败")
	}
}

func TestDecryptAES256GCMWrongKey(t *testing.T) {
	t.Parallel()

	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()
	plaintext := []byte("secret data")

	ciphertext, _ := crypto.EncryptAES256GCM(key1, plaintext)

	// 使用错误密钥解密应失败
	_, err := crypto.DecryptAES256GCM(key2, ciphertext)
	if err == nil {
		t.Error("错误密钥解密应失败")
	}
}

func TestDecryptAES256GCMInvalidInput(t *testing.T) {
	t.Parallel()

	key, _ := crypto.GenerateKey()

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"空密文", []byte{}},
		{"过短密文（小于 nonce 长度）", []byte("short")},
		{"随机字节", []byte("random-garbage-data-that-is-not-valid-ciphertext")},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := crypto.DecryptAES256GCM(key, tt.ciphertext)
			if err == nil {
				t.Error("无效密文解密应失败")
			}
		})
	}
}

func TestEncryptAES256GCMWrongKeySize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		keySize int
	}{
		{"16 字节密钥（AES-128）", 16},
		{"24 字节密钥（AES-192）", 24},
		{"空密钥", 0},
	}

	plaintext := []byte("test")
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			key := make([]byte, tt.keySize)
			_, err := crypto.EncryptAES256GCM(key, plaintext)
			if err == nil {
				t.Errorf("密钥长度 %d 应返回错误", tt.keySize)
			}
		})
	}
}

// ---- 向后兼容回退逻辑测试 ----

func TestDecryptWithFallback(t *testing.T) {
	t.Parallel()

	password := []byte("user-password")
	salt, _ := crypto.GenerateSalt()
	aad := []byte("card-uuid:cert-uuid")
	plaintext := []byte("private key data")

	t.Run("新算法加密_新算法解密_无需迁移", func(t *testing.T) {
		t.Parallel()
		// 用新算法（Argon2id + AAD）加密
		newKey := crypto.DeriveKeyArgon2id(password, salt)
		ciphertext, err := crypto.EncryptAES256GCMWithAAD(newKey, plaintext, aad)
		crypto.ZeroBytes(newKey)
		if err != nil {
			t.Fatalf("加密失败: %v", err)
		}

		got, needsMigration, err := crypto.DecryptWithFallback(password, salt, aad, ciphertext)
		if err != nil {
			t.Fatalf("DecryptWithFallback() 失败: %v", err)
		}
		if needsMigration {
			t.Error("新算法加密不应需要迁移")
		}
		if !bytes.Equal(got, plaintext) {
			t.Error("解密结果与原文不匹配")
		}
	})

	t.Run("旧算法加密_回退解密_需要迁移", func(t *testing.T) {
		t.Parallel()
		// 用旧算法（HMAC-SHA256，无 AAD）加密
		oldKey := crypto.HMACSHA256(password, salt)
		ciphertext, err := crypto.EncryptAES256GCM(oldKey, plaintext)
		crypto.ZeroBytes(oldKey)
		if err != nil {
			t.Fatalf("旧算法加密失败: %v", err)
		}

		got, needsMigration, err := crypto.DecryptWithFallback(password, salt, aad, ciphertext)
		if err != nil {
			t.Fatalf("DecryptWithFallback() 回退失败: %v", err)
		}
		if !needsMigration {
			t.Error("旧算法加密应需要迁移")
		}
		if !bytes.Equal(got, plaintext) {
			t.Error("回退解密结果与原文不匹配")
		}
	})

	t.Run("错误密码_两种算法均失败", func(t *testing.T) {
		t.Parallel()
		newKey := crypto.DeriveKeyArgon2id(password, salt)
		ciphertext, _ := crypto.EncryptAES256GCMWithAAD(newKey, plaintext, aad)
		crypto.ZeroBytes(newKey)

		_, _, err := crypto.DecryptWithFallback([]byte("wrong-password"), salt, aad, ciphertext)
		if err == nil {
			t.Error("错误密码应返回错误")
		}
	})
}

// ---- 内存清零测试 ----

func TestZeroBytes(t *testing.T) {
	t.Parallel()

	key := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	crypto.ZeroBytes(key)

	for i, b := range key {
		if b != 0 {
			t.Errorf("ZeroBytes() 后 key[%d] = %d，期望 0", i, b)
		}
	}
}

func TestZeroBytesEmpty(t *testing.T) {
	t.Parallel()
	// 空切片不应 panic
	crypto.ZeroBytes(nil)
	crypto.ZeroBytes([]byte{})
}

// ---- 随机字节生成测试 ----

func TestGenerateRandomBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		n    int
	}{
		{"生成 16 字节", 16},
		{"生成 32 字节", 32},
		{"生成 64 字节", 64},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, err := crypto.GenerateRandomBytes(tt.n)
			if err != nil {
				t.Fatalf("GenerateRandomBytes(%d) 失败: %v", tt.n, err)
			}
			if len(b) != tt.n {
				t.Errorf("长度 = %d，期望 %d", len(b), tt.n)
			}
		})
	}

	// 两次生成应不同（随机性验证）
	b1, _ := crypto.GenerateRandomBytes(32)
	b2, _ := crypto.GenerateRandomBytes(32)
	if bytes.Equal(b1, b2) {
		t.Error("两次随机生成不应相同")
	}
}

func TestGenerateKey(t *testing.T) {
	t.Parallel()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() 失败: %v", err)
	}
	if len(key) != crypto.KeySize {
		t.Errorf("密钥长度 = %d，期望 %d", len(key), crypto.KeySize)
	}
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() 失败: %v", err)
	}
	if len(salt) != crypto.SaltSize {
		t.Errorf("盐值长度 = %d，期望 %d", len(salt), crypto.SaltSize)
	}
}
