// Package storage - Repository 实现。
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---- UserRepo ----

// UserRepo 提供用户数据访问。
type UserRepo struct {
	db *DB
}

// NewUserRepo 创建 UserRepo。
func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create 创建用户。
func (r *UserRepo) Create(ctx context.Context, u *User) error {
	u.UUID = uuid.New().String()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (uuid, username, display_name, email, password_hash, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.UUID, u.Username, u.DisplayName, u.Email, u.PasswordHash, boolToInt(u.Enabled), u.CreatedAt, u.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询用户。
func (r *UserRepo) GetByUUID(ctx context.Context, uuid string) (*User, error) {
	u := &User{}
	var enabled int
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, username, display_name, email, password_hash, enabled, created_at, updated_at
		 FROM users WHERE uuid = ?`, uuid,
	).Scan(&u.UUID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", uuid)
	}
	u.Enabled = enabled == 1
	return u, err
}

// GetByUsername 按用户名查询用户。
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	var enabled int
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, username, display_name, email, password_hash, enabled, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.UUID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &enabled, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}
	u.Enabled = enabled == 1
	return u, err
}

// ---- CardRepo ----

// CardRepo 提供云端卡片数据访问。
type CardRepo struct {
	db *DB
}

// NewCardRepo 创建 CardRepo。
func NewCardRepo(db *DB) *CardRepo {
	return &CardRepo{db: db}
}

// Create 创建卡片。
func (r *CardRepo) Create(ctx context.Context, c *Card) error {
	c.UUID = uuid.New().String()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO cards (uuid, user_uuid, card_name, remark, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.UUID, c.UserUUID, c.CardName, c.Remark, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询卡片。
func (r *CardRepo) GetByUUID(ctx context.Context, cardUUID string) (*Card, error) {
	c := &Card{}
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, user_uuid, card_name, remark, created_at, updated_at
		 FROM cards WHERE uuid = ?`, cardUUID,
	).Scan(&c.UUID, &c.UserUUID, &c.CardName, &c.Remark, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("卡片不存在: %s", cardUUID)
	}
	return c, err
}

// ListByUser 查询用户的所有卡片。
func (r *CardRepo) ListByUser(ctx context.Context, userUUID string) ([]*Card, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT uuid, user_uuid, card_name, remark, created_at, updated_at
		 FROM cards WHERE user_uuid = ? ORDER BY created_at DESC`, userUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []*Card
	for rows.Next() {
		c := &Card{}
		if err := rows.Scan(&c.UUID, &c.UserUUID, &c.CardName, &c.Remark, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, rows.Err()
}

// Delete 删除卡片（级联删除证书）。
func (r *CardRepo) Delete(ctx context.Context, cardUUID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cards WHERE uuid = ?`, cardUUID)
	return err
}

// ---- CertRepo ----

// CertRepo 提供云端证书数据访问。
type CertRepo struct {
	db *DB
}

// NewCertRepo 创建 CertRepo。
func NewCertRepo(db *DB) *CertRepo {
	return &CertRepo{db: db}
}

// Create 创建证书。
func (r *CertRepo) Create(ctx context.Context, c *Certificate) error {
	c.UUID = uuid.New().String()
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO certificates (uuid, card_uuid, cert_type, key_type, cert_content, private_data, remark, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.UUID, c.CardUUID, c.CertType, c.KeyType, c.CertContent, c.PrivateData, c.Remark, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询证书（含私钥）。
func (r *CertRepo) GetByUUID(ctx context.Context, certUUID string) (*Certificate, error) {
	c := &Certificate{}
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, card_uuid, cert_type, key_type, cert_content, private_data, remark, created_at, updated_at
		 FROM certificates WHERE uuid = ?`, certUUID,
	).Scan(&c.UUID, &c.CardUUID, &c.CertType, &c.KeyType, &c.CertContent, &c.PrivateData, &c.Remark, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("证书不存在: %s", certUUID)
	}
	return c, err
}

// ListByCard 查询卡片的所有证书（不含私钥）。
func (r *CertRepo) ListByCard(ctx context.Context, cardUUID string) ([]*Certificate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT uuid, card_uuid, cert_type, key_type, cert_content, remark, created_at, updated_at
		 FROM certificates WHERE card_uuid = ? ORDER BY created_at DESC`, cardUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*Certificate
	for rows.Next() {
		c := &Certificate{}
		if err := rows.Scan(&c.UUID, &c.CardUUID, &c.CertType, &c.KeyType, &c.CertContent, &c.Remark, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

// Delete 删除证书。
func (r *CertRepo) Delete(ctx context.Context, certUUID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM certificates WHERE uuid = ?`, certUUID)
	return err
}

// ---- LogRepo ----

// LogRepo 提供操作日志数据访问。
type LogRepo struct {
	db *DB
}

// NewLogRepo 创建 LogRepo。
func NewLogRepo(db *DB) *LogRepo {
	return &LogRepo{db: db}
}

// Create 写入操作日志。
func (r *LogRepo) Create(ctx context.Context, l *Log) error {
	l.RecordedAt = time.Now()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO logs (user_uuid, card_uuid, cert_uuid, action, ip_addr, user_agent, recorded_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		l.UserUUID, l.CardUUID, l.CertUUID, l.Action, l.IPAddr, l.UserAgent, l.RecordedAt,
	)
	return err
}

// ---- 工具函数 ----

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
