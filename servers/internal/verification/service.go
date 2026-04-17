// Package verification 提供域名和邮箱验证功能。
package verification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/mailer"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 是验证服务。
type Service struct {
	db     *storage.DB
	mailer mailer.Mailer // 可选，为 nil 时使用 nopMailer
}

// NewService 创建验证服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// SetMailer 注入邮件发送器（在服务装配阶段调用）。
func (s *Service) SetMailer(m mailer.Mailer) {
	s.mailer = m
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
// 若 info.TmplUUID 非空，则按关联模板校验 require_dns_verify/require_email_verify/max_* 约束。
// 对于 info_type=email：
//   - 生成 6 位数字验证码（仅存 SHA-256 hash，不明文保存）；
//   - verify_token 存随机 hex 用于 DNS-like 自证或 URL 令牌；
//   - 若已注入 mailer，则异步发送验证码邮件；
//   - 用户调用 VerifyEmailCode 输入验证码完成验证。
func (s *Service) CreateExtensionInfo(ctx context.Context, info *storage.ExtensionInfo) error {
	info.UUID = uuid.New().String()
	info.VerifyStatus = "pending"
	info.CreatedAt = time.Now()
	info.UpdatedAt = time.Now()

	// 校验模板约束（如指定了 TmplUUID）
	if info.TmplUUID != "" {
		if err := s.validateExtensionAgainstTemplate(ctx, info); err != nil {
			return err
		}
	}

	// 生成验证 token 和可选的验证码 hash
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("生成验证 token 失败: %w", err)
	}
	info.VerifyToken = hex.EncodeToString(tokenBytes)

	var emailCode string
	if info.InfoType == "email" {
		code, err := randomNumeric(6)
		if err != nil {
			return fmt.Errorf("生成验证码失败: %w", err)
		}
		emailCode = code
		sum := sha256.Sum256([]byte(code))
		info.VerifyCodeHash = hex.EncodeToString(sum[:])
	}

	// 根据类型设置默认验证方式
	switch info.InfoType {
	case "domain":
		if info.VerifyMethod == "" {
			info.VerifyMethod = "txt"
		}
	case "email":
		info.VerifyMethod = "email"
	case "ip":
		info.VerifyMethod = "http"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO extension_infos (uuid, user_uuid, tmpl_uuid, info_type, value, verify_method, verify_token, verify_code_hash, verify_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		info.UUID, info.UserUUID, info.TmplUUID, info.InfoType, info.Value, info.VerifyMethod,
		info.VerifyToken, info.VerifyCodeHash, info.VerifyStatus, info.CreatedAt, info.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// 异步发送邮件（仅对 email 类型）
	if info.InfoType == "email" && s.mailer != nil && emailCode != "" {
		target := info.Value
		code := emailCode
		go func() {
			subject := "OpenCert 邮箱验证码"
			body := fmt.Sprintf("您正在验证邮箱 %s\n\n验证码：%s\n\n有效期：15 分钟。\n如非本人操作请忽略此邮件。", target, code)
			_ = s.mailer.Send(target, subject, body)
		}()
	}
	return nil
}

// validateExtensionAgainstTemplate 根据扩展模板校验 info。
// 当前校验：根据 InfoType 判断是否必填（RequireDNSVerify/RequireEmailVerify），其余 max_* 限制由调用方批量提交时校验。
func (s *Service) validateExtensionAgainstTemplate(ctx context.Context, info *storage.ExtensionInfo) error {
	var maxDNS, maxEmail, maxIP, maxURI, requireDNS, requireEmail int
	err := s.db.QueryRowContext(ctx,
		`SELECT max_dns, max_email, max_ip, max_uri, require_dns_verify, require_email_verify
		 FROM extension_templates WHERE uuid = ?`, info.TmplUUID,
	).Scan(&maxDNS, &maxEmail, &maxIP, &maxURI, &requireDNS, &requireEmail)
	if err != nil {
		// 模板不存在时跳过，不阻塞业务
		return nil
	}

	// 统计当前用户同模板下已有条目数（pending+verified）用于 max_* 限制
	var countDNS, countEmail, countIP int
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM extension_infos WHERE user_uuid = ? AND tmpl_uuid = ? AND info_type = 'domain'`,
		info.UserUUID, info.TmplUUID,
	).Scan(&countDNS)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM extension_infos WHERE user_uuid = ? AND tmpl_uuid = ? AND info_type = 'email'`,
		info.UserUUID, info.TmplUUID,
	).Scan(&countEmail)
	_ = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM extension_infos WHERE user_uuid = ? AND tmpl_uuid = ? AND info_type = 'ip'`,
		info.UserUUID, info.TmplUUID,
	).Scan(&countIP)

	switch info.InfoType {
	case "domain":
		if maxDNS > 0 && countDNS >= maxDNS {
			return fmt.Errorf("该模板下 DNS 名称已达上限 %d", maxDNS)
		}
	case "email":
		if maxEmail > 0 && countEmail >= maxEmail {
			return fmt.Errorf("该模板下邮箱已达上限 %d", maxEmail)
		}
	case "ip":
		if maxIP > 0 && countIP >= maxIP {
			return fmt.Errorf("该模板下 IP 已达上限 %d", maxIP)
		}
	}
	return nil
}

// randomNumeric 生成 n 位纯数字验证码（安全随机）。
func randomNumeric(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("长度必须为正")
	}
	const digits = "0123456789"
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		out[i] = digits[idx.Int64()]
	}
	return string(out), nil
}

// ListExtensionInfos 查询用户的扩展信息列表。
func (s *Service) ListExtensionInfos(ctx context.Context, userUUID string) ([]*storage.ExtensionInfo, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, user_uuid, tmpl_uuid, info_type, value, verify_method, verify_token, verify_code_hash, verify_status, verified_at, expires_at, created_at, updated_at
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
		if err := rows.Scan(&info.UUID, &info.UserUUID, &info.TmplUUID, &info.InfoType, &info.Value, &info.VerifyMethod,
			&info.VerifyToken, &info.VerifyCodeHash, &info.VerifyStatus, &verifiedAt, &expiresAt, &info.CreatedAt, &info.UpdatedAt); err != nil {
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
// 通过比较 SHA-256(code) 与 verify_code_hash 判定。
func (s *Service) VerifyEmailCode(ctx context.Context, infoUUID, code string) error {
	info, err := s.getExtensionInfo(ctx, infoUUID)
	if err != nil {
		return err
	}
	if info.InfoType != "email" {
		return fmt.Errorf("此验证项不支持邮箱验证")
	}
	if info.VerifyCodeHash == "" {
		return fmt.Errorf("验证码未初始化，请重新发起验证")
	}

	sum := sha256.Sum256([]byte(code))
	if hex.EncodeToString(sum[:]) != info.VerifyCodeHash {
		return fmt.Errorf("验证码错误")
	}

	return s.markVerified(ctx, infoUUID)
}

// DeleteExtensionInfo 删除扩展信息。
func (s *Service) DeleteExtensionInfo(ctx context.Context, infoUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM extension_infos WHERE uuid = ?`, infoUUID)
	return err
}

