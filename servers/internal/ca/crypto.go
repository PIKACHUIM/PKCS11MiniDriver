// Package ca - 加密工具函数。
package ca

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
)

// parseCertPEM 解析 PEM 格式的证书（取第一个证书）。
func parseCertPEM(certPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("无效的 PEM 证书")
	}
	return x509.ParseCertificate(block.Bytes)
}

// decryptPrivateKey 使用主密钥 AES-256-GCM 解密私钥，返回 crypto.Signer。
func decryptPrivateKey(masterKey, encData []byte) (crypto.Signer, error) {
	privDER, err := aesGCMDecrypt(masterKey, encData)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(privDER)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("私钥不支持签名操作")
	}
	return signer, nil
}

// encryptPrivateKey 使用主密钥 AES-256-GCM 加密私钥 DER。
func encryptPrivateKey(masterKey, privDER []byte) ([]byte, error) {
	return aesGCMEncrypt(masterKey, privDER)
}

// aesGCMEncrypt 使用 AES-256-GCM 加密数据。
func aesGCMEncrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

// aesGCMDecrypt 使用 AES-256-GCM 解密数据。
func aesGCMDecrypt(key, blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, fmt.Errorf("密文太短")
	}

	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
