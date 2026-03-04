package ipc

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/globaltrusts/client-card/internal/card"
	"github.com/globaltrusts/client-card/pkg/pkcs11types"
)

// Handler 注册所有 PKCS#11 命令处理函数到 IPC Server。
type PKCSHandler struct {
	manager *card.Manager
}

// NewPKCSHandler 创建 PKCS#11 命令处理器。
func NewPKCSHandler(manager *card.Manager) *PKCSHandler {
	return &PKCSHandler{manager: manager}
}

// Register 将所有命令处理函数注册到 server。
func (h *PKCSHandler) Register(s *Server) {
	s.Register(CmdGetSlotList, h.handleGetSlotList)
	s.Register(CmdGetSlotInfo, h.handleGetSlotInfo)
	s.Register(CmdGetTokenInfo, h.handleGetTokenInfo)
	s.Register(CmdGetMechanismList, h.handleGetMechanismList)
	s.Register(CmdOpenSession, h.handleOpenSession)
	s.Register(CmdCloseSession, h.handleCloseSession)
	s.Register(CmdCloseAllSessions, h.handleCloseAllSessions)
	s.Register(CmdLogin, h.handleLogin)
	s.Register(CmdLogout, h.handleLogout)
	s.Register(CmdFindObjectsInit, h.handleFindObjectsInit)
	s.Register(CmdFindObjects, h.handleFindObjects)
	s.Register(CmdFindObjectsFinal, h.handleFindObjectsFinal)
	s.Register(CmdGetAttributeValue, h.handleGetAttributeValue)
	s.Register(CmdSignInit, h.handleSignInit)
	s.Register(CmdSign, h.handleSign)
	s.Register(CmdDecryptInit, h.handleDecryptInit)
	s.Register(CmdDecrypt, h.handleDecrypt)
	s.Register(CmdEncryptInit, h.handleEncryptInit)
	s.Register(CmdEncrypt, h.handleEncrypt)
}

// ---- GetSlotList ----

type getSlotListReq struct {
	TokenPresent bool `json:"token_present"`
}

type getSlotListResp struct {
	SlotIDs []uint32 `json:"slot_ids"`
}

func (h *PKCSHandler) handleGetSlotList(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r getSlotListReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		slog.Warn("GetSlotList 解析请求失败", "error", err)
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	ids := h.manager.GetSlotList(r.TokenPresent)
	result := make([]uint32, len(ids))
	for i, id := range ids {
		result[i] = uint32(id)
	}
	return &getSlotListResp{SlotIDs: result}, uint32(pkcs11types.CKR_OK)
}

// ---- GetSlotInfo ----

type slotIDReq struct {
	SlotID uint32 `json:"slot_id"`
}

func (h *PKCSHandler) handleGetSlotInfo(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r slotIDReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	info, err := h.manager.GetSlotInfo(pkcs11types.SlotID(r.SlotID))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SLOT_ID_INVALID)
	}
	return &info, uint32(pkcs11types.CKR_OK)
}

// ---- GetTokenInfo ----

func (h *PKCSHandler) handleGetTokenInfo(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r slotIDReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	info, err := h.manager.GetTokenInfo(pkcs11types.SlotID(r.SlotID))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SLOT_ID_INVALID)
	}
	return &info, uint32(pkcs11types.CKR_OK)
}

// ---- GetMechanismList ----

type getMechListResp struct {
	Mechanisms []uint32 `json:"mechanisms"`
}

func (h *PKCSHandler) handleGetMechanismList(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r slotIDReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	mechs, err := h.manager.GetMechanisms(pkcs11types.SlotID(r.SlotID))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SLOT_ID_INVALID)
	}
	result := make([]uint32, len(mechs))
	for i, m := range mechs {
		result[i] = uint32(m)
	}
	return &getMechListResp{Mechanisms: result}, uint32(pkcs11types.CKR_OK)
}

// ---- OpenSession ----

type openSessionReq struct {
	SlotID uint32 `json:"slot_id"`
	Flags  uint32 `json:"flags"`
}

type openSessionResp struct {
	SessionHandle uint32 `json:"session_handle"`
}

func (h *PKCSHandler) handleOpenSession(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r openSessionReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	handle, err := h.manager.OpenSession(pkcs11types.SlotID(r.SlotID))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SLOT_ID_INVALID)
	}
	return &openSessionResp{SessionHandle: uint32(handle)}, uint32(pkcs11types.CKR_OK)
}

