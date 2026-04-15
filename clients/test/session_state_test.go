package test

import (
	"context"
	"sync"
	"testing"

	"github.com/globaltrusts/client-card/internal/card"
	"github.com/globaltrusts/client-card/pkg/pkcs11types"
)

// mockSlotProvider 是用于测试的 SlotProvider 模拟实现。
type mockSlotProvider struct {
	slotID   pkcs11types.SlotID
	loggedIn bool
	mu       sync.Mutex
}

func newMockSlot(id pkcs11types.SlotID) *mockSlotProvider {
	return &mockSlotProvider{slotID: id}
}

func (m *mockSlotProvider) SlotID() pkcs11types.SlotID { return m.slotID }
func (m *mockSlotProvider) SlotInfo() pkcs11types.SlotInfo {
	return pkcs11types.SlotInfo{SlotID: m.slotID, Description: "Test Slot", TokenPresent: true}
}
func (m *mockSlotProvider) TokenInfo() pkcs11types.TokenInfo {
	return pkcs11types.TokenInfo{Label: "Test Token"}
}
func (m *mockSlotProvider) Mechanisms() []pkcs11types.MechanismType {
	return []pkcs11types.MechanismType{pkcs11types.CKM_RSA_PKCS}
}
func (m *mockSlotProvider) Login(_ context.Context, _ pkcs11types.UserType, pin string) error {
	if pin != "1234" {
		return pkcs11types.CKR_PIN_INCORRECT
	}
	m.mu.Lock()
	m.loggedIn = true
	m.mu.Unlock()
	return nil
}
func (m *mockSlotProvider) Logout(_ context.Context) error {
	m.mu.Lock()
	m.loggedIn = false
	m.mu.Unlock()
	return nil
}
func (m *mockSlotProvider) IsLoggedIn() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loggedIn
}
func (m *mockSlotProvider) FindObjects(_ context.Context, _ []pkcs11types.Attribute) ([]pkcs11types.ObjectHandle, error) {
	return nil, nil
}
func (m *mockSlotProvider) GetAttributes(_ context.Context, _ pkcs11types.ObjectHandle, _ []pkcs11types.AttributeType) ([]pkcs11types.Attribute, error) {
	return nil, nil
}
func (m *mockSlotProvider) Sign(_ context.Context, _ pkcs11types.ObjectHandle, _ pkcs11types.Mechanism, _ []byte) ([]byte, error) {
	return []byte("signature"), nil
}
func (m *mockSlotProvider) Decrypt(_ context.Context, _ pkcs11types.ObjectHandle, _ pkcs11types.Mechanism, _ []byte) ([]byte, error) {
	return []byte("plaintext"), nil
}
func (m *mockSlotProvider) Encrypt(_ context.Context, _ pkcs11types.ObjectHandle, _ pkcs11types.Mechanism, _ []byte) ([]byte, error) {
	return []byte("ciphertext"), nil
}

// ---- 会话状态机测试 ----

func TestOpenSession_ROPublic(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	// 不带 CKF_RW_SESSION 标志 -> RO_PUBLIC
	handle, err := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}

	s, err := mgr.GetSession(handle)
	if err != nil {
		t.Fatalf("GetSession 失败: %v", err)
	}
	if s.State != pkcs11types.CKS_RO_PUBLIC_SESSION {
		t.Errorf("期望状态 RO_PUBLIC(0)，实际 %d", s.State)
	}
}

func TestOpenSession_RWPublic(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	// 带 CKF_RW_SESSION 标志 -> RW_PUBLIC
	handle, err := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	if err != nil {
		t.Fatalf("OpenSession 失败: %v", err)
	}

	s, err := mgr.GetSession(handle)
	if err != nil {
		t.Fatalf("GetSession 失败: %v", err)
	}
	if s.State != pkcs11types.CKS_RW_PUBLIC_SESSION {
		t.Errorf("期望状态 RW_PUBLIC(2)，实际 %d", s.State)
	}
}

func TestLogin_ROPublic_UserLogin_ToROUser(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	ctx := context.Background()

	// RO_PUBLIC + Login(USER) -> RO_USER
	err := mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	s, _ := mgr.GetSession(handle)
	if s.State != pkcs11types.CKS_RO_USER_FUNCTIONS {
		t.Errorf("期望状态 RO_USER(1)，实际 %d", s.State)
	}
}

func TestLogin_ROPublic_SOLogin_Rejected(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	ctx := context.Background()

	// RO_PUBLIC + Login(SO) -> CKR_SESSION_READ_ONLY_EXISTS
	err := mgr.Login(ctx, handle, pkcs11types.CKU_SO, "1234")
	if err != pkcs11types.CKR_SESSION_READ_ONLY_EXISTS {
		t.Errorf("期望 CKR_SESSION_READ_ONLY_EXISTS，实际 %v", err)
	}
}

