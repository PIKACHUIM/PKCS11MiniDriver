// Package api 提供 servers 的 REST API 服务。
package api

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/globaltrusts/server-card/configs"
	"github.com/globaltrusts/server-card/internal/acme"
	"github.com/globaltrusts/server-card/internal/auth"
	"github.com/globaltrusts/server-card/internal/ca"
	"github.com/globaltrusts/server-card/internal/card"
	"github.com/globaltrusts/server-card/internal/ct"
	"github.com/globaltrusts/server-card/internal/issuance"
	"github.com/globaltrusts/server-card/internal/payment"
	"github.com/globaltrusts/server-card/internal/revocation"
	"github.com/globaltrusts/server-card/internal/storage"
	"github.com/globaltrusts/server-card/internal/template"
	"github.com/globaltrusts/server-card/internal/verification"
	"github.com/globaltrusts/server-card/internal/workflow"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Services 是所有业务服务的容器，用于简化 NewServer 参数列表。
type Services struct {
	CardSvc       *card.Service
	CASvc         *ca.Service
	IssuanceSvc   *issuance.Service
	VerifySvc     *verification.Service
	WorkflowSvc   *workflow.Service
	PaymentSvc    *payment.Service
	TmplSvc       *template.Service
	RevocationSvc *revocation.Service
	CTSvc         *ct.Service
	ACMESvc       *acme.Service
}

// Server 是 REST API 服务器。
type Server struct {
	cfg           *configs.Config
	httpServer    *http.Server
	db            *storage.DB
	jwtMgr        *auth.Manager
	cardSvc       *card.Service
	caSvc         *ca.Service
	issuanceSvc   *issuance.Service
	verifySvc     *verification.Service
	workflowSvc   *workflow.Service
	userRepo      *storage.UserRepo
	logRepo       *storage.LogRepo
	paymentSvc    *payment.Service
	tmplSvc       *template.Service
	revocationSvc *revocation.Service
	ctSvc         *ct.Service
	acmeSvc       *acme.Service
}

