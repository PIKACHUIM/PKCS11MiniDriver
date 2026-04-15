// Package storage_test 提供 clients 数据库层的单元测试。
package storage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/globaltrusts/client-card/internal/storage"
)

// setupTempDB 创建临时文件数据库，返回 DB 和清理函数。
func setupTempDB(t *testing.T) (*storage.DB, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "clients-sqlcipher-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()

	db, err := storage.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		t.Fatalf("打开数据库失败: %v", err)
	}
	return db, func() {
		db.Close()
		os.Remove(f.Name())
	}
}

// setupTempEncryptedDB 创建临时加密数据库，返回 DB 和清理函数。
func setupTempEncryptedDB(t *testing.T, key string) (*storage.DB, string, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "clients-enc-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	path := f.Name()

	db, err := storage.OpenEncrypted(path, key)
	if err != nil {
		os.Remove(path)
		t.Fatalf("打开加密数据库失败: %v", err)
	}
	return db, path, func() {
		db.Close()
		os.Remove(path)
	}
}

// ---- 基本打开与迁移测试 ----

func TestOpenPlainDB(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTempDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("Open() 返回 nil")
	}
	if db.IsEncrypted() {
		t.Error("明文数据库 IsEncrypted() 应为 false")
	}

	// 验证连接可用
	if err := db.Conn().Ping(); err != nil {
		t.Fatalf("数据库 Ping 失败: %v", err)
	}
}

func TestOpenEncryptedDB(t *testing.T) {
	t.Parallel()
	db, _, cleanup := setupTempEncryptedDB(t, "test-encrypt-key-32bytes-padding")
	defer cleanup()

	if !db.IsEncrypted() {
		t.Error("加密数据库 IsEncrypted() 应为 true")
	}
}

func TestOpenMemoryDB(t *testing.T) {
	t.Parallel()
	// 内存数据库（:memory:）
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) 失败: %v", err)
	}
	defer db.Close()

	if db.IsEncrypted() {
		t.Error("内存数据库 IsEncrypted() 应为 false")
	}
}

// ---- 用户表读写测试 ----

func TestUserReadWrite(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTempDB(t)
	defer cleanup()

	repo := storage.NewUserRepo(db)
	ctx := context.Background()

	tests := []struct {
		name string
		user *storage.User
	}{
		{
			name: "本地用户",
			user: &storage.User{
				UserType:     storage.UserTypeLocal,
				DisplayName:  "本地测试用户",
				Email:        "local@test.com",
				Enabled:      true,
				PasswordHash: "$2a$12$testhashabcdefghijklmno",
			},
		},
		{
			name: "云端用户",
			user: &storage.User{
				UserType:    storage.UserTypeCloud,
				DisplayName: "云端测试用户",
				Email:       "cloud@test.com",
				Enabled:     true,
				CloudURL:    "https://server.example.com",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// 注意：用户创建有唯一约束，不能并行
			if err := repo.Create(ctx, tt.user); err != nil {
				t.Fatalf("Create() 失败: %v", err)
			}
			if tt.user.UUID == "" {
				t.Fatal("UUID 未生成")
			}

			got, err := repo.GetByUUID(ctx, tt.user.UUID)
			if err != nil {
				t.Fatalf("GetByUUID() 失败: %v", err)
			}
			if got.DisplayName != tt.user.DisplayName {
				t.Errorf("DisplayName = %q，期望 %q", got.DisplayName, tt.user.DisplayName)
			}
			if got.UserType != tt.user.UserType {
				t.Errorf("UserType = %q，期望 %q", got.UserType, tt.user.UserType)
			}
		})
	}
}

// ---- 卡片表读写测试 ----

