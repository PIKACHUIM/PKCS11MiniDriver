//go:build !windows && !linux && !darwin

// Package tpm - 其他平台的 fallback 实现。
// 在不支持 TPM 的平台上返回 ErrNotAvailable。
package tpm

// newPlatformProvider 在不支持的平台上返回 ErrNotAvailable。
func newPlatformProvider() (Provider, error) {
	return nil, ErrNotAvailable
}