// NewServer 创建 API 服务器。
func NewServer(cfg *configs.Config, db *storage.DB, jwtMgr *auth.Manager, svcs *Services, userRepo *storage.UserRepo, logRepo *storage.LogRepo) *Server {
	s := &Server{
		cfg:           cfg,
		db:            db,
		jwtMgr:        jwtMgr,
		cardSvc:       svcs.CardSvc,
		caSvc:         svcs.CASvc,
		issuanceSvc:   svcs.IssuanceSvc,
		verifySvc:     svcs.VerifySvc,
		workflowSvc:   svcs.WorkflowSvc,
		userRepo:      userRepo,
		logRepo:       logRepo,
		paymentSvc:    svcs.PaymentSvc,
		tmplSvc:       svcs.TmplSvc,
		revocationSvc: svcs.RevocationSvc,
		ctSvc:         svcs.CTSvc,
		acmeSvc:       svcs.ACMESvc,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         cfg.API.Addr(),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// registerRoutes 注册所有路由。
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// 门户首页（静态文件服务，无需认证）
	mux.HandleFunc("GET /", s.handlePortal)

	// 健康检查（无需认证）
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// 认证（无需认证）
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/refresh", s.authMiddleware(s.handleRefresh))
	mux.HandleFunc("DELETE /api/auth/logout", s.authMiddleware(s.handleLogout))

	// 用户管理（需要认证）
	mux.HandleFunc("GET /api/users/me", s.authMiddleware(s.handleGetProfile))
	mux.HandleFunc("PUT /api/users/me", s.authMiddleware(s.handleUpdateProfile))
	mux.HandleFunc("PUT /api/auth/password", s.authMiddleware(s.handleChangePassword))
	mux.HandleFunc("PUT /api/users/me/pubkey", s.authMiddleware(s.handleUpdatePublicKey))

	// 卡片管理（读取需认证，写入需 user 以上角色）
	mux.HandleFunc("GET /api/cards", s.authMiddleware(s.handleListCards))
	mux.HandleFunc("POST /api/cards", s.writeOnly(s.handleCreateCard))
	mux.HandleFunc("GET /api/cards/{uuid}", s.authMiddleware(s.handleGetCard))
	mux.HandleFunc("DELETE /api/cards/{uuid}", s.writeOnly(s.handleDeleteCard))

	// 证书管理（读取需认证，写入需 user 以上角色）
	mux.HandleFunc("GET /api/cards/{uuid}/certs", s.authMiddleware(s.handleListCerts))
	mux.HandleFunc("POST /api/cards/{uuid}/certs", s.writeOnly(s.handleImportCert))
	mux.HandleFunc("DELETE /api/cards/{uuid}/certs/{cert_uuid}", s.writeOnly(s.handleDeleteCert))

	// 证书增强管理（筛选查询、吊销、分配）
	mux.HandleFunc("GET /api/certs", s.authMiddleware(s.handleListCertsFiltered))
	mux.HandleFunc("POST /api/certs/{uuid}/revoke", s.adminOnly(s.handleRevokeCertByUUID))
	mux.HandleFunc("POST /api/certs/{uuid}/assign", s.adminOnly(s.handleAssignCertToCard))

	// 密钥生成（需要认证）
	mux.HandleFunc("POST /api/cards/{uuid}/keygen", s.authMiddleware(s.handleKeyGen))

	// 云端签名/解密（需要认证）
	mux.HandleFunc("POST /api/cards/{uuid}/sign", s.authMiddleware(s.handleSign))
	mux.HandleFunc("POST /api/cards/{uuid}/decrypt", s.authMiddleware(s.handleDecrypt))

	// 支付系统（需要认证）
	mux.HandleFunc("POST /api/payment/recharge", s.authMiddleware(s.handleCreateRecharge))
	mux.HandleFunc("GET /api/payment/orders", s.authMiddleware(s.handleListOrders))
	mux.HandleFunc("GET /api/payment/balance", s.authMiddleware(s.handleGetBalance))
	mux.HandleFunc("POST /api/payment/refund", s.authMiddleware(s.handleCreateRefund))
	// 支付回调（无需认证，由支付平台调用，内部验签）
	mux.HandleFunc("POST /api/payment/callback/{channel}", s.handlePaymentCallback)

	// 密钥存储类型模板（读取需认证，写入需管理员）
	mux.HandleFunc("GET /api/templates/key-storage", s.authMiddleware(s.handleListKeyStorageTemplates))
	mux.HandleFunc("POST /api/templates/key-storage", s.adminOnly(s.handleCreateKeyStorageTemplate))
	mux.HandleFunc("GET /api/templates/key-storage/{uuid}", s.authMiddleware(s.handleGetKeyStorageTemplate))
	mux.HandleFunc("PUT /api/templates/key-storage/{uuid}", s.adminOnly(s.handleUpdateKeyStorageTemplate))
	mux.HandleFunc("DELETE /api/templates/key-storage/{uuid}", s.adminOnly(s.handleDeleteKeyStorageTemplate))

	// CA 管理（读取需认证，写入需管理员）
	mux.HandleFunc("GET /api/cas", s.authMiddleware(s.handleListCAs))
	mux.HandleFunc("POST /api/cas", s.adminOnly(s.handleCreateCA))
	mux.HandleFunc("GET /api/cas/{uuid}", s.authMiddleware(s.handleGetCA))
	mux.HandleFunc("PUT /api/cas/{uuid}", s.adminOnly(s.handleUpdateCA))
	mux.HandleFunc("DELETE /api/cas/{uuid}", s.adminOnly(s.handleDeleteCA))
	mux.HandleFunc("POST /api/cas/{uuid}/import-chain", s.adminOnly(s.handleImportCAChain))
	mux.HandleFunc("GET /api/cas/{uuid}/revoked", s.authMiddleware(s.handleListRevokedCerts))
	mux.HandleFunc("POST /api/cas/{uuid}/revoke", s.adminOnly(s.handleRevokeCert))
	mux.HandleFunc("GET /api/cas/{uuid}/crl", s.handleGetCRL)
	mux.HandleFunc("POST /api/cas/{uuid}/issue", s.adminOnly(s.handleIssueCert))

	// 证书颁发模板管理（读取需认证，写入需管理员）
	mux.HandleFunc("GET /api/templates/issuance", s.authMiddleware(s.handleListIssuanceTemplates))
	mux.HandleFunc("POST /api/templates/issuance", s.adminOnly(s.handleCreateIssuanceTemplate))
	mux.HandleFunc("GET /api/templates/issuance/{uuid}", s.authMiddleware(s.handleGetIssuanceTemplate))
	mux.HandleFunc("DELETE /api/templates/issuance/{uuid}", s.adminOnly(s.handleDeleteIssuanceTemplate))

	// 主体模板管理（管理员）
	mux.HandleFunc("GET /api/templates/subject", s.authMiddleware(s.handleListSubjectTemplates))
	mux.HandleFunc("POST /api/templates/subject", s.adminOnly(s.handleCreateSubjectTemplate))
	mux.HandleFunc("DELETE /api/templates/subject/{uuid}", s.adminOnly(s.handleDeleteSubjectTemplate))

	// 扩展信息模板管理（管理员）
	mux.HandleFunc("GET /api/templates/extension", s.authMiddleware(s.handleListExtensionTemplates))
	mux.HandleFunc("POST /api/templates/extension", s.adminOnly(s.handleCreateExtensionTemplate))
	mux.HandleFunc("DELETE /api/templates/extension/{uuid}", s.adminOnly(s.handleDeleteExtensionTemplate))

	// 密钥用途模板管理（管理员）
	mux.HandleFunc("GET /api/templates/key-usage", s.authMiddleware(s.handleListKeyUsageTemplates))
	mux.HandleFunc("POST /api/templates/key-usage", s.adminOnly(s.handleCreateKeyUsageTemplate))
	mux.HandleFunc("DELETE /api/templates/key-usage/{uuid}", s.adminOnly(s.handleDeleteKeyUsageTemplate))

	// 存储区域管理（管理员）
	mux.HandleFunc("GET /api/storage-zones", s.authMiddleware(s.handleListStorageZones))
	mux.HandleFunc("POST /api/storage-zones", s.adminOnly(s.handleCreateStorageZone))
	mux.HandleFunc("DELETE /api/storage-zones/{uuid}", s.adminOnly(s.handleDeleteStorageZone))

	// OID 管理（管理员）
	mux.HandleFunc("GET /api/oids", s.authMiddleware(s.handleListOIDs))
	mux.HandleFunc("POST /api/oids", s.adminOnly(s.handleCreateOID))
	mux.HandleFunc("DELETE /api/oids/{uuid}", s.adminOnly(s.handleDeleteOID))

	// 云端 TOTP 管理（需要认证）
	mux.HandleFunc("GET /api/totp", s.authMiddleware(s.handleListUserTOTPs))
	mux.HandleFunc("POST /api/totp", s.writeOnly(s.handleCreateUserTOTP))
	mux.HandleFunc("GET /api/totp/{uuid}/code", s.authMiddleware(s.handleGetTOTPCode))
	mux.HandleFunc("DELETE /api/totp/{uuid}", s.writeOnly(s.handleDeleteUserTOTP))

	// 主体信息管理（需要认证）
	mux.HandleFunc("GET /api/subject-infos", s.authMiddleware(s.handleListSubjectInfos))
	mux.HandleFunc("POST /api/subject-infos", s.writeOnly(s.handleCreateSubjectInfo))
	mux.HandleFunc("PUT /api/subject-infos/{uuid}/approve", s.adminOnly(s.handleApproveSubjectInfo))
	mux.HandleFunc("PUT /api/subject-infos/{uuid}/reject", s.adminOnly(s.handleRejectSubjectInfo))

	// 扩展信息验证（需要认证）
	mux.HandleFunc("GET /api/extension-infos", s.authMiddleware(s.handleListExtensionInfos))
	mux.HandleFunc("POST /api/extension-infos", s.writeOnly(s.handleCreateExtensionInfo))
	mux.HandleFunc("POST /api/extension-infos/{uuid}/verify-dns", s.authMiddleware(s.handleVerifyDNS))
	mux.HandleFunc("POST /api/extension-infos/{uuid}/verify-email", s.authMiddleware(s.handleVerifyEmail))
	mux.HandleFunc("DELETE /api/extension-infos/{uuid}", s.writeOnly(s.handleDeleteExtensionInfo))

	// 证书订单与申请（需要认证）
	mux.HandleFunc("POST /api/cert-orders", s.writeOnly(s.handleCreateCertOrder))
	mux.HandleFunc("GET /api/cert-orders", s.authMiddleware(s.handleListCertOrders))
	mux.HandleFunc("POST /api/cert-applications", s.writeOnly(s.handleCreateCertApplication))
	mux.HandleFunc("GET /api/cert-applications", s.authMiddleware(s.handleListCertApplications))
	mux.HandleFunc("PUT /api/cert-applications/{uuid}/approve", s.adminOnly(s.handleApproveCertApplication))
	mux.HandleFunc("PUT /api/cert-applications/{uuid}/reject", s.adminOnly(s.handleRejectCertApplication))

	// 公开服务路由（无需认证，供外部客户端访问）
	// CRL 下载：GET /crl/{caUUID}
	mux.HandleFunc("GET /crl/{caUUID}", s.handlePublicCRL)
	// OCSP 查询：POST /ocsp/{caUUID}
	mux.HandleFunc("POST /ocsp/{caUUID}", s.handlePublicOCSP)
	// CA 证书下载（AIA CAIssuer）：GET /ca/{caUUID}
	mux.HandleFunc("GET /ca/{caUUID}", s.handlePublicCAIssuer)
	// ACME 目录：GET /acme/{path}/directory
	mux.HandleFunc("GET /acme/{path}/directory", s.handleACMEDirectory)
	// ACME 新 Nonce：HEAD /acme/{path}/new-nonce
	mux.HandleFunc("HEAD /acme/{path}/new-nonce", s.handleACMENewNonce)
	// CT 提交：POST /ct/submit
	mux.HandleFunc("POST /ct/submit", s.handleCTSubmit)
	// CT 查询：GET /ct/query
	mux.HandleFunc("GET /ct/query", s.handleCTQuery)

	// 吊销服务管理（管理员）
	mux.HandleFunc("GET /api/revocation-services", s.adminOnly(s.handleListRevocationServices))
	mux.HandleFunc("POST /api/revocation-services", s.adminOnly(s.handleCreateRevocationService))
	mux.HandleFunc("DELETE /api/revocation-services/{uuid}", s.adminOnly(s.handleDeleteRevocationService))

	// ACME 配置管理（管理员）
	mux.HandleFunc("GET /api/acme-configs", s.adminOnly(s.handleListACMEConfigs))
	mux.HandleFunc("POST /api/acme-configs", s.adminOnly(s.handleCreateACMEConfig))
	mux.HandleFunc("DELETE /api/acme-configs/{uuid}", s.adminOnly(s.handleDeleteACMEConfig))

	// CT 记录管理（需要认证）
	mux.HandleFunc("GET /api/ct-entries", s.authMiddleware(s.handleListCTEntries))
	mux.HandleFunc("DELETE /api/ct-entries/{uuid}", s.adminOnly(s.handleDeleteCTEntry))
}

// Start 启动 HTTP 服务。
func (s *Server) Start() error {
	slog.Info("servers API 服务启动", "addr", s.cfg.API.Addr())
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API 服务异常退出", "error", err)
		}
	}()
	return nil
}

