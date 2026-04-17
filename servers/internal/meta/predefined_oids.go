package meta

// PredefinedOID 描述一个预置 OID 条目。
type PredefinedOID struct {
	OID   string `json:"oid"`
	Short string `json:"short"` // 简短标识（如 "serverAuth"）
	Name  string `json:"name"`  // 可读名称
}

// predefinedOIDs 按分类组织的预置 OID 库（基于 XCA oids.txt）。
// 前端下拉选择、主体扩展字段时会引用。
var predefinedOIDs = map[string][]PredefinedOID{
	"core": {
		{OID: "2.5.29.32.0", Short: "anyPolicy", Name: "Any Policy"},
		{OID: "2.5.29.37.0", Short: "anyExtendedUsage", Name: "Any Extended Usage"},
	},
	"subject_extend": {
		{OID: "2.5.4.17", Short: "postalCode", Name: "postalCode"},
		{OID: "2.5.4.9", Short: "streetAddress", Name: "streetAddress"},
		{OID: "2.5.4.5", Short: "serialNumber", Name: "serialNumber"},
		{OID: "2.5.4.15", Short: "businessCategory", Name: "businessCategory"},
		{OID: "2.5.4.44", Short: "generationQualifier", Name: "Generation Qualifier"},
		{OID: "2.5.4.45", Short: "x500UniqueIdentifier", Name: "x500 Unique Identifier"},
		{OID: "2.5.4.65", Short: "pseudonym", Name: "Pseudonym (Other Name)"},
		{OID: "0.2.262.1.10.7.20", Short: "nameDistinguisher", Name: "Name distinguisher"},
	},
	"ms_extend": {
		{OID: "1.3.6.1.4.1.311.60.2.1.1", Short: "IncLocalityName", Name: "Inc Locality Name"},
		{OID: "1.3.6.1.4.1.311.60.2.1.2", Short: "IncStateOrProvinceName", Name: "Inc Province Name"},
		{OID: "1.3.6.1.4.1.311.60.2.1.3", Short: "IncCountryName", Name: "Inc CountryName"},
	},
	"ev": {
		{OID: "2.23.140.1.1", Short: "evExtend", Name: "Extended Validation (EV) Extend"},
		{OID: "2.23.140.1.3", Short: "EVSign", Name: "Extended Validation (EV) Code Signing"},
		{OID: "2.16.156.112554.3", Short: "EVServer", Name: "CFCA Extended Validation (EV) Server Cert"},
		{OID: "2.23.140.1.4.1", Short: "ptcsc", Name: "Publicly-Trusted Code Signing Certificates"},
	},
	"netscape": {
		{OID: "2.16.840.1.113730.1", Short: "selfEV", Name: "Certificates EV Extend"},
		{OID: "2.16.840.1.113730.4.1", Short: "exportApproved", Name: "Certificates EV Export Approved"},
	},
	"ssl": {
		{OID: "1.3.6.1.5.5.7.3.1", Short: "serverAuth", Name: "[SSL3] Server Auth"},
		{OID: "1.3.6.1.5.5.7.3.2", Short: "clientAuth", Name: "[SSL3] Client Auth"},
	},
	"code_sign": {
		{OID: "1.3.6.1.5.5.7.3.3", Short: "codeSigning", Name: "[Code] Code Signing"},
		{OID: "1.3.6.1.4.1.311.61.1.1", Short: "msKernSign", Name: "[Code] Microsoft Kernel Code Signing"},
		{OID: "1.3.6.1.4.1.311.10.3.39", Short: "msEVWHQL", Name: "[Code] Microsoft EV WHQL Verification"},
		{OID: "1.3.6.1.4.1.311.10.3.5", Short: "msWHQL", Name: "[Code] Microsoft SYS WHQL Verification"},
		{OID: "1.3.6.1.4.1.311.10.3.7", Short: "msWHQLOEM", Name: "[Code] Microsoft OEM WHQL Verification"},
		{OID: "1.3.6.1.4.1.311.10.3.6", Short: "msNT5C", Name: "[Code] Microsoft NT System Component Verification"},
		{OID: "1.3.6.1.4.1.311.10.3.8", Short: "msENTC", Name: "[Code] Microsoft Embedded Component Verification"},
	},
	"email": {
		{OID: "1.3.6.1.5.5.7.3.4", Short: "emailProtection", Name: "[Mail] Email Protection"},
		{OID: "1.3.6.1.4.1.311.21.19", Short: "msDSER", Name: "[Mail] Microsoft DS Email Replication"},
	},
	"document": {
		{OID: "1.2.840.113583.1.1.5", Short: "adobePDFSigning", Name: "[Docs] Adobe PDF Signing"},
		{OID: "1.3.6.1.4.1.311.10.3.12", Short: "msofficeSigning", Name: "[Docs] Microsoft Office Signing"},
	},
	"ipsec": {
		{OID: "1.3.6.1.5.5.7.3.5", Short: "ipsecEndSystem", Name: "[IPSE] IP Security End Entity"},
		{OID: "1.3.6.1.5.5.7.3.6", Short: "ipsecTunnel", Name: "[IPSE] IP Security Tunnel"},
		{OID: "1.3.6.1.5.5.7.3.7", Short: "ipsecUser", Name: "[IPSE] IP Security User"},
		{OID: "1.3.6.1.5.5.8.2.2", Short: "iKEIntermediate", Name: "[IPSE] IP Security End Entity IKE"},
		{OID: "1.3.6.1.5.5.7.3.17", Short: "ipsecIKE", Name: "[IPSE] IP Security IKE Tunnel"},
	},
	"server": {
		{OID: "1.3.6.1.5.5.7.3.8", Short: "timeStamping", Name: "[Time] Time Stamping"},
		{OID: "1.3.6.1.5.5.7.3.9", Short: "OCSPSigning", Name: "[OCSP] OCSP Signing"},
	},
	"scvp": {
		{OID: "1.3.6.1.5.5.7.3.10", Short: "pkixKeyPurpose", Name: "PKIX Key Purpose"},
		{OID: "1.3.6.1.5.5.7.3.11", Short: "sbgpCertAAServerAuth", Name: "SBGP Cert AAServer Auth"},
		{OID: "1.3.6.1.5.5.7.3.12", Short: "scvpRes", Name: "SCVP Responder"},
		{OID: "1.3.6.1.5.5.7.3.15", Short: "SCVPS", Name: "SCVP Server"},
		{OID: "1.3.6.1.5.5.7.3.16", Short: "SCVPC", Name: "SCVP Client"},
	},
	"rfc4334": {
		{OID: "1.3.6.1.5.2.3.5", Short: "pkInitKDC", Name: "Signing KDC Response"},
		{OID: "1.3.6.1.5.5.7.3.13", Short: "id-kp-eapOverPPP", Name: "EAP Over PPP"},
		{OID: "1.3.6.1.5.5.7.3.14", Short: "id-kp-eapOverLAN", Name: "EAP Over LAN"},
		{OID: "1.3.6.1.5.5.7.3.18", Short: "CAPWAPAC", Name: "CAPWAP AC"},
		{OID: "1.3.6.1.5.5.7.3.19", Short: "CAPWAPWTP", Name: "CAPWAP WTP"},
		{OID: "1.3.6.1.5.5.7.3.20", Short: "SIP", Name: "SIP"},
	},
	"ssh": {
		{OID: "1.3.6.1.5.5.7.3.21", Short: "sshc", Name: "[SSH] SSH Client"},
		{OID: "1.3.6.1.5.5.7.3.22", Short: "sshs", Name: "[SSH] SSH Server"},
		{OID: "1.3.6.1.5.2.3.4", Short: "pkInitClientAuth", Name: "[SSH] PKINIT Client Auth"},
	},
	"send": {
		{OID: "1.3.6.1.5.5.7.3.23", Short: "sendRouter", Name: "Send Router"},
		{OID: "1.3.6.1.5.5.7.3.24", Short: "sendProxy", Name: "Send Proxy"},
		{OID: "1.3.6.1.5.5.7.3.25", Short: "sendOwner", Name: "Send Owner"},
		{OID: "1.3.6.1.5.5.7.3.26", Short: "sendProxyOwner", Name: "Send Proxy Owner"},
	},
	"cmc": {
		{OID: "1.3.6.1.5.5.7.3.27", Short: "cmcCA", Name: "CMC CA Extended Key Usage"},
		{OID: "1.3.6.1.5.5.7.3.28", Short: "cmcRA", Name: "CMC Registration Authorities Key Usage"},
		{OID: "1.3.6.1.5.5.7.3.29", Short: "cmcArchive", Name: "CMC Archive Servers Extended Key Usage"},
		{OID: "1.3.6.1.5.5.7.3.30", Short: "BGPSEC", Name: "BGP SEC"},
	},
	"ms_efs": {
		{OID: "1.3.6.1.4.1.311.10.3.4", Short: "msEFS", Name: "[EFS] Microsoft EFS System"},
		{OID: "1.3.6.1.4.1.311.10.3.4.1", Short: "msEFSFR", Name: "[EFS] Microsoft EFS File Recovery"},
		{OID: "1.3.6.1.4.1.311.10.3.11", Short: "msKR", Name: "[EFS] Microsoft Key Recovery"},
	},
	"ms_bitlocker": {
		{OID: "1.3.6.1.4.1.311.67.1.1", Short: "driveEncryption", Name: "[Bit] Microsoft BitLocker Drive Encryption"},
		{OID: "1.3.6.1.4.1.311.67.1.2", Short: "dataRecoveryAgent", Name: "[Bit] Microsoft BitLocker Data Recovery Agent"},
	},
	"ms_ca": {
		{OID: "1.3.6.1.4.1.311", Short: "msALL", Name: "[MS CA] Microsoft All Usage"},
		{OID: "1.3.6.1.4.1.311.21.1", Short: "MsCaV", Name: "[MS CA] Microsoft CA Version"},
		{OID: "1.3.6.1.4.1.311.21.5", Short: "msCAE", Name: "[MS CA] Microsoft CA Exchange"},
		{OID: "1.3.6.1.4.1.311.21.3", Short: "msCVB", Name: "[MS CA] Microsoft CRL Virtual Base"},
		{OID: "1.3.6.1.4.1.311.21.4", Short: "msCNP", Name: "[MS CA] Microsoft CRL Next Publish"},
		{OID: "1.3.6.1.4.1.311.21.6", Short: "msKRA", Name: "[MS CA] Microsoft Key Recovery Agent"},
		{OID: "1.3.6.1.4.1.311.21.2", Short: "msCPCH", Name: "[MS CA] Microsoft Certsrv Previous Cert Hash"},
		{OID: "1.3.6.1.4.1.311.21.10", Short: "msACP", Name: "[MS CA] Microsoft Application Certificate Policy"},
		{OID: "1.3.6.1.4.1.311.10.3.3", Short: "msSGC", Name: "[MS CA] Microsoft Server Gated Crypto"},
	},
	"ms_ca_signing": {
		{OID: "1.3.6.1.4.1.311.10.3.1", Short: "msCTLSign", Name: "[MS CA] Microsoft CTL Signing"},
		{OID: "1.3.6.1.4.1.311.10.3.2", Short: "msTSS", Name: "[MS CA] Microsoft Time Stamp Signing"},
		{OID: "1.3.6.1.4.1.311.10.3.9", Short: "msRLS", Name: "[MS CA] Microsoft Root List Signer"},
		{OID: "1.3.6.1.4.1.311.10.3.10", Short: "msQS", Name: "[MS CA] Microsoft Qualified Subordination"},
		{OID: "1.3.6.1.4.1.311.10.3.13", Short: "msLSV", Name: "[MS CA] Microsoft Lifetime Signing"},
	},
	"ms_obj_signing": {
		{OID: "1.3.6.1.4.1.311.2.1.21", Short: "msCodeInd", Name: "[MS OS] Microsoft Individual Object Signing"},
		{OID: "1.3.6.1.4.1.311.2.1.22", Short: "msCodeCom", Name: "[MS OS] Microsoft Commercial Object Signing"},
	},
	"ms_media": {
		{OID: "1.3.6.1.4.1.311.10.5.1", Short: "msDRM", Name: "[MS ME] Microsoft Digital Right Verification"},
		{OID: "1.3.6.1.4.1.311.10.6.1", Short: "msL", Name: "[MS ME] Microsoft Licenses Client"},
		{OID: "1.3.6.1.4.1.311.10.6.2", Short: "msLS", Name: "[MS ME] Microsoft Licenses Server"},
	},
	"ms_ad": {
		{OID: "1.3.6.1.4.1.311.20.2", Short: "dom", Name: "[MS AD] Microsoft Domain Controller"},
		{OID: "1.3.6.1.4.1.311.20.2.1", Short: "msEA", Name: "[MS AD] Microsoft Enrollment Agent"},
		{OID: "1.3.6.1.4.1.311.20.2.2", Short: "msSmartcardLogin", Name: "[MS AD] Microsoft Smart Card Logon"},
		{OID: "1.3.6.1.4.1.311.20.2.3", Short: "msUPN", Name: "[MS AD] Microsoft Universal Principal Name"},
	},
	"ms_ek": {
		{OID: "1.3.6.1.4.1.311.21.30", Short: "ekVerified", Name: "[MS CA] Microsoft EK Verified"},
		{OID: "1.3.6.1.4.1.311.21.31", Short: "ekCertVerified", Name: "[MS CA] Microsoft EK Certificate Verified"},
		{OID: "1.3.6.1.4.1.311.21.32", Short: "ekUserVerified", Name: "[MS CA] Microsoft EK User Trusted Verified"},
	},
	"pika": {
		{OID: "1.3.6.1.4.1.37476.9000.173", Short: "pikaRootCA", Name: "[Pika] Pikachu Root CA"},
		{OID: "1.3.6.1.4.1.37476.9000.173.1", Short: "pikaExtended", Name: "[Pika] Pikachu Extended Validation"},
	},
}

// GetPredefinedOIDs 返回指定分类的 OID 列表。
// category 为空时返回全部分类。
func GetPredefinedOIDs(category string) map[string][]PredefinedOID {
	if category == "" {
		// 返回副本（浅拷贝 slice 头）
		out := make(map[string][]PredefinedOID, len(predefinedOIDs))
		for k, v := range predefinedOIDs {
			out[k] = v
		}
		return out
	}
	if list, ok := predefinedOIDs[category]; ok {
		return map[string][]PredefinedOID{category: list}
	}
	return map[string][]PredefinedOID{}
}

// GetPredefinedOIDCategories 返回所有分类名。
func GetPredefinedOIDCategories() []string {
	out := make([]string, 0, len(predefinedOIDs))
	for k := range predefinedOIDs {
		out = append(out, k)
	}
	return out
}
