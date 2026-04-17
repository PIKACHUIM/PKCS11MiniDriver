// Package acme 提供 ACME 协议服务（RFC 8555）。
package acme

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Config 是 ACME 服务实例配置。
type Config struct {
	UUID             string `json:"uuid"`
	Path             string `json:"path"`              // 路径前缀（如 "letsencrypt"）
	CAUUID           string `json:"ca_uuid"`            // 关联的 CA
	IssuanceTmplUUID string `json:"issuance_tmpl_uuid"` // 关联的颁发模板
	Enabled          bool   `json:"enabled"`
}

// Account 是 ACME 账户。
type Account struct {
	UUID      string    `json:"uuid"`
	ConfigID  string    `json:"config_id"`
	KeyID     string    `json:"key_id"`     // JWK Thumbprint
	PublicKey string    `json:"public_key"` // JWK JSON
	Contact   string    `json:"contact"`    // JSON 数组
	Status    string    `json:"status"`     // valid/deactivated/revoked
	CreatedAt time.Time `json:"created_at"`
}

// Order 是 ACME 订单。
type Order struct {
	UUID           string     `json:"uuid"`
	AccountUUID    string     `json:"account_uuid"`
	Status         string     `json:"status"` // pending/ready/processing/valid/invalid
	Identifiers    string     `json:"identifiers"` // JSON 数组 [{"type":"dns","value":"example.com"}]
	NotBefore      *time.Time `json:"not_before,omitempty"`
	NotAfter       *time.Time `json:"not_after,omitempty"`
	CertURL        string     `json:"cert_url,omitempty"`
	FinalizeURL    string     `json:"finalize_url"`
	Expires        time.Time  `json:"expires"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Authorization 是 ACME 授权。
type Authorization struct {
	UUID        string    `json:"uuid"`
	OrderUUID   string    `json:"order_uuid"`
	Identifier  string    `json:"identifier"` // JSON {"type":"dns","value":"example.com"}
	Status      string    `json:"status"`     // pending/valid/invalid/deactivated/expired/revoked
	Expires     time.Time `json:"expires"`
	CreatedAt   time.Time `json:"created_at"`
}

// Challenge 是 ACME 挑战。
type Challenge struct {
	UUID          string     `json:"uuid"`
	AuthzUUID     string     `json:"authz_uuid"`
	Type          string     `json:"type"`   // http-01/dns-01
	Token         string     `json:"token"`
	Status        string     `json:"status"` // pending/processing/valid/invalid
	ValidatedAt   *time.Time `json:"validated_at,omitempty"`
	ErrorJSON     string     `json:"error,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Service 是 ACME 服务。
type Service struct {
	db *storage.DB
}

// NewService 创建 ACME 服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// ---- ACME 配置管理 ----

// CreateConfig 创建 ACME 服务实例配置。
func (s *Service) CreateConfig(ctx context.Context, cfg *Config) error {
	if cfg.Path == "" {
		return fmt.Errorf("ACME 路径不能为空")
	}
	cfg.UUID = uuid.New().String()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acme_configs (uuid, path, ca_uuid, issuance_tmpl_uuid, enabled)
		 VALUES (?, ?, ?, ?, ?)`,
		cfg.UUID, cfg.Path, cfg.CAUUID, cfg.IssuanceTmplUUID, boolToInt(cfg.Enabled),
	)
	return err
}

// ListConfigs 查询所有 ACME 配置。
func (s *Service) ListConfigs(ctx context.Context) ([]*Config, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, path, ca_uuid, issuance_tmpl_uuid, enabled FROM acme_configs ORDER BY path`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*Config
	for rows.Next() {
		c := &Config{}
		var enabled int
		if err := rows.Scan(&c.UUID, &c.Path, &c.CAUUID, &c.IssuanceTmplUUID, &enabled); err != nil {
			return nil, err
		}
		c.Enabled = enabled == 1
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

// GetConfigByPath 按路径查询 ACME 配置。
func (s *Service) GetConfigByPath(ctx context.Context, path string) (*Config, error) {
	c := &Config{}
	var enabled int
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, path, ca_uuid, issuance_tmpl_uuid, enabled FROM acme_configs WHERE path = ? AND enabled = 1`, path,
	).Scan(&c.UUID, &c.Path, &c.CAUUID, &c.IssuanceTmplUUID, &enabled)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ACME 服务不存在: %s", path)
	}
	c.Enabled = enabled == 1
	return c, err
}

// DeleteConfig 删除 ACME 配置。
func (s *Service) DeleteConfig(ctx context.Context, cfgUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM acme_configs WHERE uuid = ?`, cfgUUID)
	return err
}

// ---- ACME 账户 ----

// CreateAccount 创建 ACME 账户。
func (s *Service) CreateAccount(ctx context.Context, acct *Account) error {
	acct.UUID = uuid.New().String()
	acct.Status = "valid"
	acct.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acme_accounts (uuid, config_id, key_id, public_key, contact, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		acct.UUID, acct.ConfigID, acct.KeyID, acct.PublicKey, acct.Contact, acct.Status, acct.CreatedAt,
	)
	return err
}

// GetAccountByKeyID 按 KeyID 查询账户。
func (s *Service) GetAccountByKeyID(ctx context.Context, keyID string) (*Account, error) {
	acct := &Account{}
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, config_id, key_id, public_key, contact, status, created_at
		 FROM acme_accounts WHERE key_id = ?`, keyID,
	).Scan(&acct.UUID, &acct.ConfigID, &acct.KeyID, &acct.PublicKey, &acct.Contact, &acct.Status, &acct.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ACME 账户不存在")
	}
	return acct, err
}

// GetAccountByUUID 按 UUID 查询账户。
func (s *Service) GetAccountByUUID(ctx context.Context, acctUUID string) (*Account, error) {
	acct := &Account{}
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, config_id, key_id, public_key, contact, status, created_at
		 FROM acme_accounts WHERE uuid = ?`, acctUUID,
	).Scan(&acct.UUID, &acct.ConfigID, &acct.KeyID, &acct.PublicKey, &acct.Contact, &acct.Status, &acct.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ACME 账户不存在: %s", acctUUID)
	}
	return acct, err
}

// ---- ACME 订单 ----

// CreateOrder 创建 ACME 订单。
func (s *Service) CreateOrder(ctx context.Context, order *Order) error {
	order.UUID = uuid.New().String()
	order.Status = "pending"
	order.Expires = time.Now().Add(7 * 24 * time.Hour) // 7 天过期
	order.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acme_orders (uuid, account_uuid, status, identifiers, not_before, not_after, finalize_url, expires, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		order.UUID, order.AccountUUID, order.Status, order.Identifiers,
		order.NotBefore, order.NotAfter, order.FinalizeURL, order.Expires, order.CreatedAt,
	)
	return err
}

// GetOrder 按 UUID 查询订单。
func (s *Service) GetOrder(ctx context.Context, orderUUID string) (*Order, error) {
	o := &Order{}
	var notBefore, notAfter sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, account_uuid, status, identifiers, not_before, not_after, cert_url, finalize_url, expires, created_at
		 FROM acme_orders WHERE uuid = ?`, orderUUID,
	).Scan(&o.UUID, &o.AccountUUID, &o.Status, &o.Identifiers, &notBefore, &notAfter,
		&o.CertURL, &o.FinalizeURL, &o.Expires, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ACME 订单不存在: %s", orderUUID)
	}
	if notBefore.Valid {
		o.NotBefore = &notBefore.Time
	}
	if notAfter.Valid {
		o.NotAfter = &notAfter.Time
	}
	return o, err
}

// UpdateOrderStatus 更新订单状态。
func (s *Service) UpdateOrderStatus(ctx context.Context, orderUUID, status, certURL string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE acme_orders SET status = ?, cert_url = ? WHERE uuid = ?`,
		status, certURL, orderUUID,
	)
	return err
}

