// Package storage 提供 servers 的数据存储层。
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
    uuid            TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL DEFAULT '',
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'user',
    public_key      BLOB,
    totp_secret     TEXT NOT NULL DEFAULT '',
    enabled         INTEGER NOT NULL DEFAULT 1,
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
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
    uuid               TEXT PRIMARY KEY,
    card_uuid          TEXT NOT NULL REFERENCES cards(uuid) ON DELETE CASCADE,
    cert_type          TEXT NOT NULL DEFAULT 'x509',
    key_type           TEXT NOT NULL DEFAULT 'ec256',
    cert_content       BLOB,              -- 公开部分（X.509 DER / 公钥）
    private_data       BLOB,              -- 加密的私钥（服务端主密钥加密）
    remark             TEXT NOT NULL DEFAULT '',
    order_no           TEXT NOT NULL DEFAULT '',
    ca_uuid            TEXT NOT NULL DEFAULT '',
    issuance_tmpl_uuid TEXT NOT NULL DEFAULT '',
    storage_policy     TEXT NOT NULL DEFAULT '',
    revocation_status  TEXT NOT NULL DEFAULT 'active',
    revoked_at         DATETIME,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_certs_revocation ON certificates(revocation_status);
CREATE INDEX IF NOT EXISTS idx_certs_ca ON certificates(ca_uuid);
CREATE INDEX IF NOT EXISTS idx_certs_order ON certificates(order_no);

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

