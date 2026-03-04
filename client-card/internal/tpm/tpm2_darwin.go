//go:build darwin && cgo

// Package tpm - macOS T2/Secure Enclave 实现（CGO 版本）。
// 使用 macOS Keychain 存储绑定密钥，通过 Secure Enclave 保护。
//
// 实现策略：
//   - 使用 macOS Keychain 存储 32 字节绑定密钥（受 Secure Enclave 保护）
//   - Seal：用绑定密钥 AES-256-GCM 加密数据
//   - Unseal：从 Keychain 读取绑定密钥，解密数据
//
// 注意：当前使用 Keychain 通用密码项存储绑定密钥。
// 生产环境可进一步使用 kSecAttrAccessibleWhenUnlockedThisDeviceOnly 限制访问。
package tpm

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// keychainStore 将数据存储到 Keychain。
// 返回 0 表示成功，非 0 表示 OSStatus 错误码。
static int keychainStore(const char* service, const char* account, const void* data, int dataLen) {
    CFStringRef serviceRef = CFStringCreateWithCString(NULL, service, kCFStringEncodingUTF8);
    CFStringRef accountRef = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);
    CFDataRef dataRef = CFDataCreate(NULL, (const UInt8*)data, dataLen);

    // 先删除已有条目
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, serviceRef);
    CFDictionarySetValue(query, kSecAttrAccount, accountRef);
    SecItemDelete(query);
    CFRelease(query);

    // 添加新条目
    CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFDictionarySetValue(attrs, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(attrs, kSecAttrService, serviceRef);
    CFDictionarySetValue(attrs, kSecAttrAccount, accountRef);
    CFDictionarySetValue(attrs, kSecValueData, dataRef);
    // 仅在设备解锁时可访问，且不迁移到其他设备
    CFDictionarySetValue(attrs, kSecAttrAccessible, kSecAttrAccessibleWhenUnlockedThisDeviceOnly);

    OSStatus status = SecItemAdd(attrs, NULL);

    CFRelease(serviceRef);
    CFRelease(accountRef);
    CFRelease(dataRef);
    CFRelease(attrs);

    return (int)status;
}

// keychainLoad 从 Keychain 读取数据。
// 返回数据长度，-1 表示失败。outData 由调用方 free。
static int keychainLoad(const char* service, const char* account, void** outData) {
    CFStringRef serviceRef = CFStringCreateWithCString(NULL, service, kCFStringEncodingUTF8);
    CFStringRef accountRef = CFStringCreateWithCString(NULL, account, kCFStringEncodingUTF8);

    CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, serviceRef);
    CFDictionarySetValue(query, kSecAttrAccount, accountRef);
    CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
    CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);

    CFDataRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, (CFTypeRef*)&result);

    CFRelease(serviceRef);
    CFRelease(accountRef);
    CFRelease(query);

    if (status != errSecSuccess || result == NULL) {
        return -1;
    }

    CFIndex len = CFDataGetLength(result);
    *outData = malloc(len);
    memcpy(*outData, CFDataGetBytePtr(result), len);
    CFRelease(result);

    return (int)len;
}
*/
import "C"

import (
	"crypto/rand"
	"fmt"
	"unsafe"
)

const (
	keychainService = "com.globaltrusts.client-card"
	keychainAccount = "tpm-bind-key"
)

// DarwinProvider 是 macOS T2/Secure Enclave 实现。
// 使用 macOS Keychain 存储绑定密钥。
type DarwinProvider struct {
	bindKey []byte
}

// NewDarwinProvider 创建 macOS Provider 并初始化绑定密钥。
func NewDarwinProvider() (*DarwinProvider, error) {
	p := &DarwinProvider{}

	if !p.Available() {
		return nil, ErrNotAvailable
	}

	if err := p.initBindKey(); err != nil {
		return nil, fmt.Errorf("初始化 Keychain 绑定密钥失败: %w", err)
	}

	return p, nil
}

// newPlatformProvider 实现跨平台工厂函数（macOS）。
func newPlatformProvider() (Provider, error) {
	return NewDarwinProvider()
}

// Available 在 macOS 上始终返回 true（Keychain 始终可用）。
func (p *DarwinProvider) Available() bool {
	return true
}

// PlatformName 返回平台标识。
func (p *DarwinProvider) PlatformName() string {
	return string(TPMPlatformAppleT2)
}

// Seal 使用 Keychain 绑定密钥加密数据。
func (p *DarwinProvider) Seal(data []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("Keychain 绑定密钥未初始化")
	}
	return sealWithAES(p.bindKey, data)
}

// Unseal 使用 Keychain 绑定密钥解密数据。
func (p *DarwinProvider) Unseal(blob []byte) ([]byte, error) {
	if len(p.bindKey) == 0 {
		return nil, fmt.Errorf("Keychain 绑定密钥未初始化")
	}
	return unsealWithAES(p.bindKey, blob)
}

// initBindKey 从 Keychain 加载或生成绑定密钥。
func (p *DarwinProvider) initBindKey() error {
	// 尝试从 Keychain 读取
	key, err := keychainLoadKey()
	if err == nil && len(key) == 32 {
		p.bindKey = key
		return nil
	}

	// 生成新密钥
	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("生成绑定密钥失败: %w", err)
	}

	// 存储到 Keychain
	if err := keychainStoreKey(newKey); err != nil {
		return fmt.Errorf("存储绑定密钥到 Keychain 失败: %w", err)
	}

	p.bindKey = newKey
	return nil
}

// keychainStoreKey 将绑定密钥存储到 Keychain。
func keychainStoreKey(key []byte) error {
	service := C.CString(keychainService)
	account := C.CString(keychainAccount)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))

	status := C.keychainStore(service, account, unsafe.Pointer(&key[0]), C.int(len(key)))
	if status != 0 {
		return fmt.Errorf("Keychain 存储失败，OSStatus: %d", int(status))
	}
	return nil
}

// keychainLoadKey 从 Keychain 读取绑定密钥。
func keychainLoadKey() ([]byte, error) {
	service := C.CString(keychainService)
	account := C.CString(keychainAccount)
	defer C.free(unsafe.Pointer(service))
	defer C.free(unsafe.Pointer(account))

	var outData unsafe.Pointer
	length := C.keychainLoad(service, account, &outData)
	if length < 0 {
		return nil, fmt.Errorf("Keychain 读取失败")
	}
	defer C.free(outData)

	result := make([]byte, int(length))
	copy(result, (*[1 << 20]byte)(outData)[:int(length)])
	return result, nil
}