// Stop 优雅关闭 HTTP 服务。
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// ---- 认证中间件 ----

// authMiddleware 验证 JWT Token，将 Claims 注入 context。
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "缺少认证 Token")
			return
		}

		claims, err := s.jwtMgr.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Token 无效或已过期")
			return
		}

		// 检查 Token 是否即将过期，提示客户端刷新
		if s.jwtMgr.NeedsRefresh(claims) {
			w.Header().Set("X-Token-Refresh", "true")
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next(w, r.WithContext(ctx))
	}
}

// adminOnly 仅允许 admin 角色访问的中间件（需先经过 authMiddleware）。
func (s *Server) adminOnly(next http.HandlerFunc) http.HandlerFunc {
	return s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "需要管理员权限")
			return
		}
		next(w, r)
	})
}

// writeOnly 仅允许 admin 或 user 角色访问（readonly 被拒绝），需先经过 authMiddleware。
func (s *Server) writeOnly(next http.HandlerFunc) http.HandlerFunc {
	return s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims.Role == "readonly" {
			writeError(w, http.StatusForbidden, "只读用户无权执行此操作")
			return
		}
		next(w, r)
	})
}

// ---- 处理器 ----

func (s *Server) handlePortal(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "servers"})
}

// LoginRequest 是登录请求体。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	user, err := s.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	if !user.Enabled {
		writeError(w, http.StatusForbidden, "账号已禁用")
		return
	}

	// 检查是否被锁定
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		writeError(w, http.StatusTooManyRequests, fmt.Sprintf("账号已锁定，请在 %s 后重试", user.LockedUntil.Format("15:04:05")))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// 登录失败，递增失败计数（5次锁定15分钟）
		s.userRepo.IncrementFailedAttempts(r.Context(), user.UUID, 5, 15*time.Minute)
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	// 登录成功，重置失败计数
	s.userRepo.ResetFailedAttempts(r.Context(), user.UUID)

	token, _, err := s.jwtMgr.Sign(user.UUID, user.Username, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	// 记录登录日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  user.UUID,
		Action:    "login",
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":     token,
		"user_uuid": user.UUID,
		"username":  user.Username,
		"role":      user.Role,
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	token, _, err := s.jwtMgr.Refresh(claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "刷新 Token 失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// RegisterRequest 是注册请求体。
type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "密码长度不能少于8位")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "邮箱不能为空")
		return
	}

	// 检查用户名唯一性
	if _, err := s.userRepo.GetByUsername(r.Context(), req.Username); err == nil {
		writeError(w, http.StatusConflict, "用户名已存在")
		return
	}

	// 检查邮箱唯一性
	if _, err := s.userRepo.GetByEmail(r.Context(), req.Email); err == nil {
		writeError(w, http.StatusConflict, "邮箱已被注册")
		return
	}

	// 密码哈希（bcrypt cost=13）
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 13)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码加密失败")
		return
	}

	user := &storage.User{
		Username:     req.Username,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         "user",
		Enabled:      true,
	}

	if err := s.userRepo.Create(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "创建用户失败: "+err.Error())
		return
	}

	// 记录注册日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: user.UUID,
		Action:   "register",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user_uuid": user.UUID,
		"username":  user.Username,
		"message":   "注册成功",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())

	// 将当前 Token 加入黑名单
	if claims.ExpiresAt != nil {
		s.jwtMgr.Revoke(claims.ID, claims.ExpiresAt.Time)
	}

	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   "logout",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "已登出"})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	user, err := s.userRepo.GetByUUID(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// UpdateProfileRequest 是更新个人信息请求体。
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := s.userRepo.UpdateProfile(r.Context(), claims.UserUUID, req.DisplayName, req.Email); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "个人信息已更新"})
}

// ChangePasswordRequest 是修改密码请求体。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "旧密码和新密码不能为空")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "新密码长度不能少于8位")
		return
	}

	user, err := s.userRepo.GetByUUID(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "旧密码错误")
		return
	}

	// 生成新密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 13)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "密码加密失败")
		return
	}

	if err := s.userRepo.UpdatePassword(r.Context(), claims.UserUUID, string(hash)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   "change_password",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "密码已修改"})
}

// UpdatePublicKeyRequest 是更新公钥请求体。
type UpdatePublicKeyRequest struct {
	PublicKey []byte `json:"public_key"` // DER 编码的公钥
}

func (s *Server) handleUpdatePublicKey(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req UpdatePublicKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if len(req.PublicKey) == 0 {
		writeError(w, http.StatusBadRequest, "公钥不能为空")
		return
	}

	if err := s.userRepo.UpdatePublicKey(r.Context(), claims.UserUUID, req.PublicKey); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   "update_pubkey",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "公钥已更新"})
}

