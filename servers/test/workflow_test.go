// Package test 提供 servers 的集成测试。
package test

import (
	"context"
	"testing"

	"github.com/globaltrusts/server-card/internal/storage"
	"github.com/globaltrusts/server-card/internal/workflow"
)

// createTestUser 在测试数据库中创建一个测试用户，返回 UUID。
// email 同时作为 Username（唯一），确保并行测试不冲突。
func createTestUser(t *testing.T, db *storage.DB, email string) string {
	t.Helper()
	ctx := context.Background()
	repo := storage.NewUserRepo(db)
	u := &storage.User{
		Username:     email, // servers users 表 username 有 UNIQUE 约束
		DisplayName:  "测试用户",
		Email:        email,
		PasswordHash: "$2a$12$testhashabcdefghijklmno",
		Role:         "user",
		Enabled:      true,
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("创建测试用户失败 (email=%s): %v", email, err)
	}
	return u.UUID
}

func TestWorkflowCreateOrder(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "order@test.com")

	tests := []struct {
		name    string
		order   *storage.CertOrder
		wantErr bool
	}{
		{
			name: "创建有效订单",
			order: &storage.CertOrder{
				UserUUID:           userUUID,
				IssuanceTmplUUID:   "tmpl-uuid-1",
				KeyStorageTmplUUID: "ks-tmpl-uuid-1",
				AmountCents:        100,
			},
			wantErr: false,
		},
		{
			name: "创建零金额订单",
			order: &storage.CertOrder{
				UserUUID:         userUUID,
				IssuanceTmplUUID: "tmpl-uuid-2",
				AmountCents:      0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := svc.CreateOrder(ctx, tt.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.order.UUID == "" {
					t.Error("CreateOrder() 未生成 UUID")
				}
				if tt.order.Status != storage.OrderStatusPending {
					t.Errorf("初始状态应为 pending，got %q", tt.order.Status)
				}
			}
		})
	}
}

func TestWorkflowGetOrder(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "getorder@test.com")

	order := &storage.CertOrder{
		UserUUID:         userUUID,
		IssuanceTmplUUID: "tmpl-uuid",
		AmountCents:      200,
	}
	if err := svc.CreateOrder(ctx, order); err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}

	// 查询存在的订单
	got, err := svc.GetOrder(ctx, order.UUID)
	if err != nil {
		t.Fatalf("GetOrder() 失败: %v", err)
	}
	if got.AmountCents != 200 {
		t.Errorf("AmountCents = %d，期望 200", got.AmountCents)
	}

	// 查询不存在的订单
	if _, err := svc.GetOrder(ctx, "non-existent"); err == nil {
		t.Error("查询不存在的订单应返回错误")
	}
}

func TestWorkflowListOrders(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "listorder@test.com")

	// 创建 3 个订单
	for i := 0; i < 3; i++ {
		o := &storage.CertOrder{
			UserUUID:         userUUID,
			IssuanceTmplUUID: "tmpl",
			AmountCents:      int64(100 * (i + 1)),
		}
		if err := svc.CreateOrder(ctx, o); err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
	}

	orders, total, err := svc.ListOrders(ctx, userUUID, 1, 10)
	if err != nil {
		t.Fatalf("ListOrders() 失败: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d，期望 3", total)
	}
	if len(orders) != 3 {
		t.Errorf("len(orders) = %d，期望 3", len(orders))
	}
}

