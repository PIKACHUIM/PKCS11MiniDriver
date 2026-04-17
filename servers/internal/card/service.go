// Package card 提供云端卡片的业务逻辑。
// 私钥在服务端加密存储，签名/解密操作在服务端执行，私钥不离开服务器。
package card

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/globaltrusts/server-card/internal/storage"
)

// Service 提供云端卡片的业务操作。
type Service struct {
	cardRepo *storage.CardRepo
	certRepo *storage.CertRepo
	// masterKey 是服务端主密钥，用于加密所有私钥
	// 生产环境应从 HSM 或密钥管理服务获取
	masterKey []byte
}

// NewService 创建卡片服务。
// masterKey 是 32 字节服务端主密钥。
func NewService(cardRepo *storage.CardRepo, certRepo *storage.CertRepo, masterKey []byte) *Service {
	return &Service{
		cardRepo:  cardRepo,
		certRepo:  certRepo,
		masterKey: masterKey,
	}
}

// CreateCard 创建云端卡片。
func (s *Service) CreateCard(ctx context.Context, userUUID, cardName, remark string) (*storage.Card, error) {
	card := &storage.Card{
		UserUUID: userUUID,
		CardName: cardName,
		Remark:   remark,
	}
	if err := s.cardRepo.Create(ctx, card); err != nil {
		return nil, fmt.Errorf("创建卡片失败: %w", err)
	}
	return card, nil
}

// GetCard 获取卡片（验证归属）。
func (s *Service) GetCard(ctx context.Context, cardUUID, userUUID string) (*storage.Card, error) {
	card, err := s.cardRepo.GetByUUID(ctx, cardUUID)
	if err != nil {
		return nil, err
	}
	if card.UserUUID != userUUID {
		return nil, fmt.Errorf("无权访问此卡片")
	}
	return card, nil
}

// ListCards 列出用户的所有卡片。
func (s *Service) ListCards(ctx context.Context, userUUID string) ([]*storage.Card, error) {
	return s.cardRepo.ListByUser(ctx, userUUID)
}

// DeleteCard 删除卡片（验证归属）。
func (s *Service) DeleteCard(ctx context.Context, cardUUID, userUUID string) error {
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return err
	}
	return s.cardRepo.Delete(ctx, cardUUID)
}

// GenerateKeyPair 在云端卡片中生成密钥对。
// 私钥加密存储，公钥/证书返回给调用方。
func (s *Service) GenerateKeyPair(ctx context.Context, cardUUID, userUUID, keyType, remark string) (*storage.Certificate, error) {
	// 验证卡片归属
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return nil, err
	}

	// 生成密钥对
	privKey, pubDER, err := generateKeyPair(keyType)
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 序列化私钥
	privDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("序列化私钥失败: %w", err)
	}

	// 用服务端主密钥加密私钥
	encPriv, err := encryptWithMasterKey(s.masterKey, privDER)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %w", err)
	}

	cert := &storage.Certificate{
		CardUUID:    cardUUID,
		CertType:    "x509",
		KeyType:     keyType,
		CertContent: pubDER,
		PrivateData: encPriv,
		Remark:      remark,
	}
	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("保存证书失败: %w", err)
	}

	// 返回时清空私钥字段
	cert.PrivateData = nil
	return cert, nil
}

// ImportCert 导入证书（仅公钥/证书内容，无私钥）。
func (s *Service) ImportCert(ctx context.Context, cardUUID, userUUID, certType, keyType, remark string, certContent []byte) (*storage.Certificate, error) {
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return nil, err
	}

	cert := &storage.Certificate{
		CardUUID:    cardUUID,
		CertType:    certType,
		KeyType:     keyType,
		CertContent: certContent,
		Remark:      remark,
	}
	if err := s.certRepo.Create(ctx, cert); err != nil {
		return nil, fmt.Errorf("导入证书失败: %w", err)
	}
	return cert, nil
}