// ---- 卡片处理器 ----

func (s *Server) handleListCards(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cards, err := s.cardSvc.ListCards(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"cards": cards, "total": len(cards)})
}

// CreateCardRequest 是创建卡片请求体。
type CreateCardRequest struct {
	CardName string `json:"card_name"`
	Remark   string `json:"remark"`
}

func (s *Server) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.CardName == "" {
		writeError(w, http.StatusBadRequest, "卡片名称不能为空")
		return
	}

	c, err := s.cardSvc.CreateCard(r.Context(), claims.UserUUID, req.CardName, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleGetCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	c, err := s.cardSvc.GetCard(r.Context(), cardUUID, claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	if err := s.cardSvc.DeleteCard(r.Context(), cardUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "卡片已删除"})
}

// ---- 证书处理器 ----

func (s *Server) handleListCerts(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	certs, err := s.cardSvc.ListCerts(r.Context(), cardUUID, claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"certs": certs, "total": len(certs)})
}

// ImportCertRequest 是导入证书请求体。
type ImportCertRequest struct {
	CertType    string `json:"cert_type"`
	KeyType     string `json:"key_type"`
	CertContent []byte `json:"cert_content"` // Base64 编码的 DER
	Remark      string `json:"remark"`
}

func (s *Server) handleImportCert(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req ImportCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	cert, err := s.cardSvc.ImportCert(r.Context(), cardUUID, claims.UserUUID, req.CertType, req.KeyType, req.Remark, req.CertContent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

func (s *Server) handleDeleteCert(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")
	certUUID := r.PathValue("cert_uuid")

	if err := s.cardSvc.DeleteCert(r.Context(), certUUID, cardUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已删除"})
}

// KeyGenRequest 是密钥生成请求体。
type KeyGenRequest struct {
	KeyType string `json:"key_type"` // rsa2048/rsa4096/ec256/ec384/ec521
	Remark  string `json:"remark"`
}

func (s *Server) handleKeyGen(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req KeyGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}

	cert, err := s.cardSvc.GenerateKeyPair(r.Context(), cardUUID, claims.UserUUID, req.KeyType, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

// SignRequest 是签名请求体。
type SignRequest struct {
	CertUUID  string `json:"cert_uuid"`
	Mechanism string `json:"mechanism"` // ECDSA_SHA256/SHA256_RSA_PKCS/...
	Data      []byte `json:"data"`      // 待签名数据（Base64）
}

func (s *Server) handleSign(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	sig, err := s.cardSvc.Sign(r.Context(), req.CertUUID, cardUUID, claims.UserUUID, req.Mechanism, req.Data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 记录签名日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  claims.UserUUID,
		CardUUID:  cardUUID,
		CertUUID:  req.CertUUID,
		Action:    fmt.Sprintf("sign:%s", req.Mechanism),
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string][]byte{"signature": sig})
}

// DecryptRequest 是解密请求体。
type DecryptRequest struct {
	CertUUID   string `json:"cert_uuid"`
	Mechanism  string `json:"mechanism"`
	Ciphertext []byte `json:"ciphertext"`
}

func (s *Server) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	plaintext, err := s.cardSvc.Decrypt(r.Context(), req.CertUUID, cardUUID, claims.UserUUID, req.Mechanism, req.Ciphertext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  claims.UserUUID,
		CardUUID:  cardUUID,
		CertUUID:  req.CertUUID,
		Action:    fmt.Sprintf("decrypt:%s", req.Mechanism),
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string][]byte{"plaintext": plaintext})
}

// ---- 工具函数 ----

type claimsKey struct{}

func claimsFromCtx(ctx context.Context) *auth.Claims {
	return ctx.Value(claimsKey{}).(*auth.Claims)
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return h[7:]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ---- 支付系统处理器 ----

// RechargeRequest 是充值请求体。
type RechargeRequest struct {
	AmountCents int64  `json:"amount_cents"` // 金额（分）
	Channel     string `json:"channel"`      // 支付渠道
}

func (s *Server) handleCreateRecharge(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req RechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.AmountCents <= 0 {
		writeError(w, http.StatusBadRequest, "充值金额必须大于0")
		return
	}
	if req.Channel == "" {
		writeError(w, http.StatusBadRequest, "支付渠道不能为空")
		return
	}

	notifyURL := fmt.Sprintf("%s/api/payment/callback/%s", s.cfg.API.BaseURL, req.Channel)
	order, payResp, err := s.paymentSvc.CreateRechargeOrder(r.Context(), claims.UserUUID, req.AmountCents, req.Channel, notifyURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"order":   order,
		"pay_url": payResp.PayURL,
		"qr_code": payResp.QRCode,
	})
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	orders, total, err := s.paymentSvc.ListOrders(r.Context(), claims.UserUUID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
	})
}

func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	balance, err := s.paymentSvc.GetBalance(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, balance)
}

// RefundRequestBody 是退款请求体。
type RefundRequestBody struct {
	OrderNo string `json:"order_no"`
	Reason  string `json:"reason"`
}

func (s *Server) handleCreateRefund(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req RefundRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.OrderNo == "" {
		writeError(w, http.StatusBadRequest, "订单号不能为空")
		return
	}

	refund, err := s.paymentSvc.CreateRefund(r.Context(), claims.UserUUID, req.OrderNo, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 记录退款申请日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("refund_request:%s", req.OrderNo),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "退款申请已提交，等待管理员审批",
		"refund":  refund,
	})
}

func (s *Server) handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 限制 1MB
	if err != nil {
		writeError(w, http.StatusBadRequest, "读取请求体失败")
		return
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	if err := s.paymentSvc.HandleCallback(r.Context(), channel, body, headers); err != nil {
		slog.Error("支付回调处理失败", "channel", channel, "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 大多数支付平台期望返回 "success" 字符串
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

// ---- 密钥存储类型模板处理器 ----

func (s *Server) handleListKeyStorageTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.tmplSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
		"total":     len(templates),
	})
}

// CreateKeyStorageTemplateRequest 是创建模板请求体。
type CreateKeyStorageTemplateRequest struct {
	Name            string `json:"name"`
	StorageMethods  uint32 `json:"storage_methods"`
	SecurityLevel   string `json:"security_level"`
	AllowReimport   bool   `json:"allow_reimport"`
	CloudBackup     bool   `json:"cloud_backup"`
	AllowReissue    bool   `json:"allow_reissue"`
	MaxReissueCount int    `json:"max_reissue_count"`
}

