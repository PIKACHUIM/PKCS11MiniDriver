// Package storage 提供 server-card 的数据存储层。
// 使用 SQLite（开发/单机）或 PostgreSQL（生产）。
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB 封装数据库连接。
type DB struct {
	*sql.DB
}

// Open 打开数据库并执行迁移。
func Open(path string) (*DB, error) {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	wrapped := &DB{db}
	if err := wrapped.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return wrapped, nil
}

// migrate 执行数据库 Schema 迁移。
func (db *DB) migrate() error {
	schema := `
-- 用户表
CREATE TABLE IF NOT EXISTS users (
    uuid         TEXT PRIMARY KEY,
    username     TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    email        TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    enabled      INTEGER NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 云端卡片表
CREATE TABLE IF NOT EXISTS cards (
    uuid       TEXT PRIMARY KEY,
    user_uuid  TEXT NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    card_name  TEXT NOT NULL,
    remark     TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 云端证书表（私钥加密存储，不离开服务器）
CREATE TABLE IF NOT EXISTS certificates (
    uuid          TEXT PRIMARY KEY,
    card_uuid     TEXT NOT NULL REFERENCES cards(uuid) ON DELETE CASCADE,
    cert_type     TEXT NOT NULL DEFAULT 'x509',
    key_type      TEXT NOT NULL DEFAULT 'ec256',
    cert_content  BLOB,              -- 公开部分（X.509 DER / 公钥）
    private_data  BLOB,              -- 加密的私钥（服务端主密钥加密）
    remark        TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 操作日志表
CREATE TABLE IF NOT EXISTS logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_uuid   TEXT NOT NULL DEFAULT '',
    card_uuid   TEXT NOT NULL DEFAULT '',
    cert_uuid   TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,       -- login/sign/decrypt/create_card/...
    ip_addr     TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cards_user_uuid ON cards(user_uuid);
CREATE INDEX IF NOT EXISTS idx_certs_card_uuid ON certificates(card_uuid);
CREATE INDEX IF NOT EXISTS idx_logs_user_uuid  ON logs(user_uuid);
CREATE INDEX IF NOT EXISTS idx_logs_recorded_at ON logs(recorded_at);
`
	_, err := db.Exec(schema)
	return err
}