// ListCerts 列出卡片的所有证书（不含私钥）。
func (s *Service) ListCerts(ctx context.Context, cardUUID, userUUID string) ([]*storage.Certificate, error) {
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return nil, err
	}
	return s.certRepo.ListByCard(ctx, cardUUID)
}

// DeleteCert 删除证书。
func (s *Service) DeleteCert(ctx context.Context, certUUID, cardUUID, userUUID string) error {
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return err
	}
	return s.certRepo.Delete(ctx, certUUID)
}

// Sign 使用云端私钥签名（私钥不离开服务器）。
func (s *Service) Sign(ctx context.Context, certUUID, cardUUID, userUUID, mechanism string, data []byte) ([]byte, error) {
	// 验证卡片归属
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return nil, err
	}

	// 获取证书（含私钥）
	cert, err := s.certRepo.GetByUUID(ctx, certUUID)
	if err != nil {
		return nil, err
	}
	if cert.CardUUID != cardUUID {
		return nil, fmt.Errorf("证书不属于此卡片")
	}

	// 解密私钥
	privDER, err := decryptWithMasterKey(s.masterKey, cert.PrivateData)
	if err != nil {
		return nil, fmt.Errorf("解密私钥失败: %w", err)
	}

	privKey, err := parsePrivateKey(privDER)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	return signData(privKey, mechanism, data)
}

// Decrypt 使用云端私钥解密（私钥不离开服务器）。
func (s *Service) Decrypt(ctx context.Context, certUUID, cardUUID, userUUID, mechanism string, ciphertext []byte) ([]byte, error) {
	if _, err := s.GetCard(ctx, cardUUID, userUUID); err != nil {
		return nil, err
	}

	cert, err := s.certRepo.GetByUUID(ctx, certUUID)
	if err != nil {
		return nil, err
	}
	if cert.CardUUID != cardUUID {
		return nil, fmt.Errorf("证书不属于此卡片")
	}

	privDER, err := decryptWithMasterKey(s.masterKey, cert.PrivateData)
	if err != nil {
		return nil, fmt.Errorf("解密私钥失败: %w", err)
	}

	privKey, err := parsePrivateKey(privDER)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}

	rsaKey, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("解密仅支持 RSA 私钥")
	}

	switch mechanism {
	case "RSA_PKCS":
		return rsa.DecryptPKCS1v15(rand.Reader, rsaKey, ciphertext)
	case "RSA_OAEP":
		return rsa.DecryptOAEP(sha256.New(), rand.Reader, rsaKey, ciphertext, nil)
	default:
		return nil, fmt.Errorf("不支持的解密算法: %s", mechanism)
	}
}

// ExportAsPKCS12 将证书和私钥导出为 PEM 格式的组合数据。
// 注意：当前版本不支持 PKCS12 打包，返回 PEM 格式的私钥+证书。
// 如需 PKCS12 支持，请添加 software.sslmate.com/src/go-pkcs12 依赖。
func (s *Service) ExportAsPKCS12(ctx context.Context, cert *storage.Certificate, password string) ([]byte, error) {
	// 解密私钥
	privDER, err := decryptWithMasterKey(s.masterKey, cert.PrivateData)
	if err != nil {
		return nil, fmt.Errorf("解密私钥失败: %w", err)
	}

	// 将私钥编码为 PEM
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	})

	// 将证书编码为 PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.CertContent,
	})

	// 组合输出
	result := append(privPEM, certPEM...)
	return result, nil
}

// DecryptPrivateKey 解密证书的私钥（供外部使用）。
func (s *Service) DecryptPrivateKey(ctx context.Context, cert *storage.Certificate) (crypto.PrivateKey, error) {
	privDER, err := decryptWithMasterKey(s.masterKey, cert.PrivateData)
	if err != nil {
		return nil, fmt.Errorf("解密私钥失败: %w", err)
	}
	return parsePrivateKey(privDER)
}

// EncryptData 使用主密钥加密数据（供外部使用）。
func (s *Service) EncryptData(data []byte) ([]byte, error) {
	return encryptWithMasterKey(s.masterKey, data)
}

