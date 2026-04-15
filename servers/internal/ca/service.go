// Package ca 提供 CA 证书颁发机构的管理服务。
package ca

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 是 CA 管理服务。
type Service struct {
	db        *storage.DB
	masterKey []byte // 服务端主密钥，用于加密 CA 私钥
}

// NewService 创建 CA 管理服务。
func NewService(db *storage.DB, masterKey []byte) *Service {
	return &Service{db: db, masterKey: masterKey}
}

// Create 创建 CA（自签名根 CA 或由父 CA 签发的中间 CA）。
func (s *Service) Create(ctx context.Context, ca *storage.CA) error {
	ca.UUID = uuid.New().String()
	ca.CreatedAt = time.Now()
	ca.UpdatedAt = time.Now()
	if ca.Status == "" {
		ca.Status = "active"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cas (uuid, name, cert_pem, private_enc, parent_uuid, status, not_before, not_after, issued_count, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ca.UUID, ca.Name, ca.CertPEM, ca.PrivateEnc, ca.ParentUUID, ca.Status,
		ca.NotBefore, ca.NotAfter, ca.IssuedCount, ca.CreatedAt, ca.UpdatedAt,
	)
	return err
}

// GetByUUID 按 UUID 查询 CA。
func (s *Service) GetByUUID(ctx context.Context, caUUID string) (*storage.CA, error) {
	ca := &storage.CA{}
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, name, cert_pem, private_enc, parent_uuid, status, not_before, not_after, issued_count, created_at, updated_at
		 FROM cas WHERE uuid = ?`, caUUID,
	).Scan(&ca.UUID, &ca.Name, &ca.CertPEM, &ca.PrivateEnc, &ca.ParentUUID, &ca.Status,
		&ca.NotBefore, &ca.NotAfter, &ca.IssuedCount, &ca.CreatedAt, &ca.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("CA 不存在: %s", caUUID)
	}
	return ca, err
}

// List 查询所有 CA。
func (s *Service) List(ctx context.Context) ([]*storage.CA, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, name, cert_pem, parent_uuid, status, not_before, not_after, issued_count, created_at, updated_at
		 FROM cas ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cas []*storage.CA
	for rows.Next() {
		ca := &storage.CA{}
		if err := rows.Scan(&ca.UUID, &ca.Name, &ca.CertPEM, &ca.ParentUUID, &ca.Status,
			&ca.NotBefore, &ca.NotAfter, &ca.IssuedCount, &ca.CreatedAt, &ca.UpdatedAt); err != nil {
			return nil, err
		}
		cas = append(cas, ca)
	}
	return cas, rows.Err()
}

// Update 更新 CA 信息（仅名称和状态）。
func (s *Service) Update(ctx context.Context, caUUID, name, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cas SET name = ?, status = ?, updated_at = ? WHERE uuid = ?`,
		name, status, time.Now(), caUUID,
	)
	return err
}

// Delete 删除 CA（仅允许删除无签发记录的 CA）。
func (s *Service) Delete(ctx context.Context, caUUID string) error {
	ca, err := s.GetByUUID(ctx, caUUID)
	if err != nil {
		return err
	}
	if ca.IssuedCount > 0 {
		return fmt.Errorf("CA 已签发 %d 个证书，无法删除", ca.IssuedCount)
	}

	// 检查是否有子 CA
	var childCount int
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cas WHERE parent_uuid = ?`, caUUID,
	).Scan(&childCount)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return fmt.Errorf("CA 有 %d 个子 CA，无法删除", childCount)
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM cas WHERE uuid = ?`, caUUID)
	return err
}