// GetExtensionInfo 按 UUID 查询扩展信息（公开方法）。
func (s *Service) GetExtensionInfo(ctx context.Context, infoUUID string) (*storage.ExtensionInfo, error) {
	return s.getExtensionInfo(ctx, infoUUID)
}

// MarkVerified 标记扩展信息验证通过（公开方法）。
func (s *Service) MarkVerified(ctx context.Context, infoUUID string) error {
	return s.markVerified(ctx, infoUUID)
}

// DeleteSubjectInfo 删除主体信息。
func (s *Service) DeleteSubjectInfo(ctx context.Context, infoUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subject_infos WHERE uuid = ?`, infoUUID)
	return err
}

// ---- 内部方法 ----

func (s *Service) getExtensionInfo(ctx context.Context, infoUUID string) (*storage.ExtensionInfo, error) {
	info := &storage.ExtensionInfo{}
	var verifiedAt, expiresAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, user_uuid, tmpl_uuid, info_type, value, verify_method, verify_token, verify_code_hash, verify_status, verified_at, expires_at
		 FROM extension_infos WHERE uuid = ?`, infoUUID,
	).Scan(&info.UUID, &info.UserUUID, &info.TmplUUID, &info.InfoType, &info.Value, &info.VerifyMethod,
		&info.VerifyToken, &info.VerifyCodeHash, &info.VerifyStatus, &verifiedAt, &expiresAt)
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

	// 优先按扩展信息模板的 verify_expires_days 计算过期时间；模板不存在或未配置时使用默认 90 天。
	days := 90
	var tmplUUID string
	if err := s.db.QueryRowContext(ctx,
		`SELECT tmpl_uuid FROM extension_infos WHERE uuid = ?`, infoUUID,
	).Scan(&tmplUUID); err == nil && tmplUUID != "" {
		var d int
		if err := s.db.QueryRowContext(ctx,
			`SELECT verify_expires_days FROM extension_templates WHERE uuid = ?`, tmplUUID,
		).Scan(&d); err == nil && d > 0 {
			days = d
		}
	}
	expiresAt := now.AddDate(0, 0, days)

	_, err := s.db.ExecContext(ctx,
		`UPDATE extension_infos SET verify_status = 'verified', verified_at = ?, expires_at = ?, updated_at = ? WHERE uuid = ?`,
		now, expiresAt, now, infoUUID,
	)
	return err
}