func TestCardReadWrite(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTempDB(t)
	defer cleanup()

	userRepo := storage.NewUserRepo(db)
	cardRepo := storage.NewCardRepo(db)
	ctx := context.Background()

	// 先创建用户
	u := &storage.User{
		UserType:     storage.UserTypeLocal,
		DisplayName:  "卡片测试用户",
		Email:        "cardtest@test.com",
		Enabled:      true,
		PasswordHash: "$2a$12$test",
	}
	if err := userRepo.Create(ctx, u); err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	expires := time.Now().Add(365 * 24 * time.Hour)
	card := &storage.Card{
		SlotType:  storage.SlotTypeLocal,
		CardName:  "测试本地卡",
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
		Remark: "单元测试卡片",
	}

	if err := cardRepo.Create(ctx, card); err != nil {
		t.Fatalf("Create() 失败: %v", err)
	}
	if card.UUID == "" {
		t.Fatal("UUID 未生成")
	}

	// 查询
	got, err := cardRepo.GetByUUID(ctx, card.UUID)
	if err != nil {
		t.Fatalf("GetByUUID() 失败: %v", err)
	}
	if got.CardName != card.CardName {
		t.Errorf("CardName = %q，期望 %q", got.CardName, card.CardName)
	}
	if len(got.CardKeys) != 1 {
		t.Errorf("CardKeys 数量 = %d，期望 1", len(got.CardKeys))
	}
	if got.CardKeys[0].KeyType != "user" {
		t.Errorf("CardKeys[0].KeyType = %q，期望 user", got.CardKeys[0].KeyType)
	}

	// 按用户列出
	cards, err := cardRepo.ListByUser(ctx, u.UUID)
	if err != nil {
		t.Fatalf("ListByUser() 失败: %v", err)
	}
	if len(cards) != 1 {
		t.Errorf("ListByUser() 返回 %d 个，期望 1", len(cards))
	}
}

// ---- 数据持久化测试（关闭后重新打开）----

func TestDataPersistence(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp("", "clients-persist-*.db")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	f.Close()
	path := f.Name()
	defer os.Remove(path)

	// 第一次打开，写入数据
	{
		db, err := storage.Open(path)
		if err != nil {
			t.Fatalf("第一次打开数据库失败: %v", err)
		}
		repo := storage.NewUserRepo(db)
		u := &storage.User{
			UserType:     storage.UserTypeLocal,
			DisplayName:  "持久化测试用户",
			Email:        "persist@test.com",
			Enabled:      true,
			PasswordHash: "$2a$12$test",
		}
		if err := repo.Create(context.Background(), u); err != nil {
			db.Close()
			t.Fatalf("写入数据失败: %v", err)
		}
		db.Close()
	}

	// 第二次打开，验证数据存在
	{
		db, err := storage.Open(path)
		if err != nil {
			t.Fatalf("第二次打开数据库失败: %v", err)
		}
		defer db.Close()

		repo := storage.NewUserRepo(db)
		users, err := repo.List(context.Background())
		if err != nil {
			t.Fatalf("查询用户失败: %v", err)
		}
		if len(users) != 1 {
			t.Errorf("重新打开后用户数量 = %d，期望 1", len(users))
		}
		if users[0].DisplayName != "持久化测试用户" {
			t.Errorf("DisplayName = %q，期望 %q", users[0].DisplayName, "持久化测试用户")
		}
	}
}

// ---- 并发读写安全测试 ----

func TestConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	db, cleanup := setupTempDB(t)
	defer cleanup()

	repo := storage.NewUserRepo(db)
	ctx := context.Background()

	// 并发创建多个用户
	const n = 10
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			u := &storage.User{
				UserType:     storage.UserTypeLocal,
				DisplayName:  "并发用户",
				Email:        "concurrent" + string(rune('0'+i)) + "@test.com",
				Enabled:      true,
				PasswordHash: "$2a$12$test",
			}
			done <- repo.Create(ctx, u)
		}()
	}

	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Errorf("并发创建用户失败: %v", err)
		}
	}

	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() 失败: %v", err)
	}
	if len(users) != n {
		t.Errorf("并发创建后用户数量 = %d，期望 %d", len(users), n)
	}
}

// ---- 加密数据库标志测试 ----

func TestEncryptedFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		key           string
		wantEncrypted bool
	}{
		{"无密钥（明文）", "", false},
		{"有密钥（加密）", "my-secret-key-32bytes-padding!!!", true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f, _ := os.CreateTemp("", "enc-flag-*.db")
			f.Close()
			defer os.Remove(f.Name())

			var db *storage.DB
			var err error
			if tt.key == "" {
				db, err = storage.Open(f.Name())
			} else {
				db, err = storage.OpenEncrypted(f.Name(), tt.key)
			}
			if err != nil {
				t.Fatalf("打开数据库失败: %v", err)
			}
			defer db.Close()

			if db.IsEncrypted() != tt.wantEncrypted {
				t.Errorf("IsEncrypted() = %v，期望 %v", db.IsEncrypted(), tt.wantEncrypted)
			}
		})
	}
}
