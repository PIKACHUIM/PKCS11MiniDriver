// Package verification 提供域名和邮箱验证功能。
package verification

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 是验证服务。
type Service struct {
	db *storage.DB
}

// NewService 创建验证服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// ---- 主体信息管理 ----

// CreateSubjectInfo 创建主体信息（待审核）。
func (s *Service) CreateSubjectInfo(ctx context.Context, info *storage.SubjectInfo) error {
	info.UUID = uuid.New().String()
	info.Status = "pending"
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subject_infos (uuid, user_uuid, subject_tmpl_uuid, field_values, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		info.UUID, info.UserUUID, info.SubjectTmplUUID, info.FieldValues, info.Status, info.CreatedAt, info.UpdatedAt,
	)
	return err
}

// ListSubjectInfos 查询用户的主体信息列表。
func (s *Service) ListSubjectInfos(ctx context.Context, userUUID string) ([]*storage.SubjectInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, user_uuid, subject_tmpl_uuid, field_values, status, reviewed_by, reviewed_at, created_at, updated_at
		 FROM subject_infos WHERE user_uuid = ? ORDER BY created_at DESC`, userUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []*storage.SubjectInfo
	for rows.Next() {
		info := &storage.SubjectInfo{}
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		if err := rows.Scan(&info.UUID, &info.UserUUID, &info.SubjectTmplUUID, &info.FieldValues,
			&info.Status, &reviewedBy, &reviewedAt, &info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, err
		}
		if reviewedBy.Valid {
			info.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			info.ReviewedAt = &reviewedAt.Time
		}
		infos = append(infos, info)
	}
	return infos, rows.Err()
}

// ApproveSubjectInfo 审核通过主体信息。
func (s *Service) ApproveSubjectInfo(ctx context.Context, infoUUID, adminUUID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE subject_infos SET status = 'approved', reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE uuid = ? AND status = 'pending'`,
		adminUUID, now, now, infoUUID,
	)
	return err
}

// RejectSubjectInfo 审核拒绝主体信息。
func (s *Service) RejectSubjectInfo(ctx context.Context, infoUUID, adminUUID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE subject_infos SET status = 'rejected', reviewed_by = ?, reviewed_at = ?, updated_at = ? WHERE uuid = ? AND status = 'pending'`,
		adminUUID, now, now, infoUUID,
	)
	return err
}

// ---- 扩展信息（域名/邮箱/IP）验证 ----

// CreateExtensionInfo 创建扩展信息验证请求。
func (s *Service) CreateExtensionInfo(ctx context.Context, info *storage.ExtensionInfo) error {
	info.UUID = uuid.New().String()
	info.VerifyStatus = "pending"
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()

	// 生成验证 token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("生成验证 token 失败: %w", err)
	}
	info.VerifyToken = hex.EncodeToString(tokenBytes)

	// 根据类型设置默认验证方式
	switch info.InfoType {
	case "domain":
		if info.VerifyMethod == "" {
			info.VerifyMethod = "txt" // 默认 DNS TXT 验证
		}
	case "email":
		info.VerifyMethod = "email" // 邮箱只能通过邮件验证
	case "ip":
		info.VerifyMethod = "http" // IP 通过 HTTP 验证
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO extension_infos (uuid, user_uuid, info_type, value, verify_method, verify_token, verify_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		info.UUID, info.UserUUID, info.InfoType, info.Value, info.VerifyMethod, info.VerifyToken,
		info.VerifyStatus, info.CreatedAt, info.UpdatedAt,
	)
	return err
}

// ListExtensionInfos 查询用户的扩展信息列表。
func (s *Service) ListExtensionInfos(ctx context.Context, userUUID string) ([]*storage.ExtensionInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, user_uuid, info_type, value, verify_method, verify_token, verify_status, verified_at, expires_at, created_at, updated_at
		 FROM extension_infos WHERE user_uuid = ? ORDER BY created_at DESC`, userUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var infos []*storage.ExtensionInfo
	for rows.Next() {
		info := &storage.ExtensionInfo{}
		var verifiedAt, expiresAt sql.NullTime
		if err := rows.Scan(&info.UUID, &info.UserUUID, &info.InfoType, &info.Value, &info.VerifyMethod,
			&info.VerifyToken, &info.VerifyStatus, &verifiedAt, &expiresAt, &info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, err
		}
		if verifiedAt.Valid {
			info.VerifiedAt = &verifiedAt.Time
		}
		if expiresAt.Valid {
			info.ExpiresAt = &expiresAt.Time
		}
		infos = append(infos, info)
	}
	return infos, rows.Err()
}

