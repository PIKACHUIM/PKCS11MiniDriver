// Package tpm 提供 TPM2/Secure Enclave 的抽象接口。
// 支持 Windows/Linux TPM2 和 macOS T2/Secure Enclave。
package tpm

import "fmt"

// Platform 是 TPM 平台类型标识符。
type Platform string

const (
	TPMPlatformNone    Platform = ""
	TPMPlatformTPM2    Platform = "tpm2"
	TPMPlatformAppleT2 Platform = "apple_t2"
	TPMPlatformAppleSE Platform = "apple_se"
	TPMPlatformMock    Platform = "mock"
)

// Provider 是 TPM/Secure Enclave 的抽象接口。
// 不同平台实现此接口，对上层屏蔽平台差异。
type Provider interface {
	// Available 检查当前平台是否有可用的 TPM/SE 设备。
	Available() bool

	// Seal 将数据封装（加密）到 TPM/SE 中。
	// 返回的 blob 可持久化存储，只有同一 TPM 才能解封。
	Seal(data []byte) (blob []byte, err error)

	// Unseal 解封（解密）之前 Seal 的数据。
	Unseal(blob []byte) (data []byte, err error)

	// PlatformName 返回平台标识符（用于存储到 tpm_platform 字段）。
	PlatformName() string
}

// ErrNotAvailable 表示当前平台没有可用的 TPM/SE 设备。
var ErrNotAvailable = fmt.Errorf("TPM/Secure Enclave 不可用")

// ErrUnsealFailed 表示解封失败（数据被篡改或 TPM 不匹配）。
var ErrUnsealFailed = fmt.Errorf("TPM 解封失败")

// NewProvider 创建当前平台的 TPM Provider。
// 各平台在对应的 build tag 文件中实现 newPlatformProvider()。
// 如果平台不支持 TPM，返回 ErrNotAvailable。
func NewProvider() (Provider, error) {
	return newPlatformProvider()
}
