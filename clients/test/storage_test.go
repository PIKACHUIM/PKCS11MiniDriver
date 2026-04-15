package storage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/globaltrusts/client-card/internal/storage"
)

func setupTestDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "clients-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	db, err := storage.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatal(err)
	}

	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

func TestUserCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := storage.NewUserRepo(db)
	ctx := context.Background()

	// 创建用户
	u := &storage.User{
		UserType:     storage.UserTypeLocal,
		DisplayName:  "测试用户",
		Email:        "test@example.com",
		Enabled:      true,
		PasswordHash: "$2a$12$test",
	}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	if u.UUID == "" {
		t.Fatal("UUID 未生成")
	}

	// 查询用户
	got, err := repo.GetByUUID(ctx, u.UUID)
	if err != nil {
		t.Fatalf("查询用户失败: %v", err)
	}
	if got == nil {
		t.Fatal("用户不存在")
	}
	if got.DisplayName != u.DisplayName {
		t.Errorf("DisplayName 不匹配: got %q, want %q", got.DisplayName, u.DisplayName)
	}

	// 更新用户
	got.DisplayName = "更新后的名称"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("更新用户失败: %v", err)
	}

	// 列出用户
	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("列出用户失败: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("用户数量不对: got %d, want 1", len(users))
	}

	// 删除用户
	if err := repo.Delete(ctx, u.UUID); err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}
	got, _ = repo.GetByUUID(ctx, u.UUID)
	if got != nil {
		t.Fatal("用户未被删除")
	}
}

func TestCardCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := storage.NewUserRepo(db)
	cardRepo := storage.NewCardRepo(db)
	ctx := context.Background()

	// 先创建用户
	u := &storage.User{
		UserType:    storage.UserTypeLocal,
		DisplayName: "卡片测试用户",
		Email:       "card@example.com",
		Enabled:     true,
	}
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatal(err)
	}

	// 创建卡片
	expires := time.Now().Add(365 * 24 * time.Hour)
	c := &storage.Card{
		SlotType:  storage.SlotTypeLocal,
		CardName:  "我的本地卡",
		UserUUID:  u.UUID,
		ExpiresAt: &expires,
		CardKeys: []storage.CardKeyEntry{
			{
				KeyType:      "user",
				UserUUID:     u.UUID,
				Salt:         []byte("test-salt-32bytes-padding-here!!"),
				EncMasterKey: []byte("encrypted-master-key-placeholder"),
			},
		},
		Remark: "测试卡片",
	}
	if err := cardRepo.Create(ctx, c); err != nil {
		t.Fatalf("创建卡片失败: %v", err)
	}

	// 查询卡片
	got, err := cardRepo.GetByUUID(ctx, c.UUID)
	if err != nil {
		t.Fatalf("查询卡片失败: %v", err)
	}
	if got.CardName != c.CardName {
		t.Errorf("CardName 不匹配: got %q, want %q", got.CardName, c.CardName)
	}
	if len(got.CardKeys) != 1 {
		t.Errorf("CardKeys 数量不对: got %d, want 1", len(got.CardKeys))
	}

	// 列出用户卡片
	cards, err := cardRepo.ListByUser(ctx, u.UUID)
	if err != nil {
		t.Fatalf("列出卡片失败: %v", err)
	}
	if len(cards) != 1 {
		t.Errorf("卡片数量不对: got %d, want 1", len(cards))
	}
}

func TestLogWrite(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := storage.NewLogRepo(db)
	ctx := context.Background()

	l := &storage.Log{
		LogType:  storage.LogTypeOperation,
		SlotType: storage.SlotTypeLocal,
		LogLevel: storage.LogLevelInfo,
		Title:    "测试日志",
		Content:  "这是一条测试日志内容",
	}
	if err := repo.Write(ctx, l); err != nil {
		t.Fatalf("写入日志失败: %v", err)
	}
	if l.ID == 0 {
		t.Fatal("日志 ID 未生成")
	}

	logs, err := repo.List(ctx, 10, 0)
	if err != nil {
		t.Fatalf("查询日志失败: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("日志数量不对: got %d, want 1", len(logs))
	}
}