// VerifyDNSTXT 验证域名 DNS TXT 记录。
func (s *Service) VerifyDNSTXT(ctx context.Context, infoUUID string) error {
	info, err := s.getExtensionInfo(ctx, infoUUID)
	if err != nil {
		return err
	}
	if info.InfoType != "domain" || info.VerifyMethod != "txt" {
		return fmt.Errorf("此验证项不支持 DNS TXT 验证")
	}

	// 查询 DNS TXT 记录
	expectedRecord := fmt.Sprintf("opencert-verify=%s", info.VerifyToken)
	txtRecords, err := net.LookupTXT(fmt.Sprintf("_opencert.%s", info.Value))
	if err != nil {
		return fmt.Errorf("DNS 查询失败: %w", err)
	}

	for _, txt := range txtRecords {
		if strings.TrimSpace(txt) == expectedRecord {
			return s.markVerified(ctx, infoUUID)
		}
	}

	return fmt.Errorf("未找到匹配的 DNS TXT 记录，期望: %s", expectedRecord)
}

// VerifyEmailCode 验证邮箱验证码。
func (s *Service) VerifyEmailCode(ctx context.Context, infoUUID, code string) error {
	info, err := s.getExtensionInfo(ctx, infoUUID)
	if err != nil {
		return err
	}
	if info.InfoType != "email" {
		return fmt.Errorf("此验证项不支持邮箱验证")
	}

	// 验证码是 token 的前 6 位
	expectedCode := info.VerifyToken[:6]
	if code != expectedCode {
		return fmt.Errorf("验证码错误")
	}

	return s.markVerified(ctx, infoUUID)
}

// DeleteExtensionInfo 删除扩展信息。
func (s *Service) DeleteExtensionInfo(ctx context.Context, infoUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM extension_infos WHERE uuid = ?`, infoUUID)
	return err
}

// ---- 内部方法 ----

func (s *Service) getExtensionInfo(ctx context.Context, infoUUID string) (*storage.ExtensionInfo, error) {
	info := &storage.ExtensionInfo{}
	var verifiedAt, expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, user_uuid, info_type, value, verify_method, verify_token, verify_status, verified_at, expires_at
		 FROM extension_infos WHERE uuid = ?`, infoUUID,
	).Scan(&info.UUID, &info.UserUUID, &info.InfoType, &info.Value, &info.VerifyMethod,
		&info.VerifyToken, &info.VerifyStatus, &verifiedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("验证项不存在: %s", infoUUID)
	}
	if verifiedAt.Valid {
		info.VerifiedAt = &verifiedAt.Time
	}
	if expiresAt.Valid {
		info.ExpiresAt = &expiresAt.Time
	}
	return info, err
}

func (s *Service) markVerified(ctx context.Context, infoUUID string) error {
	now := time.Now()
	expiresAt := now.AddDate(0, 0, 90) // 验证有效期 90 天
	_, err := s.db.ExecContext(ctx,
		`UPDATE extension_infos SET verify_status = 'verified', verified_at = ?, expires_at = ?, updated_at = ? WHERE uuid = ?`,
		now, expiresAt, now, infoUUID,
	)
	return err
}