func TestWorkflowApplicationApprove(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "approve@test.com")
	adminUUID := createTestUser(t, db, "admin@test.com")

	// 创建订单
	order := &storage.CertOrder{
		UserUUID:         userUUID,
		IssuanceTmplUUID: "tmpl",
		AmountCents:      100,
	}
	if err := svc.CreateOrder(ctx, order); err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}

	// 提交申请
	app := &storage.CertApplication{
		OrderUUID:   order.UUID,
		UserUUID:    userUUID,
		SubjectJSON: `{"CN":"test.example.com"}`,
		SANJSON:     `{"dns":["test.example.com"]}`,
		KeyType:     "ec256",
	}
	if err := svc.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication() 失败: %v", err)
	}
	if app.UUID == "" {
		t.Fatal("申请 UUID 未生成")
	}
	if app.Status != "pending" {
		t.Errorf("初始状态应为 pending，got %q", app.Status)
	}

	// 管理员审批通过
	if err := svc.ApproveApplication(ctx, app.UUID, adminUUID); err != nil {
		t.Fatalf("ApproveApplication() 失败: %v", err)
	}

	// 验证状态已更新
	apps, _, err := svc.ListApplications(ctx, userUUID, "approved", 1, 10)
	if err != nil {
		t.Fatalf("ListApplications() 失败: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("审批后 approved 申请数量 = %d，期望 1", len(apps))
	}
	if apps[0].ApprovedBy != adminUUID {
		t.Errorf("ApprovedBy = %q，期望 %q", apps[0].ApprovedBy, adminUUID)
	}
}

func TestWorkflowApplicationReject(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "reject@test.com")
	adminUUID := createTestUser(t, db, "admin2@test.com")

	order := &storage.CertOrder{
		UserUUID:         userUUID,
		IssuanceTmplUUID: "tmpl",
		AmountCents:      100,
	}
	if err := svc.CreateOrder(ctx, order); err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}

	app := &storage.CertApplication{
		OrderUUID:   order.UUID,
		UserUUID:    userUUID,
		SubjectJSON: `{"CN":"bad.example.com"}`,
		SANJSON:     `{}`,
		KeyType:     "ec256",
	}
	if err := svc.CreateApplication(ctx, app); err != nil {
		t.Fatalf("CreateApplication() 失败: %v", err)
	}

	// 管理员拒绝
	rejectReason := "主体信息不符合要求"
	if err := svc.RejectApplication(ctx, app.UUID, adminUUID, rejectReason); err != nil {
		t.Fatalf("RejectApplication() 失败: %v", err)
	}

	// 验证状态
	apps, _, err := svc.ListApplications(ctx, userUUID, "rejected", 1, 10)
	if err != nil {
		t.Fatalf("ListApplications() 失败: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("拒绝后 rejected 申请数量 = %d，期望 1", len(apps))
	}
	if apps[0].RejectReason != rejectReason {
		t.Errorf("RejectReason = %q，期望 %q", apps[0].RejectReason, rejectReason)
	}
}

func TestWorkflowApplicationListFilter(t *testing.T) {
	t.Parallel()
	db, cleanup := setupServerDB(t)
	defer cleanup()

svc := workflow.NewService(db, nil, nil)
	ctx := context.Background()
	userUUID := createTestUser(t, db, "filter@test.com")
	adminUUID := createTestUser(t, db, "admin3@test.com")

	// 创建 3 个申请：1 pending, 1 approved, 1 rejected
	for i := 0; i < 3; i++ {
		order := &storage.CertOrder{
			UserUUID: userUUID, IssuanceTmplUUID: "tmpl", AmountCents: 100,
		}
		if err := svc.CreateOrder(ctx, order); err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		app := &storage.CertApplication{
			OrderUUID: order.UUID, UserUUID: userUUID,
			SubjectJSON: `{}`, SANJSON: `{}`, KeyType: "ec256",
		}
		if err := svc.CreateApplication(ctx, app); err != nil {
			t.Fatalf("创建申请失败: %v", err)
		}
		switch i {
		case 1:
			svc.ApproveApplication(ctx, app.UUID, adminUUID)
		case 2:
			svc.RejectApplication(ctx, app.UUID, adminUUID, "测试拒绝")
		}
	}

	// 查询所有
	all, total, err := svc.ListApplications(ctx, userUUID, "", 1, 10)
	if err != nil {
		t.Fatalf("ListApplications(all) 失败: %v", err)
	}
	if total != 3 || len(all) != 3 {
		t.Errorf("全部申请: total=%d, len=%d，期望各为 3", total, len(all))
	}

	// 仅查询 pending
	pending, pendingTotal, _ := svc.ListApplications(ctx, userUUID, "pending", 1, 10)
	if pendingTotal != 1 || len(pending) != 1 {
		t.Errorf("pending 申请: total=%d, len=%d，期望各为 1", pendingTotal, len(pending))
	}
}
