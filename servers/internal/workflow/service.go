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

// CreateOrder 创建证书订单（检查余额并冻结金额）。
func (s *Service) CreateOrder(ctx context.Context, o *storage.CertOrder) error {
	o.UUID = uuid.New().String()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	if o.Status == "" {
		o.Status = storage.CertOrderPendingPayment
	}

	// 如果有定价，检查用户余额
	if o.AmountCents > 0 {
		var availableCents int64
		err := s.db.QueryRowContext(ctx,
			`SELECT available_cents FROM user_balances WHERE user_uuid = ?`, o.UserUUID,
		).Scan(&availableCents)
		if err != nil {
			return fmt.Errorf("查询余额失败，请先充值")
		}
		if availableCents < o.AmountCents {
			return fmt.Errorf("余额不足，当前余额 %d 分，需要 %d 分", availableCents, o.AmountCents)
		}

		// 冻结金额（扣减可用余额，增加冻结余额）
		_, err = s.db.ExecContext(ctx,
			`UPDATE user_balances SET available_cents = available_cents - ?, frozen_cents = frozen_cents + ? WHERE user_uuid = ?`,
			o.AmountCents, o.AmountCents, o.UserUUID,
		)
		if err != nil {
			return fmt.Errorf("冻结金额失败: %w", err)
		}
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
// 审批通过后自动更新订单状态，将冻结金额转为消费。
func (s *Service) ApproveApplication(ctx context.Context, appUUID, adminUUID string) error {
	now := time.Now()

	// 查询申请信息
	app := &storage.CertApplication{}
	var approvedBy sql.NullString
	var approvedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT uuid, order_uuid, user_uuid, subject_json, san_json, key_type, status
		 FROM cert_applications WHERE uuid = ?`, appUUID,
	).Scan(&app.UUID, &app.OrderUUID, &app.UserUUID, &app.SubjectJSON, &app.SANJSON, &app.KeyType, &app.Status)
	if err != nil {
		return fmt.Errorf("查询申请失败: %w", err)
	}
	_ = approvedBy
	_ = approvedAt

	if app.Status != "pending" {
		return fmt.Errorf("只能审批待处理的申请，当前状态: %s", app.Status)
	}

	// 更新申请状态为 approved
	_, err = s.db.ExecContext(ctx,
		`UPDATE cert_applications SET status = 'approved', approved_by = ?, approved_at = ?, updated_at = ? WHERE uuid = ?`,
		adminUUID, now, now, appUUID,
	)
	if err != nil {
		return fmt.Errorf("更新申请状态失败: %w", err)
	}

	// 更新关联订单状态为 issued
	if app.OrderUUID != "" {
		_, err = s.db.ExecContext(ctx,
			`UPDATE cert_orders SET status = 'issued', updated_at = ? WHERE uuid = ?`,
			now, app.OrderUUID,
		)
		if err != nil {
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		// 查询订单金额，将冻结金额转为消费
		var orderAmount int64
		var orderUserUUID string
		err = s.db.QueryRowContext(ctx,
			`SELECT user_uuid, amount_cents FROM cert_orders WHERE uuid = ?`, app.OrderUUID,
		).Scan(&orderUserUUID, &orderAmount)
		if err == nil && orderAmount > 0 {
			// 扣减冻结金额
			s.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE user_balances SET frozen_cents = frozen_cents - ? WHERE user_uuid = ?`,
				orderAmount, orderUserUUID,
			)
			// 写入消费记录
			consumeUUID := uuid.New().String()
			s.db.ExecContext(ctx, //nolint:errcheck
				`INSERT INTO consume_records (uuid, user_uuid, order_no, consume_type, amount_cents, remark, created_at)
				 VALUES (?, ?, ?, 'cert_purchase', ?, '证书签发消费', ?)`,
				consumeUUID, orderUserUUID, app.OrderUUID, orderAmount, now,
			)
		}
	}

	return nil
}

