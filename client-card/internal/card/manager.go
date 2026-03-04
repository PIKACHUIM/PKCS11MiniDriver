package card

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/globaltrusts/client-card/pkg/pkcs11types"
)

// Manager 管理所有 Slot 和会话。
type Manager struct {
	mu       sync.RWMutex
	slots    map[pkcs11types.SlotID]SlotProvider
	sessions map[pkcs11types.SessionHandle]*Session
	nextSID  atomic.Uint32
}

// NewManager 创建卡片管理器。
func NewManager() *Manager {
	m := &Manager{
		slots:    make(map[pkcs11types.SlotID]SlotProvider),
		sessions: make(map[pkcs11types.SessionHandle]*Session),
	}
	m.nextSID.Store(1)
	return m
}

// RegisterSlot 注册一个 Slot Provider。
func (m *Manager) RegisterSlot(p SlotProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.slots[p.SlotID()] = p
}

// GetSlotList 返回所有 Slot ID 列表。
// tokenPresent=true 时只返回有 Token 的 Slot。
func (m *Manager) GetSlotList(tokenPresent bool) []pkcs11types.SlotID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]pkcs11types.SlotID, 0, len(m.slots))
	for id := range m.slots {
		ids = append(ids, id)
	}
	return ids
}

// GetSlotInfo 返回指定 Slot 的信息。
func (m *Manager) GetSlotInfo(slotID pkcs11types.SlotID) (pkcs11types.SlotInfo, error) {
	m.mu.RLock()
	p, ok := m.slots[slotID]
	m.mu.RUnlock()
	if !ok {
		return pkcs11types.SlotInfo{}, fmt.Errorf("slot %d 不存在", slotID)
	}
	return p.SlotInfo(), nil
}

// GetTokenInfo 返回指定 Slot 的 Token 信息。
func (m *Manager) GetTokenInfo(slotID pkcs11types.SlotID) (pkcs11types.TokenInfo, error) {
	m.mu.RLock()
	p, ok := m.slots[slotID]
	m.mu.RUnlock()
	if !ok {
		return pkcs11types.TokenInfo{}, fmt.Errorf("slot %d 不存在", slotID)
	}
	return p.TokenInfo(), nil
}

// GetMechanisms 返回指定 Slot 支持的算法列表。
func (m *Manager) GetMechanisms(slotID pkcs11types.SlotID) ([]pkcs11types.MechanismType, error) {
	m.mu.RLock()
	p, ok := m.slots[slotID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("slot %d 不存在", slotID)
	}
	return p.Mechanisms(), nil
}

// OpenSession 打开一个新会话，返回会话句柄。
func (m *Manager) OpenSession(slotID pkcs11types.SlotID) (pkcs11types.SessionHandle, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.slots[slotID]
	if !ok {
		return 0, fmt.Errorf("slot %d 不存在", slotID)
	}

	handle := pkcs11types.SessionHandle(m.nextSID.Add(1))
	m.sessions[handle] = &Session{
		Handle:   handle,
		SlotID:   slotID,
		Provider: p,
	}
	return handle, nil
}

// CloseSession 关闭指定会话。
func (m *Manager) CloseSession(handle pkcs11types.SessionHandle) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[handle]; !ok {
		return fmt.Errorf("会话 %d 不存在", handle)
	}
	delete(m.sessions, handle)
	return nil
}

// CloseAllSessions 关闭指定 Slot 的所有会话。
func (m *Manager) CloseAllSessions(slotID pkcs11types.SlotID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for h, s := range m.sessions {
		if s.SlotID == slotID {
			delete(m.sessions, h)
		}
	}
}

// GetSession 获取会话，若不存在返回错误。
func (m *Manager) GetSession(handle pkcs11types.SessionHandle) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[handle]
	if !ok {
		return nil, fmt.Errorf("会话 %d 不存在或已关闭", handle)
	}
	return s, nil
}

// Login 在指定会话上执行登录。
func (m *Manager) Login(ctx context.Context, handle pkcs11types.SessionHandle, userType pkcs11types.UserType, pin string) error {
	s, err := m.GetSession(handle)
	if err != nil {
		return err
	}
	return s.Provider.Login(ctx, userType, pin)
}

// Logout 在指定会话上执行登出。
func (m *Manager) Logout(ctx context.Context, handle pkcs11types.SessionHandle) error {
	s, err := m.GetSession(handle)
	if err != nil {
		return err
	}
	return s.Provider.Logout(ctx)
}