-- 支付插件配置表
CREATE TABLE IF NOT EXISTS payment_plugins (
    uuid        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    plugin_type TEXT NOT NULL,
    config_enc  BLOB,
    enabled     INTEGER NOT NULL DEFAULT 1,
    sort_weight INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 充值订单表
CREATE TABLE IF NOT EXISTS recharge_orders (
    order_no     TEXT PRIMARY KEY,
    user_uuid    TEXT NOT NULL REFERENCES users(uuid),
    amount_cents INTEGER NOT NULL,
    channel      TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    callback_data BLOB,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    paid_at      DATETIME,
    expires_at   DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_recharge_orders_user ON recharge_orders(user_uuid);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_status ON recharge_orders(status);

-- 用户余额表
CREATE TABLE IF NOT EXISTS user_balances (
    user_uuid       TEXT PRIMARY KEY REFERENCES users(uuid),
    available_cents  INTEGER NOT NULL DEFAULT 0,
    frozen_cents     INTEGER NOT NULL DEFAULT 0,
    total_recharge   INTEGER NOT NULL DEFAULT 0,
    total_consume    INTEGER NOT NULL DEFAULT 0
);

-- 消费记录表
CREATE TABLE IF NOT EXISTS consume_records (
    uuid         TEXT PRIMARY KEY,
    user_uuid    TEXT NOT NULL REFERENCES users(uuid),
    order_no     TEXT,
    consume_type TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    remark       TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_consume_records_user ON consume_records(user_uuid);

-- 退款工单表
CREATE TABLE IF NOT EXISTS refund_requests (
    uuid         TEXT PRIMARY KEY,
    user_uuid    TEXT NOT NULL REFERENCES users(uuid),
    order_no     TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    reason       TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    approved_by  TEXT,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at DATETIME
);

-- 密钥存储类型模板表
CREATE TABLE IF NOT EXISTS key_storage_templates (
    uuid              TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    storage_methods   INTEGER NOT NULL DEFAULT 0,
    security_level    TEXT NOT NULL DEFAULT '',
    allow_reimport    INTEGER NOT NULL DEFAULT 0,
    cloud_backup      INTEGER NOT NULL DEFAULT 0,
    allow_reissue     INTEGER NOT NULL DEFAULT 0,
    max_reissue_count INTEGER NOT NULL DEFAULT 0,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 证书下发记录表
CREATE TABLE IF NOT EXISTS cert_issuance_records (
    uuid            TEXT PRIMARY KEY,
    cert_uuid       TEXT NOT NULL,
    user_uuid       TEXT NOT NULL,
    issuance_method TEXT NOT NULL,
    device_info     TEXT NOT NULL DEFAULT '',
    issued_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cert_issuance_cert ON cert_issuance_records(cert_uuid);

-- 证书重新下发计数器表
CREATE TABLE IF NOT EXISTS cert_reissue_counters (
    cert_uuid     TEXT NOT NULL,
    template_uuid TEXT NOT NULL REFERENCES key_storage_templates(uuid),
    issued_count  INTEGER NOT NULL DEFAULT 0,
    max_count     INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (cert_uuid, template_uuid)
);

-- CA 证书颁发机构表
CREATE TABLE IF NOT EXISTS cas (
    uuid         TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    cert_pem     TEXT NOT NULL,
    private_enc  BLOB NOT NULL,
    parent_uuid  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'active',
    not_before   DATETIME NOT NULL,
    not_after    DATETIME NOT NULL,
    issued_count INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cas_parent ON cas(parent_uuid);
CREATE INDEX IF NOT EXISTS idx_cas_status ON cas(status);

-- 已吊销证书表
CREATE TABLE IF NOT EXISTS revoked_certs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    ca_uuid       TEXT NOT NULL REFERENCES cas(uuid),
    serial_number TEXT NOT NULL,
    revoked_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reason        INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_revoked_certs_ca ON revoked_certs(ca_uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_revoked_certs_serial ON revoked_certs(ca_uuid, serial_number);

-- 主体模板表
CREATE TABLE IF NOT EXISTS subject_templates (
    uuid       TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    fields     TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 扩展信息模板表（SAN 配置）
CREATE TABLE IF NOT EXISTS extension_templates (
    uuid                TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    max_dns             INTEGER NOT NULL DEFAULT 10,
    max_email           INTEGER NOT NULL DEFAULT 5,
    max_ip              INTEGER NOT NULL DEFAULT 5,
    max_uri             INTEGER NOT NULL DEFAULT 5,
    require_dns_verify  INTEGER NOT NULL DEFAULT 0,
    require_email_verify INTEGER NOT NULL DEFAULT 0,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 密钥用途模板表
CREATE TABLE IF NOT EXISTS key_usage_templates (
    uuid           TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    key_usage      INTEGER NOT NULL DEFAULT 0,
    ext_key_usages TEXT NOT NULL DEFAULT '[]',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 证书扩展模板表
CREATE TABLE IF NOT EXISTS cert_ext_templates (
    uuid            TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    crl_dist_points TEXT NOT NULL DEFAULT '[]',
    ocsp_servers    TEXT NOT NULL DEFAULT '[]',
    aia_issuers     TEXT NOT NULL DEFAULT '[]',
    ct_servers      TEXT NOT NULL DEFAULT '[]',
    ev_policy_oid   TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 证书颁发模板表
CREATE TABLE IF NOT EXISTS issuance_templates (
    uuid                 TEXT PRIMARY KEY,
    name                 TEXT NOT NULL,
    is_ca                INTEGER NOT NULL DEFAULT 0,
    path_len             INTEGER NOT NULL DEFAULT 0,
    valid_days           TEXT NOT NULL DEFAULT '[365]',
    allowed_key_types    TEXT NOT NULL DEFAULT '["ec256","rsa2048"]',
    allowed_ca_uuids     TEXT NOT NULL DEFAULT '[]',
    subject_tmpl_uuid    TEXT NOT NULL DEFAULT '',
    extension_tmpl_uuid  TEXT NOT NULL DEFAULT '',
    key_usage_tmpl_uuid  TEXT NOT NULL DEFAULT '',
    key_storage_tmpl_uuid TEXT NOT NULL DEFAULT '',
    price_cents          INTEGER NOT NULL DEFAULT 0,
    stock                INTEGER NOT NULL DEFAULT -1,
    category             TEXT NOT NULL DEFAULT 'custom',
    enabled              INTEGER NOT NULL DEFAULT 1,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_issuance_templates_category ON issuance_templates(category);
CREATE INDEX IF NOT EXISTS idx_issuance_templates_enabled ON issuance_templates(enabled);

-- 吊销服务配置表
CREATE TABLE IF NOT EXISTS revocation_services (
    uuid         TEXT PRIMARY KEY,
    ca_uuid      TEXT NOT NULL REFERENCES cas(uuid),
    service_type TEXT NOT NULL,
    path         TEXT NOT NULL,
    enabled      INTEGER NOT NULL DEFAULT 1,
    crl_interval INTEGER NOT NULL DEFAULT 60
);

CREATE INDEX IF NOT EXISTS idx_revocation_services_ca ON revocation_services(ca_uuid);

-- 证书订单表
CREATE TABLE IF NOT EXISTS cert_orders (
    uuid                  TEXT PRIMARY KEY,
    user_uuid             TEXT NOT NULL REFERENCES users(uuid),
    issuance_tmpl_uuid    TEXT NOT NULL DEFAULT '',
    key_storage_tmpl_uuid TEXT NOT NULL DEFAULT '',
    amount_cents          INTEGER NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'pending',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cert_orders_user ON cert_orders(user_uuid);
CREATE INDEX IF NOT EXISTS idx_cert_orders_status ON cert_orders(status);

-- 证书申请表
CREATE TABLE IF NOT EXISTS cert_applications (
    uuid           TEXT PRIMARY KEY,
    order_uuid     TEXT NOT NULL REFERENCES cert_orders(uuid),
    user_uuid      TEXT NOT NULL REFERENCES users(uuid),
    subject_json   TEXT NOT NULL DEFAULT '{}',
    san_json       TEXT NOT NULL DEFAULT '{}',
    key_type       TEXT NOT NULL DEFAULT 'ec256',
    status         TEXT NOT NULL DEFAULT 'pending',
    approved_by    TEXT,
    approved_at    DATETIME,
    reject_reason  TEXT NOT NULL DEFAULT '',
    cert_uuid      TEXT NOT NULL DEFAULT '',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cert_applications_user ON cert_applications(user_uuid);
CREATE INDEX IF NOT EXISTS idx_cert_applications_order ON cert_applications(order_uuid);
CREATE INDEX IF NOT EXISTS idx_cert_applications_status ON cert_applications(status);

-- 主体信息表（用户提交的主体信息，需审核）
CREATE TABLE IF NOT EXISTS subject_infos (
    uuid              TEXT PRIMARY KEY,
    user_uuid         TEXT NOT NULL REFERENCES users(uuid),
    subject_tmpl_uuid TEXT NOT NULL DEFAULT '',
    field_values      TEXT NOT NULL DEFAULT '{}',
    status            TEXT NOT NULL DEFAULT 'pending',
    reviewed_by       TEXT,
    reviewed_at       DATETIME,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subject_infos_user ON subject_infos(user_uuid);
CREATE INDEX IF NOT EXISTS idx_subject_infos_status ON subject_infos(status);

-- 扩展信息表（域名/邮箱/IP 验证）
CREATE TABLE IF NOT EXISTS extension_infos (
    uuid           TEXT PRIMARY KEY,
    user_uuid      TEXT NOT NULL REFERENCES users(uuid),
    info_type      TEXT NOT NULL,
    value          TEXT NOT NULL,
    verify_method  TEXT NOT NULL DEFAULT '',
    verify_token   TEXT NOT NULL DEFAULT '',
    verify_status  TEXT NOT NULL DEFAULT 'pending',
    verified_at    DATETIME,
    expires_at     DATETIME,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_extension_infos_user ON extension_infos(user_uuid);
CREATE INDEX IF NOT EXISTS idx_extension_infos_status ON extension_infos(verify_status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_extension_infos_value ON extension_infos(user_uuid, info_type, value);

-- 云端智能卡存储区域表
CREATE TABLE IF NOT EXISTS storage_zones (
    uuid         TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    storage_type TEXT NOT NULL DEFAULT 'database',
    hsm_driver   TEXT NOT NULL DEFAULT '',
    hsm_auth_enc BLOB,
    status       TEXT NOT NULL DEFAULT 'active',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 自定义 OID 表
CREATE TABLE IF NOT EXISTS custom_oids (
    uuid        TEXT PRIMARY KEY,
    oid_value   TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    usage_type  TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_custom_oids_usage ON custom_oids(usage_type);

-- 云端 TOTP 条目表
CREATE TABLE IF NOT EXISTS user_totps (
    uuid       TEXT PRIMARY KEY,
    user_uuid  TEXT NOT NULL REFERENCES users(uuid),
    issuer     TEXT NOT NULL DEFAULT '',
    account    TEXT NOT NULL DEFAULT '',
    secret_enc BLOB NOT NULL,
    algorithm  TEXT NOT NULL DEFAULT 'SHA1',
    digits     INTEGER NOT NULL DEFAULT 6,
    period     INTEGER NOT NULL DEFAULT 30,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_totps_user ON user_totps(user_uuid);

-- ACME 服务配置表
CREATE TABLE IF NOT EXISTS acme_configs (
    uuid               TEXT PRIMARY KEY,
    path               TEXT NOT NULL UNIQUE,
    ca_uuid            TEXT NOT NULL DEFAULT '',
    issuance_tmpl_uuid TEXT NOT NULL DEFAULT '',
    enabled            INTEGER NOT NULL DEFAULT 1
);

-- ACME 账户表
CREATE TABLE IF NOT EXISTS acme_accounts (
    uuid       TEXT PRIMARY KEY,
    config_id  TEXT NOT NULL REFERENCES acme_configs(uuid),
    key_id     TEXT NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    contact    TEXT NOT NULL DEFAULT '[]',
    status     TEXT NOT NULL DEFAULT 'valid',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ACME 订单表
CREATE TABLE IF NOT EXISTS acme_orders (
    uuid          TEXT PRIMARY KEY,
    account_uuid  TEXT NOT NULL REFERENCES acme_accounts(uuid),
    status        TEXT NOT NULL DEFAULT 'pending',
    identifiers   TEXT NOT NULL DEFAULT '[]',
    not_before    DATETIME,
    not_after     DATETIME,
    cert_url      TEXT NOT NULL DEFAULT '',
    finalize_url  TEXT NOT NULL DEFAULT '',
    expires       DATETIME NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_acme_orders_account ON acme_orders(account_uuid);
CREATE INDEX IF NOT EXISTS idx_acme_orders_status ON acme_orders(status);

-- ACME 授权表
CREATE TABLE IF NOT EXISTS acme_authorizations (
    uuid        TEXT PRIMARY KEY,
    order_uuid  TEXT NOT NULL REFERENCES acme_orders(uuid),
    identifier  TEXT NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'pending',
    expires     DATETIME NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_acme_authz_order ON acme_authorizations(order_uuid);

-- ACME 挑战表
CREATE TABLE IF NOT EXISTS acme_challenges (
    uuid         TEXT PRIMARY KEY,
    authz_uuid   TEXT NOT NULL REFERENCES acme_authorizations(uuid),
    type         TEXT NOT NULL,
    token        TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    validated_at DATETIME,
    error        TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_acme_challenges_authz ON acme_challenges(authz_uuid);

-- CT 提交记录表
CREATE TABLE IF NOT EXISTS ct_entries (
    uuid         TEXT PRIMARY KEY,
    cert_uuid    TEXT NOT NULL DEFAULT '',
    ca_uuid      TEXT NOT NULL DEFAULT '',
    cert_hash    TEXT NOT NULL,
    ct_server    TEXT NOT NULL,
    sct_data     BLOB,
    status       TEXT NOT NULL DEFAULT 'pending',
    submitted_by TEXT NOT NULL DEFAULT '',
    submitted_at DATETIME,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ct_entries_cert ON ct_entries(cert_uuid);
CREATE INDEX IF NOT EXISTS idx_ct_entries_hash ON ct_entries(cert_hash);
`
	_, err := db.Exec(schema)
	return err
}