// DecryptData 使用主密钥解密数据（供外部使用）。
func (s *Service) DecryptData(blob []byte) ([]byte, error) {
	return decryptWithMasterKey(s.masterKey, blob)
}

// generateKeyPair 生成密钥对，返回私钥和公钥 DER。
func generateKeyPair(keyType string) (crypto.PrivateKey, []byte, error) {
	var privKey crypto.PrivateKey
	var err error

	switch keyType {
	case "rsa2048":
		privKey, err = rsa.GenerateKey(rand.Reader, 2048)
	case "rsa4096":
		privKey, err = rsa.GenerateKey(rand.Reader, 4096)
	case "ec256":
		privKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ec384":
		privKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "ec521":
		privKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return nil, nil, fmt.Errorf("不支持的密钥类型: %s", keyType)
	}
	if err != nil {
		return nil, nil, err
	}

	// 提取公钥 DER
	var pubKey crypto.PublicKey
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		pubKey = &k.PublicKey
	case *ecdsa.PrivateKey:
		pubKey = &k.PublicKey
	}

	pubDER, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化公钥失败: %w", err)
	}

	return privKey, pubDER, nil
}

// encryptWithMasterKey 用服务端主密钥 AES-256-GCM 加密数据。
func encryptWithMasterKey(masterKey, data []byte) ([]byte, error) {
	return aesGCMEncrypt(masterKey, data)
}

// decryptWithMasterKey 用服务端主密钥 AES-256-GCM 解密数据。
func decryptWithMasterKey(masterKey, blob []byte) ([]byte, error) {
	return aesGCMDecrypt(masterKey, blob)
}

// parsePrivateKey 解析 DER 格式私钥。
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("无法解析私钥格式")
}

// signData 使用私钥签名。
func signData(privKey crypto.PrivateKey, mechanism string, data []byte) ([]byte, error) {
	switch key := privKey.(type) {
	case *rsa.PrivateKey:
		return signRSA(key, mechanism, data)
	case *ecdsa.PrivateKey:
		return signECDSA(key, mechanism, data)
	default:
		return nil, fmt.Errorf("不支持的私钥类型: %T", privKey)
	}
}

func signRSA(key *rsa.PrivateKey, mechanism string, data []byte) ([]byte, error) {
	switch mechanism {
	case "SHA256_RSA_PKCS":
		h := sha256.Sum256(data)
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	case "SHA384_RSA_PKCS":
		h := sha512.Sum384(data)
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA384, h[:])
	case "SHA512_RSA_PKCS":
		h := sha512.Sum512(data)
		return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA512, h[:])
	case "SHA256_RSA_PSS":
		h := sha256.Sum256(data)
		return rsa.SignPSS(rand.Reader, key, crypto.SHA256, h[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto})
	default:
		return nil, fmt.Errorf("不支持的 RSA 签名算法: %s", mechanism)
	}
}

func signECDSA(key *ecdsa.PrivateKey, mechanism string, data []byte) ([]byte, error) {
	var digest []byte
	switch mechanism {
	case "ECDSA":
		digest = data
	case "ECDSA_SHA256":
		h := sha256.Sum256(data)
		digest = h[:]
	case "ECDSA_SHA384":
		h := sha512.Sum384(data)
		digest = h[:]
	case "ECDSA_SHA512":
		h := sha512.Sum512(data)
		digest = h[:]
	default:
		return nil, fmt.Errorf("不支持的 ECDSA 签名算法: %s", mechanism)
	}

	r, s, err := ecdsa.Sign(rand.Reader, key, digest)
	if err != nil {
		return nil, fmt.Errorf("ECDSA 签名失败: %w", err)
	}

	keyBytes := (key.Curve.Params().BitSize + 7) / 8
	result := make([]byte, 2*keyBytes)
	rBytes, sBytes := r.Bytes(), s.Bytes()
	copy(result[keyBytes-len(rBytes):keyBytes], rBytes)
	copy(result[2*keyBytes-len(sBytes):], sBytes)
	return result, nil
}