// RejectApplication 拒绝证书申请（解冻冻结金额）。
func (s *Service) RejectApplication(ctx context.Context, appUUID, adminUUID, reason string) error {
	now := time.Now()

	// 查询申请信息
	var orderUUID, userUUID, status string
	err := s.db.QueryRowContext(ctx,
		`SELECT order_uuid, user_uuid, status FROM cert_applications WHERE uuid = ?`, appUUID,
	).Scan(&orderUUID, &userUUID, &status)
	if err != nil {
		return fmt.Errorf("查询申请失败: %w", err)
	}

	if status != "pending" {
		return fmt.Errorf("只能拒绝待处理的申请，当前状态: %s", status)
	}

	// 更新申请状态为 rejected
	_, err = s.db.ExecContext(ctx,
		`UPDATE cert_applications SET status = 'rejected', approved_by = ?, approved_at = ?, reject_reason = ?, updated_at = ? WHERE uuid = ? AND status = 'pending'`,
		adminUUID, now, reason, now, appUUID,
	)
	if err != nil {
		return fmt.Errorf("更新申请状态失败: %w", err)
	}

	// 解冻冻结金额
	if orderUUID != "" {
		var orderAmount int64
		err = s.db.QueryRowContext(ctx,
			`SELECT amount_cents FROM cert_orders WHERE uuid = ?`, orderUUID,
		).Scan(&orderAmount)
		if err == nil && orderAmount > 0 {
			// 解冻金额退回可用余额
			s.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE user_balances SET available_cents = available_cents + ?, frozen_cents = frozen_cents - ? WHERE user_uuid = ?`,
				orderAmount, orderAmount, userUUID,
			)
			// 更新订单状态为 rejected
			s.db.ExecContext(ctx, //nolint:errcheck
				`UPDATE cert_orders SET status = 'rejected', updated_at = ? WHERE uuid = ?`,
				now, orderUUID,
			)
		}
	}

	return nil
}

// PayOrder 标记订单为已支付（模拟支付，实际支付由 payment 模块处理）。
func (s *Service) PayOrder(ctx context.Context, orderUUID, userUUID string) error {
	now := time.Now()
	result, err := s.db.ExecContext(ctx,
		`UPDATE cert_orders SET status = ?, paid_at = ?, updated_at = ? WHERE uuid = ? AND user_uuid = ? AND status = ?`,
		string(storage.CertOrderPaid), now, now, orderUUID, userUUID, string(storage.CertOrderPendingPayment),
	)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("订单不存在或状态不允许支付")
	}
	return nil
}

// SubmitCertApplication 提交证书申请（订单状态 paid → applying/reviewing）。
func (s *Service) SubmitCertApplication(ctx context.Context, app *storage.CertApplication, requireApproval bool) error {
	now := time.Now()

	newStatus := string(storage.CertOrderApplying)
	if requireApproval {
		newStatus = string(storage.CertOrderReviewing)
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE cert_orders SET status = ?, updated_at = ? WHERE uuid = ? AND status = ?`,
		newStatus, now, app.OrderUUID, string(storage.CertOrderPaid),
	)
	if err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	return s.CreateApplication(ctx, app)
}

// CancelOrder 取消订单（解冻金额）。
func (s *Service) CancelOrder(ctx context.Context, orderUUID, userUUID string) error {
	now := time.Now()

	order, err := s.GetOrder(ctx, orderUUID)
	if err != nil {
		return err
	}
	if order.UserUUID != userUUID {
		return fmt.Errorf("无权取消此订单")
	}
	if order.Status == storage.CertOrderCompleted || order.Status == storage.CertOrderCancelled {
		return fmt.Errorf("订单状态不允许取消: %s", order.Status)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE cert_orders SET status = ?, updated_at = ? WHERE uuid = ?`,
		string(storage.CertOrderCancelled), now, orderUUID,
	)
	if err != nil {
		return fmt.Errorf("取消订单失败: %w", err)
	}

	if order.FrozenCents > 0 {
		s.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE user_balances SET available_cents = available_cents + ?, frozen_cents = frozen_cents - ? WHERE user_uuid = ?`,
			order.FrozenCents, order.FrozenCents, userUUID,
		)
		refundUUID := uuid.New().String()
		s.db.ExecContext(ctx, //nolint:errcheck
			`INSERT INTO consume_records (uuid, user_uuid, order_no, consume_type, amount_cents, remark, created_at)
			 VALUES (?, ?, ?, 'refund', ?, '订单取消退款', ?)`,
			refundUUID, userUUID, orderUUID, -order.FrozenCents, now,
		)
	}
	return nil
}

// CompleteOrder 完成订单（签发成功后调用）。
func (s *Service) CompleteOrder(ctx context.Context, orderUUID, certUUID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE cert_orders SET status = ?, updated_at = ? WHERE uuid = ?`,
		string(storage.CertOrderCompleted), now, orderUUID,
	)
	if err != nil {
		return fmt.Errorf("完成订单失败: %w", err)
	}
	if certUUID != "" {
		s.db.ExecContext(ctx, //nolint:errcheck
			`UPDATE cert_applications SET cert_uuid = ?, status = 'approved', updated_at = ? WHERE order_uuid = ?`,
			certUUID, now, orderUUID,
		)
	}
	return nil
}
