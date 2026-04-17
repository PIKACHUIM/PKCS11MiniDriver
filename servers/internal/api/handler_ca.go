package api

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/globaltrusts/server-card/internal/ca"
	"github.com/globaltrusts/server-card/internal/storage"
)

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
	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
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
	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
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
	w.Write(crlDER) //nolint:errcheck
}

// IssueCertRequest 是签发证书请求体。
type IssueCertRequest struct {
	KeyType          string   `json:"key_type"`
	ValidDays        int      `json:"valid_days"`
	CommonName       string   `json:"common_name"`
	Org              string   `json:"organization"`
	Country          string   `json:"country"`
	IsCA             bool     `json:"is_ca"`
	PathLen          int      `json:"path_len"`
	DNSNames         []string `json:"dns_names"`
	IPAddresses      []string `json:"ip_addresses"`
	EmailAddrs       []string `json:"email_addresses"`
	CardUUID         string   `json:"card_uuid"`          // 签发后存入的卡片 UUID（可选）
	IssuanceTmplUUID string   `json:"issuance_tmpl_uuid"` // 颁发模板 UUID（可选）
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
		CAUUID:           caUUID,
		Subject:          subject,
		KeyType:          req.KeyType,
		ValidDays:        req.ValidDays,
		IsCA:             req.IsCA,
		PathLen:          req.PathLen,
		KeyUsage:         keyUsage,
		ExtKeyUsage:      []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:         req.DNSNames,
		IPAddresses:      ips,
		EmailAddrs:       req.EmailAddrs,
		IssuanceTmplUUID: req.IssuanceTmplUUID,
	}

	resp, err := s.caSvc.IssueCert(r.Context(), issueReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	claims := claimsFromCtx(r.Context())

	// 如果指定了卡片 UUID，将证书存入卡片
	var certUUID string
	if req.CardUUID != "" {
		certRepo := storage.NewCertRepo(s.db)
		cert := &storage.Certificate{
			CardUUID:         req.CardUUID,
			UserUUID:         claims.UserUUID,
			CertType:         "x509",
			KeyType:          req.KeyType,
			CertContent:      []byte(resp.CertPEM),
			PrivateData:      resp.PrivateEnc,
			CAUUID:           caUUID,
			SerialNumber:     resp.SerialNumber,
			SerialHex:        resp.SerialNumber,
			SubjectDN:        resp.SubjectDN,
			IssuerDN:         resp.IssuerDN,
			NotBefore:        &resp.NotBefore,
			NotAfter:         &resp.NotAfter,
			IssuanceTmplUUID: req.IssuanceTmplUUID,
			RevocationStatus: "active",
		}
		if err := certRepo.Create(r.Context(), cert); err != nil {
			writeError(w, http.StatusInternalServerError, "保存证书失败: "+err.Error())
			return
		}
		certUUID = cert.UUID
	}

	// 写入审计日志
	s.auditLogRepo.Create(r.Context(), &storage.AuditLog{ //nolint:errcheck
		UserUUID:     claims.UserUUID,
		Action:       "issue_cert",
		ResourceType: "certificate",
		ResourceUUID: certUUID,
		Detail:       fmt.Sprintf(`{"ca_uuid":"%s","cn":"%s","serial":"%s"}`, caUUID, req.CommonName, resp.SerialNumber),
		IPAddress:    r.RemoteAddr,
	})

	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("issue_cert:%s:%s", caUUID, req.CommonName),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"cert_pem":      resp.CertPEM,
		"serial_number": resp.SerialNumber,
		"subject_dn":    resp.SubjectDN,
		"issuer_dn":     resp.IssuerDN,
		"not_before":    resp.NotBefore,
		"not_after":     resp.NotAfter,
		"cert_uuid":     certUUID,
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

// ---- 自定义路径的吊销服务处理器 ----

// lookupCAByPath 通过自定义路径查找 CA UUID。
func (s *Server) lookupCAByPath(ctx context.Context, serviceType, path string) (string, error) {
	var caUUID string
	err := s.db.QueryRowContext(ctx,
		`SELECT ca_uuid FROM revocation_services WHERE service_type = ? AND path = ? AND enabled = 1`,
		serviceType, path,
	).Scan(&caUUID)
	if err != nil {
		return "", fmt.Errorf("未找到路径 %s 对应的 %s 服务配置", path, serviceType)
	}
	return caUUID, nil
}

// handlePublicCRLByPath 通过自定义路径返回 CRL。
func (s *Server) handlePublicCRLByPath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	caUUID, err := s.lookupCAByPath(r.Context(), "crl", path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	crl, err := s.revocationSvc.GetCRL(r.Context(), caUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/pkix-crl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.crl\"", path))
	w.WriteHeader(http.StatusOK)
	w.Write(crl) //nolint:errcheck
}

// handlePublicOCSPByPath 通过自定义路径处理 OCSP 请求。
func (s *Server) handlePublicOCSPByPath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	caUUID, err := s.lookupCAByPath(r.Context(), "ocsp", path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
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

// handlePublicCAIssuerByPath 通过自定义路径返回 CA 证书。
func (s *Server) handlePublicCAIssuerByPath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if s.revocationSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "吊销服务未启用")
		return
	}
	caUUID, err := s.lookupCAByPath(r.Context(), "caissuer", path)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
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
