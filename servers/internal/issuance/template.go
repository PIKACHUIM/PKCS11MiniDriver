// Package issuance 提供证书颁发模板管理和签发流程。
package issuance

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 是证书颁发模板管理服务。
type Service struct {
	db *storage.DB
}

// NewService 创建颁发模板服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// ---- 颁发模板 CRUD ----

// CreateIssuanceTemplate 创建颁发模板。
func (s *Service) CreateIssuanceTemplate(ctx context.Context, t *storage.IssuanceTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}
	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO issuance_templates (uuid, name, is_ca, path_len, valid_days, allowed_key_types, allowed_ca_uuids,
		 subject_tmpl_uuid, extension_tmpl_uuid, key_usage_tmpl_uuid, key_storage_tmpl_uuid,
		 price_cents, stock, category, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.UUID, t.Name, boolToInt(t.IsCA), t.PathLen, t.ValidDays, t.AllowedKeyTypes, t.AllowedCAUUIDs,
		t.SubjectTmplUUID, t.ExtensionTmplUUID, t.KeyUsageTmplUUID, t.KeyStorageTmplUUID,
		t.PriceCents, t.Stock, t.Category, boolToInt(t.Enabled), t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// GetIssuanceTemplate 按 UUID 查询颁发模板。
func (s *Service) GetIssuanceTemplate(ctx context.Context, tmplUUID string) (*storage.IssuanceTemplate, error) {
	t := &storage.IssuanceTemplate{}
	var isCA, enabled int
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, name, is_ca, path_len, valid_days, allowed_key_types, allowed_ca_uuids,
		 subject_tmpl_uuid, extension_tmpl_uuid, key_usage_tmpl_uuid, key_storage_tmpl_uuid,
		 price_cents, stock, category, enabled, created_at, updated_at
		 FROM issuance_templates WHERE uuid = ?`, tmplUUID,
	).Scan(&t.UUID, &t.Name, &isCA, &t.PathLen, &t.ValidDays, &t.AllowedKeyTypes, &t.AllowedCAUUIDs,
		&t.SubjectTmplUUID, &t.ExtensionTmplUUID, &t.KeyUsageTmplUUID, &t.KeyStorageTmplUUID,
		&t.PriceCents, &t.Stock, &t.Category, &enabled, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("颁发模板不存在: %s", tmplUUID)
	}
	t.IsCA = isCA == 1
	t.Enabled = enabled == 1
	return t, err
}

// ListIssuanceTemplates 查询所有颁发模板。
func (s *Service) ListIssuanceTemplates(ctx context.Context, category string, enabledOnly bool) ([]*storage.IssuanceTemplate, error) {
	query := `SELECT uuid, name, is_ca, path_len, valid_days, allowed_key_types, allowed_ca_uuids,
		 subject_tmpl_uuid, extension_tmpl_uuid, key_usage_tmpl_uuid, key_storage_tmpl_uuid,
		 price_cents, stock, category, enabled, created_at, updated_at
		 FROM issuance_templates WHERE 1=1`
	var args []interface{}

	if category != "" {
		query += ` AND category = ?`
		args = append(args, category)
	}
	if enabledOnly {
		query += ` AND enabled = 1`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*storage.IssuanceTemplate
	for rows.Next() {
		t := &storage.IssuanceTemplate{}
		var isCA, enabled int
		if err := rows.Scan(&t.UUID, &t.Name, &isCA, &t.PathLen, &t.ValidDays, &t.AllowedKeyTypes, &t.AllowedCAUUIDs,
			&t.SubjectTmplUUID, &t.ExtensionTmplUUID, &t.KeyUsageTmplUUID, &t.KeyStorageTmplUUID,
			&t.PriceCents, &t.Stock, &t.Category, &enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.IsCA = isCA == 1
		t.Enabled = enabled == 1
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// UpdateIssuanceTemplate 更新颁发模板。
func (s *Service) UpdateIssuanceTemplate(ctx context.Context, t *storage.IssuanceTemplate) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE issuance_templates SET name = ?, is_ca = ?, path_len = ?, valid_days = ?, allowed_key_types = ?,
		 allowed_ca_uuids = ?, subject_tmpl_uuid = ?, extension_tmpl_uuid = ?, key_usage_tmpl_uuid = ?,
		 key_storage_tmpl_uuid = ?, price_cents = ?, stock = ?, category = ?, enabled = ?, updated_at = ?
		 WHERE uuid = ?`,
		t.Name, boolToInt(t.IsCA), t.PathLen, t.ValidDays, t.AllowedKeyTypes, t.AllowedCAUUIDs,
		t.SubjectTmplUUID, t.ExtensionTmplUUID, t.KeyUsageTmplUUID, t.KeyStorageTmplUUID,
		t.PriceCents, t.Stock, t.Category, boolToInt(t.Enabled), t.UpdatedAt, t.UUID,
	)
	return err
}