// ---- CloseSession ----

type sessionReq struct {
	SessionHandle uint32 `json:"session_handle"`
}

func (h *PKCSHandler) handleCloseSession(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r sessionReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	if err := h.manager.CloseSession(pkcs11types.SessionHandle(r.SessionHandle)); err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- CloseAllSessions ----

func (h *PKCSHandler) handleCloseAllSessions(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r slotIDReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}
	h.manager.CloseAllSessions(pkcs11types.SlotID(r.SlotID))
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- Login ----

type loginReq struct {
	SessionHandle uint32 `json:"session_handle"`
	UserType      uint32 `json:"user_type"`
	PIN           string `json:"pin"`
}

func (h *PKCSHandler) handleLogin(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r loginReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	err := h.manager.Login(ctx, pkcs11types.SessionHandle(r.SessionHandle),
		pkcs11types.UserType(r.UserType), r.PIN)
	if err != nil {
		slog.Warn("Login 失败", "session", r.SessionHandle, "error", err)
		return nil, uint32(pkcs11types.CKR_PIN_INCORRECT)
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- Logout ----

func (h *PKCSHandler) handleLogout(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r sessionReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	if err := h.manager.Logout(ctx, pkcs11types.SessionHandle(r.SessionHandle)); err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- FindObjectsInit ----

type findObjectsInitReq struct {
	SessionHandle uint32 `json:"session_handle"`
	Template      []struct {
		Type  uint32 `json:"type"`
		Value []byte `json:"value"`
	} `json:"template"`
}

func (h *PKCSHandler) handleFindObjectsInit(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r findObjectsInitReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	template := make([]pkcs11types.Attribute, len(r.Template))
	for i, t := range r.Template {
		template[i] = pkcs11types.Attribute{
			Type:  pkcs11types.AttributeType(t.Type),
			Value: t.Value,
		}
	}

	handles, err := s.Provider.FindObjects(ctx, template)
	if err != nil {
		slog.Error("FindObjects 失败", "error", err)
		return nil, uint32(pkcs11types.CKR_FUNCTION_FAILED)
	}

	s.FindTemplate = template
	s.FindResults = handles
	s.FindPos = 0
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- FindObjects ----

type findObjectsReq struct {
	SessionHandle uint32 `json:"session_handle"`
	MaxCount      uint32 `json:"max_count"`
}

type findObjectsResp struct {
	Handles []uint32 `json:"handles"`
}

func (h *PKCSHandler) handleFindObjects(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r findObjectsReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	remaining := s.FindResults[s.FindPos:]
	count := int(r.MaxCount)
	if count > len(remaining) {
		count = len(remaining)
	}

	result := make([]uint32, count)
	for i := 0; i < count; i++ {
		result[i] = uint32(remaining[i])
	}
	s.FindPos += count

	return &findObjectsResp{Handles: result}, uint32(pkcs11types.CKR_OK)
}

// ---- FindObjectsFinal ----

func (h *PKCSHandler) handleFindObjectsFinal(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r sessionReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	s.FindResults = nil
	s.FindPos = 0
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- GetAttributeValue ----

type getAttrReq struct {
	SessionHandle uint32   `json:"session_handle"`
	ObjectHandle  uint32   `json:"object_handle"`
	AttrTypes     []uint32 `json:"attr_types"`
}

type getAttrResp struct {
	Attributes []struct {
		Type  uint32 `json:"type"`
		Value []byte `json:"value"`
	} `json:"attributes"`
}

func (h *PKCSHandler) handleGetAttributeValue(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r getAttrReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	attrTypes := make([]pkcs11types.AttributeType, len(r.AttrTypes))
	for i, t := range r.AttrTypes {
		attrTypes[i] = pkcs11types.AttributeType(t)
	}

	attrs, err := s.Provider.GetAttributes(ctx, pkcs11types.ObjectHandle(r.ObjectHandle), attrTypes)
	if err != nil {
		return nil, uint32(pkcs11types.CKR_OBJECT_HANDLE_INVALID)
	}

	resp := &getAttrResp{}
	for _, a := range attrs {
		resp.Attributes = append(resp.Attributes, struct {
			Type  uint32 `json:"type"`
			Value []byte `json:"value"`
		}{Type: uint32(a.Type), Value: a.Value})
	}
	return resp, uint32(pkcs11types.CKR_OK)
}

// ---- SignInit ----

type signInitReq struct {
	SessionHandle uint32 `json:"session_handle"`
	MechanismType uint32 `json:"mechanism_type"`
	MechParam     []byte `json:"mech_param,omitempty"`
	KeyHandle     uint32 `json:"key_handle"`
}

func (h *PKCSHandler) handleSignInit(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r signInitReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	s.SignHandle = pkcs11types.ObjectHandle(r.KeyHandle)
	s.SignMechanism = pkcs11types.Mechanism{
		Type:      pkcs11types.MechanismType(r.MechanismType),
		Parameter: r.MechParam,
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- Sign ----

type signReq struct {
	SessionHandle uint32 `json:"session_handle"`
	Data          []byte `json:"data"`
}

type signResp struct {
	Signature []byte `json:"signature"`
}

func (h *PKCSHandler) handleSign(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r signReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	if !s.Provider.IsLoggedIn() {
		return nil, uint32(pkcs11types.CKR_USER_NOT_LOGGED_IN)
	}

	sig, err := s.Provider.Sign(ctx, s.SignHandle, s.SignMechanism, r.Data)
	if err != nil {
		slog.Error("Sign 失败", "error", err)
		return nil, uint32(pkcs11types.CKR_FUNCTION_FAILED)
	}
	return &signResp{Signature: sig}, uint32(pkcs11types.CKR_OK)
}

// ---- DecryptInit ----

type cryptInitReq struct {
	SessionHandle uint32 `json:"session_handle"`
	MechanismType uint32 `json:"mechanism_type"`
	MechParam     []byte `json:"mech_param,omitempty"`
	KeyHandle     uint32 `json:"key_handle"`
}

func (h *PKCSHandler) handleDecryptInit(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r cryptInitReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	s.DecryptHandle = pkcs11types.ObjectHandle(r.KeyHandle)
	s.DecryptMech = pkcs11types.Mechanism{
		Type:      pkcs11types.MechanismType(r.MechanismType),
		Parameter: r.MechParam,
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- Decrypt ----

type decryptReq struct {
	SessionHandle uint32 `json:"session_handle"`
	Ciphertext    []byte `json:"ciphertext"`
}

type decryptResp struct {
	Plaintext []byte `json:"plaintext"`
}

func (h *PKCSHandler) handleDecrypt(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r decryptReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	if !s.Provider.IsLoggedIn() {
		return nil, uint32(pkcs11types.CKR_USER_NOT_LOGGED_IN)
	}

	plain, err := s.Provider.Decrypt(ctx, s.DecryptHandle, s.DecryptMech, r.Ciphertext)
	if err != nil {
		slog.Error("Decrypt 失败", "error", err)
		return nil, uint32(pkcs11types.CKR_FUNCTION_FAILED)
	}
	return &decryptResp{Plaintext: plain}, uint32(pkcs11types.CKR_OK)
}

// ---- EncryptInit ----

func (h *PKCSHandler) handleEncryptInit(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r cryptInitReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	s.EncryptHandle = pkcs11types.ObjectHandle(r.KeyHandle)
	s.EncryptMech = pkcs11types.Mechanism{
		Type:      pkcs11types.MechanismType(r.MechanismType),
		Parameter: r.MechParam,
	}
	return nil, uint32(pkcs11types.CKR_OK)
}

// ---- Encrypt ----

type encryptReq struct {
	SessionHandle uint32 `json:"session_handle"`
	Plaintext     []byte `json:"plaintext"`
}

type encryptResp struct {
	Ciphertext []byte `json:"ciphertext"`
}

func (h *PKCSHandler) handleEncrypt(ctx context.Context, req *Frame) (interface{}, uint32) {
	var r encryptReq
	if err := ParseRequest(req.Payload, &r); err != nil {
		return nil, uint32(pkcs11types.CKR_ARGUMENTS_BAD)
	}

	s, err := h.manager.GetSession(pkcs11types.SessionHandle(r.SessionHandle))
	if err != nil {
		return nil, uint32(pkcs11types.CKR_SESSION_HANDLE_INVALID)
	}

	cipher, err := s.Provider.Encrypt(ctx, s.EncryptHandle, s.EncryptMech, r.Plaintext)
	if err != nil {
		slog.Error("Encrypt 失败", "error", err)
		return nil, uint32(pkcs11types.CKR_FUNCTION_FAILED)
	}
	return &encryptResp{Ciphertext: cipher}, uint32(pkcs11types.CKR_OK)
}

// 确保 json 包被使用（用于 RawMessage）
var _ = json.Marshal
