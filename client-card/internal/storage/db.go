// Package storage 提供 SQLite 数据库访问层。
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 数据库连接。
type DB struct {
	conn *sql.DB
}

// Open 打开（或创建）SQLite 数据库。
func Open(path string) (*DB, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	// 启用 WAL 模式和外键约束
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(30 * time.Minute)

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return db, nil
}

// Close 关闭数据库连接。
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn 返回底层 *sql.DB，供 Repository 使用。
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// migrate 执行数据库表结构初始化。
func (db *DB) migrate() error {
	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("执行 schema 失败: %w", err)
	}
	return nil
}

// schema 是数据库建表 SQL。
const schema = `
-- 用户管理表
CREATE TABLE IF NOT EXISTS users (
    uuid            TEXT PRIMARY KEY,
    user_type       TEXT NOT NULL DEFAULT 'local',  -- local / cloud
    display_name    TEXT NOT NULL,
    email           TEXT NOT NULL DEFAULT '',
    enabled         INTEGER NOT NULL DEFAULT 1,
    cloud_url       TEXT NOT NULL DEFAULT '',
    password_hash   TEXT NOT NULL DEFAULT '',       -- bcrypt，云端留空
    auth_token      BLOB,                           -- 加密存储的 WebToken 或 PIN 加密的密码
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 卡片管理表
CREATE TABLE IF NOT EXISTS cards (
    uuid            TEXT PRIMARY KEY,
    slot_type       TEXT NOT NULL,                  -- local / tpm2 / cloud
    card_name       TEXT NOT NULL,
    user_uuid       TEXT NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    expires_at      DATETIME,
    card_keys       BLOB NOT NULL,                  -- JSON 数组，存储多个加密的主密钥记录
    remark          TEXT NOT NULL DEFAULT '',
    cloud_url       TEXT NOT NULL DEFAULT '',       -- Cloud Slot: server-card 服务地址
    cloud_card_uuid TEXT NOT NULL DEFAULT '',       -- Cloud Slot: 在 server-card 中的卡片 UUID
    FOREIGN KEY (user_uuid) REFERENCES users(uuid) ON DELETE CASCADE
);

-- 证书/密钥表（本地和 TPM2 卡片使用）
CREATE TABLE IF NOT EXISTS certificates (
    uuid            TEXT PRIMARY KEY,
    slot_type       TEXT NOT NULL,                  -- local / tpm2 / cloud(缓存)
    card_uuid       TEXT NOT NULL,
    cert_type       TEXT NOT NULL,                  -- x509/ssh/gpg/totp/fido/login/text/note/payment
    key_type        TEXT NOT NULL DEFAULT '',       -- rsa2048/ec256/ed25519/...
    cert_content    BLOB,                           -- 公开部分（X509/SSH/GPG 公钥）
    temp_key_salt   BLOB,                           -- 32 字节随机盐值
    temp_key_enc    BLOB,                           -- AES256 加密的临时密钥
    private_data    BLOB,                           -- 临时密钥加密的私钥/私密数据
    -- TPM2 专用字段
    tpm_platform    TEXT NOT NULL DEFAULT '',       -- tpm2 / apple_t2 / apple_se
    tpm_key_handle  INTEGER,                        -- TPM 持久化句柄
    tpm_public_blob BLOB,                           -- TPM 公钥 Blob
    tpm_private_blob BLOB,                          -- TPM 私钥 Blob
    tpm_pcr_policy  BLOB,                           -- PCR 策略（可选）
    tpm_auth_policy BLOB,                           -- 授权策略哈希
    remark          TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at      DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (card_uuid) REFERENCES cards(uuid) ON DELETE CASCADE
);

-- 日志管理表
CREATE TABLE IF NOT EXISTS logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    log_type        TEXT NOT NULL DEFAULT 'operation', -- operation / security / error
    slot_type       TEXT NOT NULL DEFAULT '',
    card_uuid       TEXT NOT NULL DEFAULT '',
    user_uuid       TEXT NOT NULL DEFAULT '',
    log_level       TEXT NOT NULL DEFAULT 'info',      -- debug / info / warn / error
    recorded_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    title           TEXT NOT NULL,
    content         TEXT NOT NULL DEFAULT ''
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_cards_user_uuid ON cards(user_uuid);
CREATE INDEX IF NOT EXISTS idx_certs_card_uuid ON certificates(card_uuid);
CREATE INDEX IF NOT EXISTS idx_logs_recorded_at ON logs(recorded_at);
CREATE INDEX IF NOT EXISTS idx_logs_card_uuid ON logs(card_uuid);
`
