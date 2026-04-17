// Package ca - 证书签发逻辑。
package ca

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/globaltrusts/server-card/internal/storage"
)

// IssueRequest 是证书签发请求。
type IssueRequest struct {
	CAUUID      string            // 签发 CA 的 UUID
	Subject     pkix.Name         // 证书主体
	KeyType     string            // 密钥类型（rsa2048/rsa4096/ec256/ec384/ec521）
	ValidDays   int               // 有效期（天）
	IsCA        bool              // 是否为 CA 证书
	PathLen     int               // CA 路径长度（仅 IsCA=true 时有效）
	KeyUsage    x509.KeyUsage     // 密钥用途
	ExtKeyUsage []x509.ExtKeyUsage // 扩展密钥用途
	DNSNames    []string          // SAN DNS 名称
	IPAddresses []net.IP          // SAN IP 地址
	EmailAddrs  []string          // SAN 邮箱

	// 模板约束（可选，签发前验证）
	IssuanceTmplUUID string // 颁发模板 UUID（用于约束验证）

	// 证书拓展模板（可选，签发时写入扩展）
	CRLDistPoints  []string // CRL 分发点
	OCSPServers    []string // OCSP 服务器
	AIAIssuers     []string // AIA 颁发者
	CTServers      []string // CT 服务器
	EVPolicyOID    string   // EV 策略 OID
}

// IssueResponse 是证书签发响应。
type IssueResponse struct {
	CertPEM      string    // 签发的证书 PEM
	CertDER      []byte    // 签发的证书 DER
	PrivateEnc   []byte    // 加密的私钥
	SerialNumber string    // 证书序列号（十六进制）
	SubjectDN    string    // 主体 DN
	IssuerDN     string    // 颁发者 DN
	NotBefore    time.Time // 生效时间
	NotAfter     time.Time // 失效时间
}

// IssueCert 使用 CA 签发证书。
func (s *Service) IssueCert(ctx context.Context, req *IssueRequest) (*IssueResponse, error) {
	// 获取 CA
	ca, err := s.GetByUUID(ctx, req.CAUUID)
	if err != nil {
		return nil, fmt.Errorf("获取 CA 失败: %w", err)
	}
	if ca.Status != "active" {
		return nil, fmt.Errorf("CA 状态不可用: %s", ca.Status)
	}

	// 解析 CA 证书
	caCert, err := parseCertPEM(ca.CertPEM)
	if err != nil {
		return nil, fmt.Errorf("解析 CA 证书失败: %w", err)
	}

	// 解密 CA 私钥
	caKey, err := decryptPrivateKey(s.masterKey, ca.PrivateEnc)
	if err != nil {
		return nil, fmt.Errorf("解密 CA 私钥失败: %w", err)
	}

	// 生成新密钥对
	privKey, err := generateKey(req.KeyType)
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 生成随机序列号（128 位）
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("生成序列号失败: %w", err)
	}

	now := time.Now()
	notAfter := now.AddDate(0, 0, req.ValidDays)

	// 限制有效期不超过 CA 有效期
	if notAfter.After(ca.NotAfter) {
		notAfter = ca.NotAfter
	}

	// 构建证书模板
	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               req.Subject,
		NotBefore:             now,
		NotAfter:              notAfter,
		KeyUsage:              req.KeyUsage,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  req.IsCA,
		DNSNames:              req.DNSNames,
		IPAddresses:           req.IPAddresses,
		EmailAddresses:        req.EmailAddrs,
	}

	if req.IsCA {
		template.MaxPathLen = req.PathLen
		template.MaxPathLenZero = req.PathLen == 0
	}

	// 写入证书拓展模板的扩展信息
	if len(req.CRLDistPoints) > 0 {
		template.CRLDistributionPoints = req.CRLDistPoints
	}
	if len(req.OCSPServers) > 0 {
		template.OCSPServer = req.OCSPServers
	}
	if len(req.AIAIssuers) > 0 {
		template.IssuingCertificateURL = req.AIAIssuers
	}

	// 获取公钥
	pubKey := publicKey(privKey)

	// 签发证书
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, pubKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("签发证书失败: %w", err)
	}

	// 解析签发后的证书获取完整元数据
	issuedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("解析签发证书失败: %w", err)
	}

	// 编码为 PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// 加密私钥
	privDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("序列化私钥失败: %w", err)
	}
	privEnc, err := encryptPrivateKey(s.masterKey, privDER)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %w", err)
	}

	// 递增 CA 签发计数
	if err := s.IncrementIssuedCount(ctx, req.CAUUID); err != nil {
		return nil, fmt.Errorf("更新签发计数失败: %w", err)
	}

	return &IssueResponse{
		CertPEM:      string(certPEM),
		CertDER:      certDER,
		PrivateEnc:   privEnc,
		SerialNumber: fmt.Sprintf("%x", serialNumber),
		SubjectDN:    issuedCert.Subject.String(),
		IssuerDN:     issuedCert.Issuer.String(),
		NotBefore:    issuedCert.NotBefore,
		NotAfter:     issuedCert.NotAfter,
	}, nil
}

// CreateSelfSignedCA 创建自签名根 CA。
func (s *Service) CreateSelfSignedCA(ctx context.Context, name string, subject pkix.Name, keyType string, validYears int) (*storage.CA, error) {
	// 生成密钥对
	privKey, err := generateKey(keyType)
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 生成随机序列号
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("生成序列号失败: %w", err)
	}

	now := time.Now()
	notAfter := now.AddDate(validYears, 0, 0)

	// 构建自签名 CA 证书模板
	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             now,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        false,
		MaxPathLen:            -1,
	}

	pubKey := publicKey(privKey)

	// 自签名
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pubKey, privKey)
	if err != nil {
		return nil, fmt.Errorf("创建自签名证书失败: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// 加密私钥
	privDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("序列化私钥失败: %w", err)
	}
	privEnc, err := encryptPrivateKey(s.masterKey, privDER)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %w", err)
	}

	ca := &storage.CA{
		Name:       name,
		CertPEM:    string(certPEM),
		PrivateEnc: privEnc,
		Status:     "active",
		NotBefore:  now,
		NotAfter:   notAfter,
	}

	if err := s.Create(ctx, ca); err != nil {
		return nil, fmt.Errorf("保存 CA 失败: %w", err)
	}

	return ca, nil
}

// ---- 内部工具函数 ----

// generateKey 生成密钥对。
func generateKey(keyType string) (crypto.PrivateKey, error) {
	switch keyType {
	case "rsa2048":
		return rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096":
		return rsa.GenerateKey(rand.Reader, 4096)
	case "ec256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ec384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "ec521":
		return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return nil, fmt.Errorf("不支持的密钥类型: %s", keyType)
	}
}

// publicKey 从私钥提取公钥。
func publicKey(priv crypto.PrivateKey) crypto.PublicKey {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}
