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
	if u.Role == "" {
		u.Role = "user"
	}
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (uuid, username, display_name, email, password_hash, role, public_key, totp_secret, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.UUID, u.Username, u.DisplayName, u.Email, u.PasswordHash, u.Role, u.PublicKey, u.TOTPSecret, boolToInt(u.Enabled), u.CreatedAt, u.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询用户。
func (r *UserRepo) GetByUUID(ctx context.Context, userUUID string) (*User, error) {
	u := &User{}
	var enabled, failedAttempts int
	var lockedUntil sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, username, display_name, email, password_hash, role, public_key, totp_secret, enabled, failed_attempts, locked_until, created_at, updated_at
		 FROM users WHERE uuid = ?`, userUUID,
	).Scan(&u.UUID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Role, &u.PublicKey, &u.TOTPSecret, &enabled, &failedAttempts, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", userUUID)
	}
	u.Enabled = enabled == 1
	u.FailedAttempts = failedAttempts
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	return u, err
}

// GetByUsername 按用户名查询用户。
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	var enabled, failedAttempts int
	var lockedUntil sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, username, display_name, email, password_hash, role, public_key, totp_secret, enabled, failed_attempts, locked_until, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	).Scan(&u.UUID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Role, &u.PublicKey, &u.TOTPSecret, &enabled, &failedAttempts, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}
	u.Enabled = enabled == 1
	u.FailedAttempts = failedAttempts
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	return u, err
}

// GetByEmail 按邮箱查询用户。
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	var enabled, failedAttempts int
	var lockedUntil sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, username, display_name, email, password_hash, role, public_key, totp_secret, enabled, failed_attempts, locked_until, created_at, updated_at
		 FROM users WHERE email = ?`, email,
	).Scan(&u.UUID, &u.Username, &u.DisplayName, &u.Email, &u.PasswordHash, &u.Role, &u.PublicKey, &u.TOTPSecret, &enabled, &failedAttempts, &lockedUntil, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("用户不存在: %s", email)
	}
	u.Enabled = enabled == 1
	u.FailedAttempts = failedAttempts
	if lockedUntil.Valid {
		u.LockedUntil = &lockedUntil.Time
	}
	return u, err
}

// UpdatePassword 更新用户密码。
func (r *UserRepo) UpdatePassword(ctx context.Context, userUUID, newPasswordHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE uuid = ?`,
		newPasswordHash, time.Now(), userUUID,
	)
	return err
}

// UpdateProfile 更新用户个人信息。
func (r *UserRepo) UpdateProfile(ctx context.Context, userUUID, displayName, email string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, email = ?, updated_at = ? WHERE uuid = ?`,
		displayName, email, time.Now(), userUUID,
	)
	return err
}

// UpdatePublicKey 更新用户云端公钥。
func (r *UserRepo) UpdatePublicKey(ctx context.Context, userUUID string, publicKey []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET public_key = ?, updated_at = ? WHERE uuid = ?`,
		publicKey, time.Now(), userUUID,
	)
	return err
}

// IncrementFailedAttempts 递增登录失败次数，达到阈值时锁定。
func (r *UserRepo) IncrementFailedAttempts(ctx context.Context, userUUID string, maxAttempts int, lockDuration time.Duration) error {
	now := time.Now()
	lockedUntil := now.Add(lockDuration)

	// 递增失败次数，达到阈值时设置锁定时间
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET
			failed_attempts = failed_attempts + 1,
			locked_until = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END,
			updated_at = ?
		 WHERE uuid = ?`,
		maxAttempts, lockedUntil, now, userUUID,
	)
	return err
}

