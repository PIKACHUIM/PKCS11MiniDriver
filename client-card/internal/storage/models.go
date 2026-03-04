package storage

import (
	"time"
)

// ---- 用户模型 ----

// UserType 是用户类型。
type UserType string

const (
	UserTypeLocal UserType = "local"
	UserTypeCloud UserType = "cloud"
)

// User 是用户数据模型。
type User struct {
	UUID        string    `json:"uuid"`
	UserType    UserType  `json:"user_type"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Enabled     bool      `json:"enabled"`
	CloudURL    string    `json:"cloud_url"`
	PasswordHash string   `json:"-"`           // bcrypt 哈希，不序列化到 JSON
	AuthToken   []byte    `json:"-"`           // 加密存储，不序列化到 JSON
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ---- 卡片模型 ----

// SlotType 是卡片 Slot 类型。
type SlotType string

const (
	SlotTypeLocal SlotType = "local"
	SlotTypeTPM2  SlotType = "tpm2"
	SlotTypeCloud SlotType = "cloud"
)

// CardKeyEntry 是卡片主密钥的一条加密记录。
// 卡片密码以列表形式存储，支持多用户权限。
type CardKeyEntry struct {
	// KeyType 区分是用户密码加密还是卡片密码加密。
	// "user" = HMAC(用户密码, salt) 加密
	// "card" = HMAC(设定密码, salt) 加密
	KeyType    string `json:"key_type"`
	UserUUID   string `json:"user_uuid,omitempty"` // KeyType=user 时有效
	Salt       []byte `json:"salt"`                // 32 字节随机盐值
	EncMasterKey []byte `json:"enc_master_key"`    // AES256 加密的主密钥
}

// Card 是卡片数据模型。
type Card struct {
	UUID      string         `json:"uuid"`
	SlotType  SlotType       `json:"slot_type"`
	CardName  string         `json:"card_name"`
	UserUUID  string         `json:"user_uuid"`
	CreatedAt time.Time      `json:"created_at"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	CardKeys  []CardKeyEntry `json:"card_keys"` // 存储为 JSON BLOB
	Remark    string         `json:"remark"`
	// Cloud Slot 专用字段
	CloudURL      string `json:"cloud_url,omitempty"`       // server-card 服务地址，如 http://localhost:1027
	CloudCardUUID string `json:"cloud_card_uuid,omitempty"` // 在 server-card 中的卡片 UUID
}

// ---- 证书模型 ----

// CertType 是证书/密钥类型。
type CertType string

const (
	CertTypeX509    CertType = "x509"
	CertTypeSSH     CertType = "ssh"
	CertTypeGPG     CertType = "gpg"
	CertTypeTOTP    CertType = "totp"
	CertTypeFIDO    CertType = "fido"
	CertTypeLogin   CertType = "login"
	CertTypeText    CertType = "text"
	CertTypeNote    CertType = "note"
	CertTypePayment CertType = "payment"
)

// TPMPlatform 是 TPM 平台类型。
type TPMPlatform string

const (
	TPMPlatformNone    TPMPlatform = ""
	TPMPlatformTPM2    TPMPlatform = "tpm2"
	TPMPlatformAppleT2 TPMPlatform = "apple_t2"
	TPMPlatformAppleSE TPMPlatform = "apple_se"
)

// Certificate 是证书/密钥数据模型。
type Certificate struct {
	UUID        string      `json:"uuid"`
	SlotType    SlotType    `json:"slot_type"`
	CardUUID    string      `json:"card_uuid"`
	CertType    CertType    `json:"cert_type"`
	KeyType     string      `json:"key_type"`     // rsa2048/ec256/ed25519/...
	CertContent []byte      `json:"cert_content"` // 公开部分
	TempKeySalt []byte      `json:"-"`            // 32 字节随机盐值
	TempKeyEnc  []byte      `json:"-"`            // 加密的临时密钥
	PrivateData []byte      `json:"-"`            // 加密的私钥/私密数据
	// TPM2 专用
	TPMPlatform    TPMPlatform `json:"tpm_platform,omitempty"`
	TPMKeyHandle   *int64      `json:"-"`
	TPMPublicBlob  []byte      `json:"-"`
	TPMPrivateBlob []byte      `json:"-"`
	TPMPCRPolicy   []byte      `json:"-"`
	TPMAuthPolicy  []byte      `json:"-"`
	Remark      string      `json:"remark"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ---- 日志模型 ----

// LogType 是日志类型。
type LogType string

const (
	LogTypeOperation LogType = "operation"
	LogTypeSecurity  LogType = "security"
	LogTypeError     LogType = "error"
)

// LogLevel 是日志等级。
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Log 是日志数据模型。
type Log struct {
	ID         int64     `json:"id"`
	LogType    LogType   `json:"log_type"`
	SlotType   SlotType  `json:"slot_type"`
	CardUUID   string    `json:"card_uuid"`
	UserUUID   string    `json:"user_uuid"`
	LogLevel   LogLevel  `json:"log_level"`
	RecordedAt time.Time `json:"recorded_at"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
}