// DeleteIssuanceTemplate 删除颁发模板。
func (s *Service) DeleteIssuanceTemplate(ctx context.Context, tmplUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issuance_templates WHERE uuid = ?`, tmplUUID)
	return err
}

// ---- 主体模板 CRUD ----

// CreateSubjectTemplate 创建主体模板。
func (s *Service) CreateSubjectTemplate(ctx context.Context, t *storage.SubjectTemplate) error {
	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO subject_templates (uuid, name, fields, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		t.UUID, t.Name, t.Fields, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ListSubjectTemplates 查询所有主体模板。
func (s *Service) ListSubjectTemplates(ctx context.Context) ([]*storage.SubjectTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, name, fields, created_at, updated_at FROM subject_templates ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*storage.SubjectTemplate
	for rows.Next() {
		t := &storage.SubjectTemplate{}
		if err := rows.Scan(&t.UUID, &t.Name, &t.Fields, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// DeleteSubjectTemplate 删除主体模板。
func (s *Service) DeleteSubjectTemplate(ctx context.Context, tmplUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subject_templates WHERE uuid = ?`, tmplUUID)
	return err
}

// ---- 扩展信息模板 CRUD ----

// CreateExtensionTemplate 创建扩展信息模板。
func (s *Service) CreateExtensionTemplate(ctx context.Context, t *storage.ExtensionTemplate) error {
	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO extension_templates (uuid, name, max_dns, max_email, max_ip, max_uri, require_dns_verify, require_email_verify, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.UUID, t.Name, t.MaxDNS, t.MaxEmail, t.MaxIP, t.MaxURI,
		boolToInt(t.RequireDNSVerify), boolToInt(t.RequireEmailVerify), t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ListExtensionTemplates 查询所有扩展信息模板。
func (s *Service) ListExtensionTemplates(ctx context.Context) ([]*storage.ExtensionTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, name, max_dns, max_email, max_ip, max_uri, require_dns_verify, require_email_verify, created_at, updated_at
		 FROM extension_templates ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*storage.ExtensionTemplate
	for rows.Next() {
		t := &storage.ExtensionTemplate{}
		var dnsVerify, emailVerify int
		if err := rows.Scan(&t.UUID, &t.Name, &t.MaxDNS, &t.MaxEmail, &t.MaxIP, &t.MaxURI,
			&dnsVerify, &emailVerify, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.RequireDNSVerify = dnsVerify == 1
		t.RequireEmailVerify = emailVerify == 1
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// DeleteExtensionTemplate 删除扩展信息模板。
func (s *Service) DeleteExtensionTemplate(ctx context.Context, tmplUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM extension_templates WHERE uuid = ?`, tmplUUID)
	return err
}

// ---- 密钥用途模板 CRUD ----

// CreateKeyUsageTemplate 创建密钥用途模板。
func (s *Service) CreateKeyUsageTemplate(ctx context.Context, t *storage.KeyUsageTemplate) error {
	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO key_usage_templates (uuid, name, key_usage, ext_key_usages, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.UUID, t.Name, t.KeyUsage, t.ExtKeyUsages, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ListKeyUsageTemplates 查询所有密钥用途模板。
func (s *Service) ListKeyUsageTemplates(ctx context.Context) ([]*storage.KeyUsageTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, name, key_usage, ext_key_usages, created_at, updated_at FROM key_usage_templates ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*storage.KeyUsageTemplate
	for rows.Next() {
		t := &storage.KeyUsageTemplate{}
		if err := rows.Scan(&t.UUID, &t.Name, &t.KeyUsage, &t.ExtKeyUsages, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// DeleteKeyUsageTemplate 删除密钥用途模板。
func (s *Service) DeleteKeyUsageTemplate(ctx context.Context, tmplUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM key_usage_templates WHERE uuid = ?`, tmplUUID)
	return err
}

// ---- 证书扩展模板 CRUD ----

// CreateCertExtTemplate 创建证书扩展模板。
func (s *Service) CreateCertExtTemplate(ctx context.Context, t *storage.CertExtTemplate) error {
	t.UUID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cert_ext_templates (uuid, name, crl_dist_points, ocsp_servers, aia_issuers, ct_servers, ev_policy_oid, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.UUID, t.Name, t.CRLDistPoints, t.OCSPServers, t.AIAIssuers, t.CTServers, t.EVPolicyOID, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// ListCertExtTemplates 查询所有证书扩展模板。
func (s *Service) ListCertExtTemplates(ctx context.Context) ([]*storage.CertExtTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, name, crl_dist_points, ocsp_servers, aia_issuers, ct_servers, ev_policy_oid, created_at, updated_at
		 FROM cert_ext_templates ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*storage.CertExtTemplate
	for rows.Next() {
		t := &storage.CertExtTemplate{}
		if err := rows.Scan(&t.UUID, &t.Name, &t.CRLDistPoints, &t.OCSPServers, &t.AIAIssuers, &t.CTServers, &t.EVPolicyOID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// DeleteCertExtTemplate 删除证书扩展模板。
func (s *Service) DeleteCertExtTemplate(ctx context.Context, tmplUUID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM cert_ext_templates WHERE uuid = ?`, tmplUUID)
	return err
}

// ---- 工具函数 ----

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
