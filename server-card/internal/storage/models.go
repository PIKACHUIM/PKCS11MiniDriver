// Package storage - 数据模型定义。
package storage

import "time"

// User 是服务端用户模型。
type User struct {
	UUID         string    `json:"uuid"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Card 是云端卡片模型。
type Card struct {
	UUID      string    `json:"uuid"`
	UserUUID  string    `json:"user_uuid"`
	CardName  string    `json:"card_name"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Certificate 是云端证书模型。
// 私钥加密存储在服务端，不对外暴露。
type Certificate struct {
	UUID        string    `json:"uuid"`
	CardUUID    string    `json:"card_uuid"`
	CertType    string    `json:"cert_type"`
	KeyType     string    `json:"key_type"`
	CertContent []byte    `json:"cert_content"` // 公开部分
	PrivateData []byte    `json:"-"`             // 加密的私钥，不序列化
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Log 是操作日志模型。
type Log struct {
	ID         int64     `json:"id"`
	UserUUID   string    `json:"user_uuid"`
	CardUUID   string    `json:"card_uuid"`
	CertUUID   string    `json:"cert_uuid"`
	Action     string    `json:"action"`
	IPAddr     string    `json:"ip_addr"`
	UserAgent  string    `json:"user_agent"`
	RecordedAt time.Time `json:"recorded_at"`
}