func (s *Server) handleCreateKeyStorageTemplate(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyStorageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	tmpl := &storage.KeyStorageTemplate{
		Name:            req.Name,
		StorageMethods:  storage.StorageMethod(req.StorageMethods),
		SecurityLevel:   storage.SecurityLevel(req.SecurityLevel),
		AllowReimport:   req.AllowReimport,
		CloudBackup:     req.CloudBackup,
		AllowReissue:    req.AllowReissue,
		MaxReissueCount: req.MaxReissueCount,
	}

	if err := s.tmplSvc.Create(r.Context(), tmpl); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tmpl)
}

func (s *Server) handleGetKeyStorageTemplate(w http.ResponseWriter, r *http.Request) {
	templateUUID := r.PathValue("uuid")
	tmpl, err := s.tmplSvc.GetByUUID(r.Context(), templateUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, tmpl)
}

// UpdateKeyStorageTemplateRequest 是更新模板请求体（仅允许修改非安全属性）。
type UpdateKeyStorageTemplateRequest struct {
	Name            string `json:"name"`
	MaxReissueCount int    `json:"max_reissue_count"`
}

func (s *Server) handleUpdateKeyStorageTemplate(w http.ResponseWriter, r *http.Request) {
	templateUUID := r.PathValue("uuid")
	var req UpdateKeyStorageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := s.tmplSvc.Update(r.Context(), templateUUID, req.Name, req.MaxReissueCount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "模板已更新"})
}

func (s *Server) handleDeleteKeyStorageTemplate(w http.ResponseWriter, r *http.Request) {
	templateUUID := r.PathValue("uuid")
	if err := s.tmplSvc.Delete(r.Context(), templateUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "模板已删除"})
}

// ---- CA 管理处理器 ----

func (s *Server) handleListCAs(w http.ResponseWriter, r *http.Request) {
	cas, err := s.caSvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"cas": cas, "total": len(cas)})
}

// CreateCARequest 是创建 CA 请求体。
type CreateCARequest struct {
	Name       string `json:"name"`
	KeyType    string `json:"key_type"`    // rsa2048/rsa4096/ec256/ec384/ec521
	ValidYears int    `json:"valid_years"` // 有效期（年）
	CommonName string `json:"common_name"`
	Org        string `json:"organization"`
	Country    string `json:"country"`
}

func (s *Server) handleCreateCA(w http.ResponseWriter, r *http.Request) {
	var req CreateCARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Name == "" || req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "CA 名称和 CommonName 不能为空")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}
	if req.ValidYears <= 0 || req.ValidYears > 10 {
		req.ValidYears = 10
	}

	subject := pkixName(req.CommonName, req.Org, req.Country)
	newCA, err := s.caSvc.CreateSelfSignedCA(r.Context(), req.Name, subject, req.KeyType, req.ValidYears)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	claims := claimsFromCtx(r.Context())
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   "create_ca:" + req.Name,
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, newCA)
}

func (s *Server) handleGetCA(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	result, err := s.caSvc.GetByUUID(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	// 不返回加密的私钥
	result.PrivateEnc = nil
	writeJSON(w, http.StatusOK, result)
}

// UpdateCARequest 是更新 CA 请求体。
type UpdateCARequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (s *Server) handleUpdateCA(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	var req UpdateCARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.caSvc.Update(r.Context(), caUUID, req.Name, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "CA 已更新"})
}

func (s *Server) handleDeleteCA(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	if err := s.caSvc.Delete(r.Context(), caUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "CA 已删除"})
}

// ImportChainRequest 是导入证书链请求体。
type ImportChainRequest struct {
	ChainPEM string `json:"chain_pem"`
}

func (s *Server) handleImportCAChain(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	var req ImportChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.caSvc.ImportChain(r.Context(), caUUID, req.ChainPEM); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书链已导入"})
}

func (s *Server) handleListRevokedCerts(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	certs, err := s.caSvc.ListRevokedCerts(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"revoked_certs": certs, "total": len(certs)})
}

// RevokeRequest 是吊销证书请求体。
type RevokeRequest struct {
	SerialNumber string `json:"serial_number"` // 十六进制序列号
	Reason       int    `json:"reason"`        // RFC 5280 吊销原因码
}

func (s *Server) handleRevokeCert(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.SerialNumber == "" {
		writeError(w, http.StatusBadRequest, "证书序列号不能为空")
		return
	}
	if err := s.caSvc.RevokeCert(r.Context(), caUUID, req.SerialNumber, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	claims := claimsFromCtx(r.Context())
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("revoke_cert:%s:%s", caUUID, req.SerialNumber),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已吊销"})
}

func (s *Server) handleGetCRL(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	crlDER, err := s.caSvc.GenerateCRL(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.crl", caUUID))
	w.WriteHeader(http.StatusOK)
	w.Write(crlDER)
}

// IssueCertRequest 是签发证书请求体。
type IssueCertRequest struct {
	KeyType     string   `json:"key_type"`
	ValidDays   int      `json:"valid_days"`
	CommonName  string   `json:"common_name"`
	Org         string   `json:"organization"`
	Country     string   `json:"country"`
	IsCA        bool     `json:"is_ca"`
	PathLen     int      `json:"path_len"`
	DNSNames    []string `json:"dns_names"`
	IPAddresses []string `json:"ip_addresses"`
	EmailAddrs  []string `json:"email_addresses"`
}

func (s *Server) handleIssueCert(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("uuid")
	var req IssueCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.CommonName == "" {
		writeError(w, http.StatusBadRequest, "CommonName 不能为空")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}
	if req.ValidDays <= 0 {
		req.ValidDays = 365
	}

	// 解析 IP 地址
	var ips []net.IP
	for _, ipStr := range req.IPAddresses {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("无效的 IP 地址: %s", ipStr))
			return
		}
		ips = append(ips, ip)
	}

	subject := pkixName(req.CommonName, req.Org, req.Country)
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if req.IsCA {
		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	}

	issueReq := &ca.IssueRequest{
		CAUUID:      caUUID,
		Subject:     subject,
		KeyType:     req.KeyType,
		ValidDays:   req.ValidDays,
		IsCA:        req.IsCA,
		PathLen:     req.PathLen,
		KeyUsage:    keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:    req.DNSNames,
		IPAddresses: ips,
		EmailAddrs:  req.EmailAddrs,
	}

	resp, err := s.caSvc.IssueCert(r.Context(), issueReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	claims := claimsFromCtx(r.Context())
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("issue_cert:%s:%s", caUUID, req.CommonName),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"cert_pem":      resp.CertPEM,
		"serial_number": resp.SerialNumber,
	})
}

// pkixName 构建 pkix.Name。
func pkixName(cn, org, country string) pkix.Name {
	name := pkix.Name{CommonName: cn}
	if org != "" {
		name.Organization = []string{org}
	}
	if country != "" {
		name.Country = []string{country}
	}
	return name
}

// ---- 颁发模板处理器 ----

