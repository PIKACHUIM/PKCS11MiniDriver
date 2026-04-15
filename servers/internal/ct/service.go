// Package ct 提供证书透明度（Certificate Transparency）功能。
package ct

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// CTEntry 是 CT 提交记录模型。
type CTEntry struct {
	UUID         string    `json:"uuid"`
	CertUUID     string    `json:"cert_uuid"`
	CAUUID       string    `json:"ca_uuid"`
	CertHash     string    `json:"cert_hash"`      // 证书 SHA-256 指纹
	CTServer     string    `json:"ct_server"`       // CT 日志服务器地址
	SCTData      []byte    `json:"sct_data"`        // Signed Certificate Timestamp 数据
	Status       string    `json:"status"`          // pending/submitted/failed
	SubmittedBy  string    `json:"submitted_by"`    // 提交者用户 UUID
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Service 是 CT 服务。
type Service struct {
	db *storage.DB
}

// NewService 创建 CT 服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// Submit 提交证书到 CT 日志。
func (s *Service) Submit(ctx context.Context, certUUID, caUUID, ctServer, submittedBy string, certDER []byte) (*CTEntry, error) {
	if certUUID == "" || ctServer == "" {
		return nil, fmt.Errorf("证书 UUID 和 CT 服务器地址不能为空")
	}

	// 计算证书哈希
	hash := sha256.Sum256(certDER)
	certHash := hex.EncodeToString(hash[:])

	entry := &CTEntry{
		UUID:        uuid.New().String(),
		CertUUID:    certUUID,
		CAUUID:      caUUID,
		CertHash:    certHash,
		CTServer:    ctServer,
		Status:      "submitted",
		SubmittedBy: submittedBy,
		CreatedAt:   time.Now(),
	}
	now := time.Now()
	entry.SubmittedAt = &now

	// TODO: 实际调用 CT 日志服务器 API（RFC 6962 add-chain）
	// 这里先记录提交记录，实际的 SCT 获取需要 HTTP 调用 CT 服务器

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ct_entries (uuid, cert_uuid, ca_uuid, cert_hash, ct_server, sct_data, status, submitted_by, submitted_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.UUID, entry.CertUUID, entry.CAUUID, entry.CertHash, entry.CTServer,
		entry.SCTData, entry.Status, entry.SubmittedBy, entry.SubmittedAt, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("保存 CT 记录失败: %w", err)
	}

	return entry, nil
}

// List 查询 CT 提交记录列表。
func (s *Service) List(ctx context.Context, certUUID string, page, pageSize int) ([]*CTEntry, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `SELECT COUNT(*) FROM ct_entries`
	countArgs := []interface{}{}
	if certUUID != "" {
		query += ` WHERE cert_uuid = ?`
		countArgs = append(countArgs, certUUID)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, query, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT uuid, cert_uuid, ca_uuid, cert_hash, ct_server, sct_data, status, submitted_by, submitted_at, created_at
		 FROM ct_entries`
	selectArgs := []interface{}{}
	if certUUID != "" {
		selectQuery += ` WHERE cert_uuid = ?`
		selectArgs = append(selectArgs, certUUID)
	}
	selectQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	selectArgs = append(selectArgs, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*CTEntry
	for rows.Next() {
		e := &CTEntry{}
		var submittedAt sql.NullTime
		if err := rows.Scan(&e.UUID, &e.CertUUID, &e.CAUUID, &e.CertHash, &e.CTServer,
			&e.SCTData, &e.Status, &e.SubmittedBy, &submittedAt, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		if submittedAt.Valid {
			e.SubmittedAt = &submittedAt.Time
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// GetByUUID 按 UUID 查询 CT 记录。
func (s *Service) GetByUUID(ctx context.Context, entryUUID string) (*CTEntry, error) {
	e := &CTEntry{}
	var submittedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, cert_uuid, ca_uuid, cert_hash, ct_server, sct_data, status, submitted_by, submitted_at, created_at
		 FROM ct_entries WHERE uuid = ?`, entryUUID,
	).Scan(&e.UUID, &e.CertUUID, &e.CAUUID, &e.CertHash, &e.CTServer,
		&e.SCTData, &e.Status, &e.SubmittedBy, &submittedAt, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CT 记录不存在: %s", entryUUID)
	}
	if submittedAt.Valid {
		e.SubmittedAt = &submittedAt.Time
	}
	return e, err
}

// Delete 删除 CT 记录。
func (s *Service) Delete(ctx context.Context, entryUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ct_entries WHERE uuid = ?`, entryUUID)
	return err
}

// QueryByCertHash 按证书哈希查询 CT 记录（供外部查询）。
func (s *Service) QueryByCertHash(ctx context.Context, certHash string) ([]*CTEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, cert_uuid, ca_uuid, cert_hash, ct_server, sct_data, status, submitted_by, submitted_at, created_at
		 FROM ct_entries WHERE cert_hash = ? AND status = 'submitted' ORDER BY created_at DESC`, certHash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*CTEntry
	for rows.Next() {
		e := &CTEntry{}
		var submittedAt sql.NullTime
		if err := rows.Scan(&e.UUID, &e.CertUUID, &e.CAUUID, &e.CertHash, &e.CTServer,
			&e.SCTData, &e.Status, &e.SubmittedBy, &submittedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		if submittedAt.Valid {
			e.SubmittedAt = &submittedAt.Time
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