func TestLogin_RWPublic_UserLogin_ToRWUser(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	ctx := context.Background()

	// RW_PUBLIC + Login(USER) -> RW_USER
	err := mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	s, _ := mgr.GetSession(handle)
	if s.State != pkcs11types.CKS_RW_USER_FUNCTIONS {
		t.Errorf("期望状态 RW_USER(3)，实际 %d", s.State)
	}
}

func TestLogin_RWPublic_SOLogin_ToRWSO(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	ctx := context.Background()

	// RW_PUBLIC + Login(SO) -> RW_SO
	err := mgr.Login(ctx, handle, pkcs11types.CKU_SO, "1234")
	if err != nil {
		t.Fatalf("Login 失败: %v", err)
	}

	s, _ := mgr.GetSession(handle)
	if s.State != pkcs11types.CKS_RW_SO_FUNCTIONS {
		t.Errorf("期望状态 RW_SO(4)，实际 %d", s.State)
	}
}

func TestLogin_AlreadyLoggedIn_Rejected(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	ctx := context.Background()

	// 第一次登录成功
	_ = mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")

	// 第二次登录应被拒绝
	err := mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")
	if err != pkcs11types.CKR_USER_ALREADY_LOGGED_IN {
		t.Errorf("期望 CKR_USER_ALREADY_LOGGED_IN，实际 %v", err)
	}
}

func TestLogout_RWUser_ToRWPublic(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	ctx := context.Background()

	_ = mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")

	// Logout -> 回退到 RW_PUBLIC
	err := mgr.Logout(ctx, handle)
	if err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}

	s, _ := mgr.GetSession(handle)
	if s.State != pkcs11types.CKS_RW_PUBLIC_SESSION {
		t.Errorf("期望状态 RW_PUBLIC(2)，实际 %d", s.State)
	}
}

func TestLogout_ROUser_ToROPublic(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	ctx := context.Background()

	_ = mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")

	// Logout -> 回退到 RO_PUBLIC
	err := mgr.Logout(ctx, handle)
	if err != nil {
		t.Fatalf("Logout 失败: %v", err)
	}

	s, _ := mgr.GetSession(handle)
	if s.State != pkcs11types.CKS_RO_PUBLIC_SESSION {
		t.Errorf("期望状态 RO_PUBLIC(0)，实际 %d", s.State)
	}
}

func TestLogout_NotLoggedIn_Rejected(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	handle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	ctx := context.Background()

	// 未登录时 Logout 应返回错误
	err := mgr.Logout(ctx, handle)
	if err != pkcs11types.CKR_USER_NOT_LOGGED_IN {
		t.Errorf("期望 CKR_USER_NOT_LOGGED_IN，实际 %v", err)
	}
}

func TestSOLogin_WithROSession_Rejected(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	// 先打开一个 RO 会话
	_, _ = mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION)
	// 再打开一个 RW 会话
	rwHandle, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
	ctx := context.Background()

	// 存在 RO 会话时，SO 登录应被拒绝
	err := mgr.Login(ctx, rwHandle, pkcs11types.CKU_SO, "1234")
	if err != pkcs11types.CKR_SESSION_READ_ONLY_EXISTS {
		t.Errorf("期望 CKR_SESSION_READ_ONLY_EXISTS，实际 %v", err)
	}
}

// ---- 并发安全测试 ----

func TestConcurrentOpenSession(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))

	var wg sync.WaitGroup
	handles := make([]pkcs11types.SessionHandle, 100)
	errs := make([]error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			h, err := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
			handles[idx] = h
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: OpenSession 失败: %v", i, err)
		}
	}

	// 验证所有句柄唯一
	seen := make(map[pkcs11types.SessionHandle]bool)
	for _, h := range handles {
		if seen[h] {
			t.Errorf("重复的会话句柄: %d", h)
		}
		seen[h] = true
	}
}

func TestConcurrentLoginLogout(t *testing.T) {
	t.Parallel()
	mgr := card.NewManager()
	mgr.RegisterSlot(newMockSlot(0))
	ctx := context.Background()

	// 打开多个 RW 会话
	var handles []pkcs11types.SessionHandle
	for i := 0; i < 10; i++ {
		h, _ := mgr.OpenSession(0, pkcs11types.CKF_SERIAL_SESSION|pkcs11types.CKF_RW_SESSION)
		handles = append(handles, h)
	}

	// 并发登录（只有第一个应该成功触发 Provider.Login，其余应该通过状态同步）
	var wg sync.WaitGroup
	for _, h := range handles {
		wg.Add(1)
		go func(handle pkcs11types.SessionHandle) {
			defer wg.Done()
			// 可能返回 OK 或 ALREADY_LOGGED_IN，两者都是合法的
			err := mgr.Login(ctx, handle, pkcs11types.CKU_USER, "1234")
			if err != nil && err != pkcs11types.CKR_USER_ALREADY_LOGGED_IN {
				t.Errorf("Login 返回意外错误: %v", err)
			}
		}(h)
	}
	wg.Wait()
}