func (s *Server) handleListIssuanceTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	templates, err := s.issuanceSvc.ListIssuanceTemplates(r.Context(), category, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "total": len(templates)})
}

func (s *Server) handleCreateIssuanceTemplate(w http.ResponseWriter, r *http.Request) {
	var t storage.IssuanceTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.issuanceSvc.CreateIssuanceTemplate(r.Context(), &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetIssuanceTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	t, err := s.issuanceSvc.GetIssuanceTemplate(r.Context(), tmplUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteIssuanceTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	if err := s.issuanceSvc.DeleteIssuanceTemplate(r.Context(), tmplUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "颁发模板已删除"})
}

func (s *Server) handleListSubjectTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.issuanceSvc.ListSubjectTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "total": len(templates)})
}

func (s *Server) handleCreateSubjectTemplate(w http.ResponseWriter, r *http.Request) {
	var t storage.SubjectTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.issuanceSvc.CreateSubjectTemplate(r.Context(), &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleDeleteSubjectTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	if err := s.issuanceSvc.DeleteSubjectTemplate(r.Context(), tmplUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体模板已删除"})
}

func (s *Server) handleListExtensionTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.issuanceSvc.ListExtensionTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "total": len(templates)})
}

func (s *Server) handleCreateExtensionTemplate(w http.ResponseWriter, r *http.Request) {
	var t storage.ExtensionTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.issuanceSvc.CreateExtensionTemplate(r.Context(), &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleDeleteExtensionTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	if err := s.issuanceSvc.DeleteExtensionTemplate(r.Context(), tmplUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "扩展信息模板已删除"})
}

func (s *Server) handleListKeyUsageTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.issuanceSvc.ListKeyUsageTemplates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates, "total": len(templates)})
}

func (s *Server) handleCreateKeyUsageTemplate(w http.ResponseWriter, r *http.Request) {
	var t storage.KeyUsageTemplate
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.issuanceSvc.CreateKeyUsageTemplate(r.Context(), &t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleDeleteKeyUsageTemplate(w http.ResponseWriter, r *http.Request) {
	tmplUUID := r.PathValue("uuid")
	if err := s.issuanceSvc.DeleteKeyUsageTemplate(r.Context(), tmplUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "密钥用途模板已删除"})
}

// ---- 存储区域处理器 ----

func (s *Server) handleListStorageZones(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT uuid, name, storage_type, hsm_driver, status, created_at, updated_at FROM storage_zones ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var zones []storage.StorageZone
	for rows.Next() {
		var z storage.StorageZone
		if err := rows.Scan(&z.UUID, &z.Name, &z.StorageType, &z.HSMDriver, &z.Status, &z.CreatedAt, &z.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		zones = append(zones, z)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"zones": zones, "total": len(zones)})
}

func (s *Server) handleCreateStorageZone(w http.ResponseWriter, r *http.Request) {
	var z storage.StorageZone
	if err := json.NewDecoder(r.Body).Decode(&z); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if z.Name == "" {
		writeError(w, http.StatusBadRequest, "区域名称不能为空")
		return
	}
	z.UUID = fmt.Sprintf("%s", uuid.New().String())
	z.CreatedAt = time.Now()
	z.UpdatedAt = time.Now()
	if z.Status == "" {
		z.Status = "active"
	}
	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO storage_zones (uuid, name, storage_type, hsm_driver, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		z.UUID, z.Name, z.StorageType, z.HSMDriver, z.Status, z.CreatedAt, z.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, z)
}

func (s *Server) handleDeleteStorageZone(w http.ResponseWriter, r *http.Request) {
	zoneUUID := r.PathValue("uuid")
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM storage_zones WHERE uuid = ?`, zoneUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "存储区域已删除"})
}

// ---- OID 管理处理器 ----

func (s *Server) handleListOIDs(w http.ResponseWriter, r *http.Request) {
	usageType := r.URL.Query().Get("usage_type")
	query := `SELECT uuid, oid_value, name, description, usage_type, created_at, updated_at FROM custom_oids`
	var args []interface{}
	if usageType != "" {
		query += ` WHERE usage_type = ?`
		args = append(args, usageType)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var oids []storage.CustomOID
	for rows.Next() {
		var o storage.CustomOID
		if err := rows.Scan(&o.UUID, &o.OIDValue, &o.Name, &o.Description, &o.UsageType, &o.CreatedAt, &o.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		oids = append(oids, o)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"oids": oids, "total": len(oids)})
}

func (s *Server) handleCreateOID(w http.ResponseWriter, r *http.Request) {
	var o storage.CustomOID
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if o.OIDValue == "" || o.Name == "" {
		writeError(w, http.StatusBadRequest, "OID 值和名称不能为空")
		return
	}
	o.UUID = uuid.New().String()
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO custom_oids (uuid, oid_value, name, description, usage_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		o.UUID, o.OIDValue, o.Name, o.Description, o.UsageType, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handleDeleteOID(w http.ResponseWriter, r *http.Request) {
	oidUUID := r.PathValue("uuid")
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM custom_oids WHERE uuid = ?`, oidUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "OID 已删除"})
}

// ---- 云端 TOTP 处理器 ----

func (s *Server) handleListUserTOTPs(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT uuid, user_uuid, issuer, account, algorithm, digits, period, created_at FROM user_totps WHERE user_uuid = ? ORDER BY created_at DESC`,
		claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var totps []map[string]interface{}
	for rows.Next() {
		var id, userUUID, issuer, account, algorithm string
		var digits, period int
		var createdAt time.Time
		if err := rows.Scan(&id, &userUUID, &issuer, &account, &algorithm, &digits, &period, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		totps = append(totps, map[string]interface{}{
			"uuid": id, "issuer": issuer, "account": account,
			"algorithm": algorithm, "digits": digits, "period": period, "created_at": createdAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"totps": totps, "total": len(totps)})
}

func (s *Server) handleCreateUserTOTP(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req struct {
		Issuer    string `json:"issuer"`
		Account   string `json:"account"`
		Secret    string `json:"secret"` // Base32 编码
		Algorithm string `json:"algorithm"`
		Digits    int    `json:"digits"`
		Period    int    `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Secret == "" {
		writeError(w, http.StatusBadRequest, "TOTP 密钥不能为空")
		return
	}
	if req.Algorithm == "" {
		req.Algorithm = "SHA1"
	}
	if req.Digits == 0 {
		req.Digits = 6
	}
	if req.Period == 0 {
		req.Period = 30
	}
	totpUUID := uuid.New().String()
	// 简化处理：将密钥直接存储（生产环境应加密）
	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO user_totps (uuid, user_uuid, issuer, account, secret_enc, algorithm, digits, period, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		totpUUID, claims.UserUUID, req.Issuer, req.Account, []byte(req.Secret),
		req.Algorithm, req.Digits, req.Period, time.Now(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"uuid": totpUUID, "message": "TOTP 已添加"})
}