// ResetFailedAttempts 重置登录失败次数（登录成功时调用）。
func (r *UserRepo) ResetFailedAttempts(ctx context.Context, userUUID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_attempts = 0, locked_until = NULL, updated_at = ? WHERE uuid = ?`,
		time.Now(), userUUID,
	)
	return err
}

// IsLocked 检查用户是否被锁定。
func (r *UserRepo) IsLocked(ctx context.Context, userUUID string) (bool, error) {
	var lockedUntil sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT locked_until FROM users WHERE uuid = ?`, userUUID,
	).Scan(&lockedUntil)
	if err != nil {
		return false, err
	}
	if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
		return true, nil
	}
	return false, nil
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
	if c.RevocationStatus == "" {
		c.RevocationStatus = "active"
	}
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO certificates (uuid, card_uuid, cert_type, key_type, cert_content, private_data, remark,
		 order_no, ca_uuid, issuance_tmpl_uuid, storage_policy, revocation_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.UUID, c.CardUUID, c.CertType, c.KeyType, c.CertContent, c.PrivateData, c.Remark,
		c.OrderNo, c.CAUUID, c.IssuanceTmplUUID, c.StoragePolicy, c.RevocationStatus, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询证书（含私钥）。
func (r *CertRepo) GetByUUID(ctx context.Context, certUUID string) (*Certificate, error) {
	c := &Certificate{}
	var revokedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT uuid, card_uuid, cert_type, key_type, cert_content, private_data, remark,
		 order_no, ca_uuid, issuance_tmpl_uuid, storage_policy, revocation_status, revoked_at, created_at, updated_at
		 FROM certificates WHERE uuid = ?`, certUUID,
	).Scan(&c.UUID, &c.CardUUID, &c.CertType, &c.KeyType, &c.CertContent, &c.PrivateData, &c.Remark,
		&c.OrderNo, &c.CAUUID, &c.IssuanceTmplUUID, &c.StoragePolicy, &c.RevocationStatus, &revokedAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("证书不存在: %s", certUUID)
	}
	if revokedAt.Valid {
		c.RevokedAt = &revokedAt.Time
	}
	return c, err
}

// ListByCard 查询卡片的所有证书（不含私钥）。
func (r *CertRepo) ListByCard(ctx context.Context, cardUUID string) ([]*Certificate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT uuid, card_uuid, cert_type, key_type, cert_content, remark,
		 order_no, ca_uuid, issuance_tmpl_uuid, storage_policy, revocation_status, revoked_at, created_at, updated_at
		 FROM certificates WHERE card_uuid = ? ORDER BY created_at DESC`, cardUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*Certificate
	for rows.Next() {
		c := &Certificate{}
		var revokedAt sql.NullTime
		if err := rows.Scan(&c.UUID, &c.CardUUID, &c.CertType, &c.KeyType, &c.CertContent, &c.Remark,
			&c.OrderNo, &c.CAUUID, &c.IssuanceTmplUUID, &c.StoragePolicy, &c.RevocationStatus, &revokedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		if revokedAt.Valid {
			c.RevokedAt = &revokedAt.Time
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

// ListFiltered 按条件筛选查询证书（支持按用户/CA/模板/状态筛选）。
func (r *CertRepo) ListFiltered(ctx context.Context, userUUID, caUUID, tmplUUID, status string, page, pageSize int) ([]*Certificate, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 构建 WHERE 子句
	where := `WHERE 1=1`
	var args []interface{}
	if userUUID != "" {
		where += ` AND c.card_uuid IN (SELECT uuid FROM cards WHERE user_uuid = ?)`
		args = append(args, userUUID)
	}
	if caUUID != "" {
		where += ` AND c.ca_uuid = ?`
		args = append(args, caUUID)
	}
	if tmplUUID != "" {
		where += ` AND c.issuance_tmpl_uuid = ?`
		args = append(args, tmplUUID)
	}
	if status != "" {
		where += ` AND c.revocation_status = ?`
		args = append(args, status)
	}

	// 查询总数
	var total int
	countQuery := `SELECT COUNT(*) FROM certificates c ` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 查询列表
	selectQuery := `SELECT c.uuid, c.card_uuid, c.cert_type, c.key_type, c.cert_content, c.remark,
		 c.order_no, c.ca_uuid, c.issuance_tmpl_uuid, c.storage_policy, c.revocation_status, c.revoked_at, c.created_at, c.updated_at
		 FROM certificates c ` + where + ` ORDER BY c.created_at DESC LIMIT ? OFFSET ?`
	selectArgs := append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var certs []*Certificate
	for rows.Next() {
		c := &Certificate{}
		var revokedAt sql.NullTime
		if err := rows.Scan(&c.UUID, &c.CardUUID, &c.CertType, &c.KeyType, &c.CertContent, &c.Remark,
			&c.OrderNo, &c.CAUUID, &c.IssuanceTmplUUID, &c.StoragePolicy, &c.RevocationStatus, &revokedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if revokedAt.Valid {
			c.RevokedAt = &revokedAt.Time
		}
		certs = append(certs, c)
	}
	return certs, total, rows.Err()
}

// Revoke 吊销证书（更新状态）。
func (r *CertRepo) Revoke(ctx context.Context, certUUID string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE certificates SET revocation_status = 'revoked', revoked_at = ?, updated_at = ? WHERE uuid = ?`,
		now, now, certUUID,
	)
	return err
}

// AssignToCard 将证书分配到指定智能卡。
func (r *CertRepo) AssignToCard(ctx context.Context, certUUID, targetCardUUID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE certificates SET card_uuid = ?, updated_at = ? WHERE uuid = ?`,
		targetCardUUID, time.Now(), certUUID,
	)
	return err
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
