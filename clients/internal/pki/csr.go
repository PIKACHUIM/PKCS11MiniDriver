package pki

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
)

// CSRRequest 是 CSR 生成请求参数。
type CSRRequest struct {
	CommonName   string   `json:"common_name"`
	Organization string   `json:"organization"`
	OrgUnit      string   `json:"org_unit"`
	Country      string   `json:"country"`
	Province     string   `json:"province"`
	Locality     string   `json:"locality"`
	KeyType      string   `json:"key_type"` // rsa2048/rsa4096/ec256/ec384/ec521
	DNSNames     []string `json:"dns_names"`
	IPAddresses  []string `json:"ip_addresses"`
	Emails       []string `json:"emails"`
}

// CSRResult 是 CSR 生成结果。
type CSRResult struct {
	CSRPEM  []byte `json:"csr_pem"`
	CSRDER  []byte `json:"csr_der"`
	KeyPEM  []byte `json:"key_pem"`
	KeyDER  []byte `json:"key_der"`
}

// GenerateCSR 生成 CSR（证书签名请求）。
// 密钥对在本地生成，CSR 使用私钥签名，确保片上生成的可信性。
func GenerateCSR(req *CSRRequest) (*CSRResult, error) {
	// 生成密钥对
	privKey, _, err := generateKeyPair(req.KeyType)
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 构建主体
	subject := pkix.Name{CommonName: req.CommonName}
	if req.Organization != "" {
		subject.Organization = []string{req.Organization}
	}
	if req.OrgUnit != "" {
		subject.OrganizationalUnit = []string{req.OrgUnit}
	}
	if req.Country != "" {
		subject.Country = []string{req.Country}
	}
	if req.Province != "" {
		subject.Province = []string{req.Province}
	}
	if req.Locality != "" {
		subject.Locality = []string{req.Locality}
	}

	// 构建 CSR 模板
	template := &x509.CertificateRequest{
		Subject:        subject,
		DNSNames:       req.DNSNames,
		EmailAddresses: req.Emails,
	}

	// 解析 IP 地址
	for _, ipStr := range req.IPAddresses {
		if ip := net.ParseIP(ipStr); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	// 使用私钥签名 CSR
	signer, ok := privKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("私钥不支持签名操作")
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, signer)
	if err != nil {
		return nil, fmt.Errorf("创建 CSR 失败: %w", err)
	}

	// 验证 CSR 签名
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, fmt.Errorf("解析 CSR 失败: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR 签名验证失败: %w", err)
	}

	// 编码 PEM
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	keyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("编码私钥失败: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	return &CSRResult{
		CSRPEM: csrPEM,
		CSRDER: csrDER,
		KeyPEM: keyPEM,
		KeyDER: keyDER,
	}, nil
}