func (s *Server) handleGetTOTPCode(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	totpUUID := r.PathValue("uuid")
	var secretEnc []byte
	var algorithm string
	var digits, period int
	err := s.db.QueryRowContext(r.Context(),
		`SELECT secret_enc, algorithm, digits, period FROM user_totps WHERE uuid = ? AND user_uuid = ?`,
		totpUUID, claims.UserUUID).Scan(&secretEnc, &algorithm, &digits, &period)
	if err != nil {
		writeError(w, http.StatusNotFound, "TOTP 条目不存在")
		return
	}
	// 简化：返回密钥信息（实际应计算 TOTP 码）
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uuid": totpUUID, "algorithm": algorithm, "digits": digits, "period": period,
		"message": "TOTP 验证码计算需要在客户端完成",
	})
}

func (s *Server) handleDeleteUserTOTP(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	totpUUID := r.PathValue("uuid")
	_, err := s.db.ExecContext(r.Context(),
		`DELETE FROM user_totps WHERE uuid = ? AND user_uuid = ?`, totpUUID, claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "TOTP 已删除"})
}

// ---- 主体信息处理器 ----

func (s *Server) handleListSubjectInfos(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	userUUID := claims.UserUUID
	if claims.Role == "admin" {
		if q := r.URL.Query().Get("user_uuid"); q != "" {
			userUUID = q
		}
	}
	infos, err := s.verifySvc.ListSubjectInfos(r.Context(), userUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"subject_infos": infos, "total": len(infos)})
}

func (s *Server) handleCreateSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var info storage.SubjectInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	info.UserUUID = claims.UserUUID
	if err := s.verifySvc.CreateSubjectInfo(r.Context(), &info); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, info)
}

func (s *Server) handleApproveSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.ApproveSubjectInfo(r.Context(), infoUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体信息已审核通过"})
}

func (s *Server) handleRejectSubjectInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.RejectSubjectInfo(r.Context(), infoUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "主体信息已拒绝"})
}

// ---- 扩展信息验证处理器 ----

func (s *Server) handleListExtensionInfos(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	infos, err := s.verifySvc.ListExtensionInfos(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"extension_infos": infos, "total": len(infos)})
}

func (s *Server) handleCreateExtensionInfo(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var info storage.ExtensionInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	info.UserUUID = claims.UserUUID
	if err := s.verifySvc.CreateExtensionInfo(r.Context(), &info); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"extension_info": info,
		"verify_token":   info.VerifyToken,
		"message":        fmt.Sprintf("请配置验证记录，token: %s", info.VerifyToken),
	})
}

func (s *Server) handleVerifyDNS(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.VerifyDNSTXT(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "DNS 验证通过"})
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.verifySvc.VerifyEmailCode(r.Context(), infoUUID, req.Code); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "邮箱验证通过"})
}

func (s *Server) handleDeleteExtensionInfo(w http.ResponseWriter, r *http.Request) {
	infoUUID := r.PathValue("uuid")
	if err := s.verifySvc.DeleteExtensionInfo(r.Context(), infoUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "扩展信息已删除"})
}

// ---- 证书订单与申请处理器 ----

func (s *Server) handleCreateCertOrder(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var order storage.CertOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	order.UserUUID = claims.UserUUID
	if err := s.workflowSvc.CreateOrder(r.Context(), &order); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleListCertOrders(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	orders, total, err := s.workflowSvc.ListOrders(r.Context(), claims.UserUUID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": orders, "total": total})
}

func (s *Server) handleCreateCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var app storage.CertApplication
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	app.UserUUID = claims.UserUUID
	if err := s.workflowSvc.CreateApplication(r.Context(), &app); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (s *Server) handleListCertApplications(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	userUUID := claims.UserUUID
	if claims.Role == "admin" {
		userUUID = "" // 管理员查看所有
	}
	statusFilter := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	apps, total, err := s.workflowSvc.ListApplications(r.Context(), userUUID, statusFilter, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"applications": apps, "total": total})
}

func (s *Server) handleApproveCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	appUUID := r.PathValue("uuid")
	if err := s.workflowSvc.ApproveApplication(r.Context(), appUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请已审批通过"})
}

func (s *Server) handleRejectCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	appUUID := r.PathValue("uuid")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.workflowSvc.RejectApplication(r.Context(), appUUID, claims.UserUUID, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请已拒绝"})
}

// ---- 证书增强管理处理器 ----

func (s *Server) handleListCertsFiltered(w http.ResponseWriter, r *http.Request) {
	userUUID := r.URL.Query().Get("user_uuid")
	caUUID := r.URL.Query().Get("ca_uuid")
	tmplUUID := r.URL.Query().Get("template_uuid")
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// 非管理员只能查看自己的证书
	claims := claimsFromCtx(r.Context())
	if claims.Role != "admin" {
		userUUID = claims.UserUUID
	}

	certRepo := storage.NewCertRepo(s.db)
	certs, total, err := certRepo.ListFiltered(r.Context(), userUUID, caUUID, tmplUUID, status, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"certs": certs,
		"total": total,
		"page":  page,
	})
}

// RevokeCertRequest 是吊销证书请求体。
type RevokeCertRequest struct {
	Reason int `json:"reason"` // RFC 5280 吊销原因码
}