// RevokeCert 吊销证书（将序列号加入吊销列表）。
func (s *Service) RevokeCert(ctx context.Context, caUUID, serialNumber string, reason int) error {
	// 检查 CA 是否存在
	if _, err := s.GetByUUID(ctx, caUUID); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO revoked_certs (ca_uuid, serial_number, revoked_at, reason)
		 VALUES (?, ?, ?, ?)`,
		caUUID, serialNumber, time.Now(), reason,
	)
	return err
}

// ListRevokedCerts 查询 CA 的吊销证书列表。
func (s *Service) ListRevokedCerts(ctx context.Context, caUUID string) ([]*storage.RevokedCert, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, ca_uuid, serial_number, revoked_at, reason
		 FROM revoked_certs WHERE ca_uuid = ? ORDER BY revoked_at DESC`, caUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*storage.RevokedCert
	for rows.Next() {
		c := &storage.RevokedCert{}
		if err := rows.Scan(&c.ID, &c.CAUUID, &c.SerialNumber, &c.RevokedAt, &c.Reason); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

// GenerateCRL 生成 CRL（证书吊销列表）DER 格式。
func (s *Service) GenerateCRL(ctx context.Context, caUUID string) ([]byte, error) {
	ca, err := s.GetByUUID(ctx, caUUID)
	if err != nil {
		return nil, err
	}

	// 解析 CA 证书
	caCert, err := parseCertPEM(ca.CertPEM)
	if err != nil {
		return nil, fmt.Errorf("解析 CA 证书失败: %w", err)
	}

	// 解密 CA 私钥
	caKey, err := decryptPrivateKey(s.masterKey, ca.PrivateEnc)
	if err != nil {
		return nil, fmt.Errorf("解密 CA 私钥失败: %w", err)
	}

	// 获取吊销列表
	revokedCerts, err := s.ListRevokedCerts(ctx, caUUID)
	if err != nil {
		return nil, fmt.Errorf("获取吊销列表失败: %w", err)
	}

	// 构建 CRL 吊销条目
	revokedEntries := make([]pkix.RevokedCertificate, 0, len(revokedCerts))
	for _, rc := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(rc.SerialNumber, 16)
		revokedEntries = append(revokedEntries, pkix.RevokedCertificate{
			SerialNumber:   serial,
			RevocationTime: rc.RevokedAt,
		})
	}

	// 生成 CRL
	now := time.Now()
	crlTemplate := &x509.RevocationList{
		RevokedCertificateEntries: make([]x509.RevocationListEntry, 0, len(revokedCerts)),
		Number:                    big.NewInt(now.Unix()),
		ThisUpdate:                now,
		NextUpdate:                now.Add(24 * time.Hour), // 默认 24 小时更新
	}

	for _, rc := range revokedCerts {
		serial := new(big.Int)
		serial.SetString(rc.SerialNumber, 16)
		crlTemplate.RevokedCertificateEntries = append(crlTemplate.RevokedCertificateEntries, x509.RevocationListEntry{
			SerialNumber:   serial,
			RevocationTime: rc.RevokedAt,
			ReasonCode:     rc.Reason,
		})
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, crlTemplate, caCert, caKey)
	if err != nil {
		return nil, fmt.Errorf("生成 CRL 失败: %w", err)
	}

	return crlDER, nil
}

// IncrementIssuedCount 递增 CA 的签发计数。
func (s *Service) IncrementIssuedCount(ctx context.Context, caUUID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE cas SET issued_count = issued_count + 1, updated_at = ? WHERE uuid = ?`,
		time.Now(), caUUID,
	)
	return err
}

// ImportChain 导入证书链（PEM 格式，包含多个证书）。
func (s *Service) ImportChain(ctx context.Context, caUUID string, chainPEM string) error {
	// 验证 CA 存在
	if _, err := s.GetByUUID(ctx, caUUID); err != nil {
		return err
	}

	// 解析证书链验证格式
	rest := []byte(chainPEM)
	count := 0
	for len(rest) > 0 {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("证书链中包含非证书类型: %s", block.Type)
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return fmt.Errorf("解析证书链中第 %d 个证书失败: %w", count+1, err)
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("证书链为空")
	}

	// 更新 CA 的证书 PEM（追加证书链）
	_, err := s.db.ExecContext(ctx,
		`UPDATE cas SET cert_pem = cert_pem || ? || ?, updated_at = ? WHERE uuid = ?`,
		"\n", chainPEM, time.Now(), caUUID,
	)
	return err
}
