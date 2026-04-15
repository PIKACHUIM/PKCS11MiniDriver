// Package workflow 提供证书订单和申请审批工作流。
package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 是工作流服务。
type Service struct {
	db *storage.DB
}

// NewService 创建工作流服务。
func NewService(db *storage.DB) *Service {
	return &Service{db: db}
}

// CreateOrder 创建证书订单。
func (s *Service) CreateOrder(ctx context.Context, o *storage.CertOrder) error {
	o.UUID = uuid.New().String()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	if o.Status == "" {
		o.Status = storage.OrderStatusPending
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cert_orders (uuid, user_uuid, issuance_tmpl_uuid, key_storage_tmpl_uuid, amount_cents, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		o.UUID, o.UserUUID, o.IssuanceTmplUUID, o.KeyStorageTmplUUID, o.AmountCents, o.Status, o.CreatedAt, o.UpdatedAt,
	)
	return err
}

// GetOrder 按 UUID 查询订单。
func (s *Service) GetOrder(ctx context.Context, orderUUID string) (*storage.CertOrder, error) {
	o := &storage.CertOrder{}
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, user_uuid, issuance_tmpl_uuid, key_storage_tmpl_uuid, amount_cents, status, created_at, updated_at
		 FROM cert_orders WHERE uuid = ?`, orderUUID,
	).Scan(&o.UUID, &o.UserUUID, &o.IssuanceTmplUUID, &o.KeyStorageTmplUUID, &o.AmountCents, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("订单不存在: %s", orderUUID)
	}
	return o, err
}

// ListOrders 查询用户的订单列表。
func (s *Service) ListOrders(ctx context.Context, userUUID string, page, pageSize int) ([]*storage.CertOrder, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cert_orders WHERE user_uuid = ?`, userUUID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT uuid, user_uuid, issuance_tmpl_uuid, key_storage_tmpl_uuid, amount_cents, status, created_at, updated_at
		 FROM cert_orders WHERE user_uuid = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userUUID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*storage.CertOrder
	for rows.Next() {
		o := &storage.CertOrder{}
		if err := rows.Scan(&o.UUID, &o.UserUUID, &o.IssuanceTmplUUID, &o.KeyStorageTmplUUID, &o.AmountCents, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

// CreateApplication 提交证书申请。
func (s *Service) CreateApplication(ctx context.Context, app *storage.CertApplication) error {
	app.UUID = uuid.New().String()
	app.Status = "pending"
	app.CreatedAt = time.Now()
	app.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO cert_applications (uuid, order_uuid, user_uuid, subject_json, san_json, key_type, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		app.UUID, app.OrderUUID, app.UserUUID, app.SubjectJSON, app.SANJSON, app.KeyType, app.Status, app.CreatedAt, app.UpdatedAt,
	)
	return err
}

// ListApplications 查询申请列表（管理员查看所有，用户查看自己的）。
func (s *Service) ListApplications(ctx context.Context, userUUID string, statusFilter string, page, pageSize int) ([]*storage.CertApplication, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := `SELECT COUNT(*) FROM cert_applications WHERE 1=1`
	countArgs := []interface{}{}
	if userUUID != "" {
		query += ` AND user_uuid = ?`
		countArgs = append(countArgs, userUUID)
	}
	if statusFilter != "" {
		query += ` AND status = ?`
		countArgs = append(countArgs, statusFilter)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, query, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT uuid, order_uuid, user_uuid, subject_json, san_json, key_type, status, approved_by, approved_at, reject_reason, cert_uuid, created_at, updated_at
		 FROM cert_applications WHERE 1=1`
	selectArgs := []interface{}{}
	if userUUID != "" {
		selectQuery += ` AND user_uuid = ?`
		selectArgs = append(selectArgs, userUUID)
	}
	if statusFilter != "" {
		selectQuery += ` AND status = ?`
		selectArgs = append(selectArgs, statusFilter)
	}
	selectQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	selectArgs = append(selectArgs, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var apps []*storage.CertApplication
	for rows.Next() {
		a := &storage.CertApplication{}
		var approvedBy sql.NullString
		var approvedAt sql.NullTime
		if err := rows.Scan(&a.UUID, &a.OrderUUID, &a.UserUUID, &a.SubjectJSON, &a.SANJSON, &a.KeyType,
			&a.Status, &approvedBy, &approvedAt, &a.RejectReason, &a.CertUUID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if approvedBy.Valid {
			a.ApprovedBy = approvedBy.String
		}
		if approvedAt.Valid {
			a.ApprovedAt = &approvedAt.Time
		}
		apps = append(apps, a)
	}
	return apps, total, rows.Err()
}

// ApproveApplication 审批通过证书申请。
func (s *Service) ApproveApplication(ctx context.Context, appUUID, adminUUID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE cert_applications SET status = 'approved', approved_by = ?, approved_at = ?, updated_at = ? WHERE uuid = ? AND status = 'pending'`,
		adminUUID, now, now, appUUID,
	)
	return err
}

// RejectApplication 拒绝证书申请。
func (s *Service) RejectApplication(ctx context.Context, appUUID, adminUUID, reason string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE cert_applications SET status = 'rejected', approved_by = ?, approved_at = ?, reject_reason = ?, updated_at = ? WHERE uuid = ? AND status = 'pending'`,
		adminUUID, now, reason, now, appUUID,
	)
	return err
}