func (s *Server) handleRevokeCertByUUID(w http.ResponseWriter, r *http.Request) {
	certUUID := r.PathValue("uuid")
	var req RevokeCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	certRepo := storage.NewCertRepo(s.db)

	// 获取证书信息
	cert, err := certRepo.GetByUUID(r.Context(), certUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	if cert.RevocationStatus == "revoked" {
		writeError(w, http.StatusBadRequest, "证书已被吊销")
		return
	}

	// 更新证书吊销状态
	if err := certRepo.Revoke(r.Context(), certUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 如果证书关联了 CA，将序列号加入 CA 的吊销列表
	if cert.CAUUID != "" {
		// 从证书内容中提取序列号（简化处理，使用 UUID 作为标识）
		if err := s.caSvc.RevokeCert(r.Context(), cert.CAUUID, certUUID, req.Reason); err != nil {
			slog.Warn("加入 CA 吊销列表失败", "cert_uuid", certUUID, "ca_uuid", cert.CAUUID, "error", err)
		}
	}

	claims := claimsFromCtx(r.Context())
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		CertUUID: certUUID,
		Action:   "revoke_cert",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已吊销"})
}

// AssignCertRequest 是分配证书到智能卡请求体。
type AssignCertRequest struct {
	TargetCardUUID string `json:"target_card_uuid"`
}

func (s *Server) handleAssignCertToCard(w http.ResponseWriter, r *http.Request) {
	certUUID := r.PathValue("uuid")
	var req AssignCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.TargetCardUUID == "" {
		writeError(w, http.StatusBadRequest, "目标智能卡 UUID 不能为空")
		return
	}

	certRepo := storage.NewCertRepo(s.db)

	// 验证证书存在
	if _, err := certRepo.GetByUUID(r.Context(), certUUID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// 验证目标卡片存在
	cardRepo := storage.NewCardRepo(s.db)
	if _, err := cardRepo.GetByUUID(r.Context(), req.TargetCardUUID); err != nil {
		writeError(w, http.StatusNotFound, "目标智能卡不存在")
		return
	}

	// 执行分配
	if err := certRepo.AssignToCard(r.Context(), certUUID, req.TargetCardUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	claims := claimsFromCtx(r.Context())
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID: claims.UserUUID,
		CertUUID: certUUID,
		CardUUID: req.TargetCardUUID,
		Action:   "assign_cert_to_card",
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已分配到智能卡"})
}

// ---- 公开服务处理器（无需认证）----

// handlePublicCRL 返回 CA 的 CRL 文件（DER 格式）。
func (s *Server) handlePublicCRL(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("caUUID")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	crl, err := s.revocationSvc.GetCRL(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.crl\"", caUUID))
	w.WriteHeader(http.StatusOK)
	w.Write(crl) //nolint:errcheck
}

// handlePublicOCSP 处理 OCSP 查询请求。
func (s *Server) handlePublicOCSP(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("caUUID")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	serialNumber := r.URL.Query().Get("serial")
	if serialNumber == "" {
		writeError(w, http.StatusBadRequest, "缺少 serial 参数")
		return
	}
	status, err := s.revocationSvc.QueryOCSPStatus(r.Context(), caUUID, serialNumber)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// handlePublicCAIssuer 返回 CA 证书 PEM（用于 AIA CAIssuer）。
func (s *Server) handlePublicCAIssuer(w http.ResponseWriter, r *http.Request) {
	caUUID := r.PathValue("caUUID")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	certPEM, err := s.revocationSvc.GetCAIssuerCert(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, certPEM)
}

// handleACMEDirectory 返回 ACME 目录 JSON。
func (s *Server) handleACMEDirectory(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if s.acmeSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "ACME 服务未启用")
		return
	}
	cfg, err := s.acmeSvc.GetConfigByPath(r.Context(), path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	baseURL := fmt.Sprintf("%s://%s/acme/%s", scheme(r), r.Host, path)
	directory := map[string]interface{}{
		"newNonce":   baseURL + "/new-nonce",
		"newAccount": baseURL + "/new-account",
		"newOrder":   baseURL + "/new-order",
		"revokeCert": baseURL + "/revoke-cert",
		"keyChange":  baseURL + "/key-change",
		"meta": map[string]interface{}{
			"caaIdentities":  []string{r.Host},
			"termsOfService": baseURL + "/terms",
			"website":        fmt.Sprintf("%s://%s", scheme(r), r.Host),
		},
		"caUUID": cfg.CAUUID,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	writeJSON(w, http.StatusOK, directory)
}

// handleACMENewNonce 返回 ACME Replay-Nonce。
func (s *Server) handleACMENewNonce(w http.ResponseWriter, r *http.Request) {
	if s.acmeSvc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	nonce, err := s.acmeSvc.GenerateNonce()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Replay-Nonce", nonce)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
}

// handleCTSubmit 接受证书 CT 提交请求。
func (s *Server) handleCTSubmit(w http.ResponseWriter, r *http.Request) {
	if s.ctSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "CT 服务未启用")
		return
	}
	var req struct {
		CertUUID    string `json:"cert_uuid"`
		CAUUID      string `json:"ca_uuid"`
		CTServer    string `json:"ct_server"`
		SubmittedBy string `json:"submitted_by"`
		CertDER     []byte `json:"cert_der"` // base64 编码的 DER 数据
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.CertUUID == "" || req.CTServer == "" {
		writeError(w, http.StatusBadRequest, "cert_uuid 和 ct_server 不能为空")
		return
	}
	entry, err := s.ctSvc.Submit(r.Context(), req.CertUUID, req.CAUUID, req.CTServer, req.SubmittedBy, req.CertDER)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

// handleCTQuery 按证书哈希查询 CT 记录。
func (s *Server) handleCTQuery(w http.ResponseWriter, r *http.Request) {
	if s.ctSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "CT 服务未启用")
		return
	}
	certHash := r.URL.Query().Get("cert_hash")
	if certHash == "" {
		writeError(w, http.StatusBadRequest, "缺少 cert_hash 参数")
		return
	}
	entries, err := s.ctSvc.QueryByCertHash(r.Context(), certHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries, "total": len(entries)})
}

// ---- 吊销服务管理处理器 ----

func (s *Server) handleListRevocationServices(w http.ResponseWriter, r *http.Request) {
	caUUID := r.URL.Query().Get("ca_uuid")
	configs, err := s.revocationSvc.ListServiceConfigs(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"services": configs, "total": len(configs)})
}

func (s *Server) handleCreateRevocationService(w http.ResponseWriter, r *http.Request) {
	var cfg revocation.ServiceConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.revocationSvc.CreateServiceConfig(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

func (s *Server) handleDeleteRevocationService(w http.ResponseWriter, r *http.Request) {
	cfgUUID := r.PathValue("uuid")
	if err := s.revocationSvc.DeleteServiceConfig(r.Context(), cfgUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "吊销服务配置已删除"})
}

// ---- ACME 配置管理处理器 ----

func (s *Server) handleListACMEConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := s.acmeSvc.ListConfigs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"configs": configs, "total": len(configs)})
}

func (s *Server) handleCreateACMEConfig(w http.ResponseWriter, r *http.Request) {
	var cfg acme.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.acmeSvc.CreateConfig(r.Context(), &cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cfg)
}

func (s *Server) handleDeleteACMEConfig(w http.ResponseWriter, r *http.Request) {
	cfgUUID := r.PathValue("uuid")
	if err := s.acmeSvc.DeleteConfig(r.Context(), cfgUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ACME 配置已删除"})
}

// ---- CT 记录管理处理器 ----

func (s *Server) handleListCTEntries(w http.ResponseWriter, r *http.Request) {
	certUUID := r.URL.Query().Get("cert_uuid")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	entries, total, err := s.ctSvc.List(r.Context(), certUUID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries, "total": total})
}

func (s *Server) handleDeleteCTEntry(w http.ResponseWriter, r *http.Request) {
	entryUUID := r.PathValue("uuid")
	if err := s.ctSvc.Delete(r.Context(), entryUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "CT 记录已删除"})
}

// scheme 返回请求的协议（http 或 https）。
func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}