// ---- ACME 授权与挑战 ----

// CreateAuthorization 创建授权。
func (s *Service) CreateAuthorization(ctx context.Context, authz *Authorization) error {
	authz.UUID = uuid.New().String()
	authz.Status = "pending"
	authz.Expires = time.Now().Add(7 * 24 * time.Hour)
	authz.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acme_authorizations (uuid, order_uuid, identifier, status, expires, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		authz.UUID, authz.OrderUUID, authz.Identifier, authz.Status, authz.Expires, authz.CreatedAt,
	)
	return err
}

// CreateChallenge 创建挑战。
func (s *Service) CreateChallenge(ctx context.Context, ch *Challenge) error {
	ch.UUID = uuid.New().String()
	ch.Status = "pending"
	ch.CreatedAt = time.Now()

	// 生成随机 token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("生成 token 失败: %w", err)
	}
	ch.Token = base64.RawURLEncoding.EncodeToString(tokenBytes)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO acme_challenges (uuid, authz_uuid, type, token, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		ch.UUID, ch.AuthzUUID, ch.Type, ch.Token, ch.Status, ch.CreatedAt,
	)
	return err
}

// GetChallenge 按 UUID 查询挑战。
func (s *Service) GetChallenge(ctx context.Context, chUUID string) (*Challenge, error) {
	ch := &Challenge{}
	var validatedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, authz_uuid, type, token, status, validated_at, error, created_at
		 FROM acme_challenges WHERE uuid = ?`, chUUID,
	).Scan(&ch.UUID, &ch.AuthzUUID, &ch.Type, &ch.Token, &ch.Status, &validatedAt, &ch.ErrorJSON, &ch.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ACME 挑战不存在: %s", chUUID)
	}
	if validatedAt.Valid {
		ch.ValidatedAt = &validatedAt.Time
	}
	return ch, err
}

// ValidateChallenge 标记挑战为已验证。
func (s *Service) ValidateChallenge(ctx context.Context, chUUID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE acme_challenges SET status = 'valid', validated_at = ? WHERE uuid = ? AND status = 'pending'`,
		now, chUUID,
	)
	return err
}

// GenerateNonce 生成 ACME Nonce。
func (s *Service) GenerateNonce() (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(nonceBytes), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
