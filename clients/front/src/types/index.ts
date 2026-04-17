// API 数据类型定义，与 client-card(:1026) 后端模型对应

export interface User {
  uuid: string;
  user_type: string;
  display_name: string;
  email: string;
  cloud_url?: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Card {
  uuid: string;
  slot_type: 'local' | 'tpm2' | 'cloud';
  card_name: string;
  user_uuid: string;
  remark?: string;
  cloud_url?: string;
  cloud_card_uuid?: string;
  created_at: string;
  expires_at?: string;
}

export interface Certificate {
  uuid: string;
  card_uuid: string;
  slot_type: string;
  cert_type: string;
  key_type: string;
  cert_content?: string; // base64
  remark?: string;
  created_at: string;
}

export interface Log {
  uuid: string;
  log_type: string;
  slot_type: string;
  card_uuid: string;
  user_uuid: string;
  level: string;
  title: string;
  content: string;
  created_at: string;
}

export interface SlotInfo {
  slot_id: number;
  description: string;
  token_present: boolean;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// 密钥生成请求
export interface KeyGenRequest {
  key_type: 'rsa2048' | 'rsa4096' | 'ec256' | 'ec384' | 'ec521';
  remark?: string;
  password: string;
}

// 创建卡片请求
export interface CreateCardRequest {
  slot_type: 'local' | 'tpm2' | 'cloud';
  card_name: string;
  user_uuid: string;
  password: string;
  remark?: string;
  cloud_url?: string;
  cloud_card_uuid?: string;
}

// 创建用户请求
export interface CreateUserRequest {
  user_type: string;
  display_name: string;
  email: string;
  password: string;
  cloud_url?: string;
}

// ---- TOTP 类型（本地卡片内） ----

export interface TOTPEntry {
  uuid: string;
  card_uuid: string;
  issuer: string;
  account: string;
  algorithm: 'SHA1' | 'SHA256' | 'SHA512';
  digits: 6 | 8;
  period: number;
  created_at: string;
}

export interface CreateTOTPRequest {
  card_uuid?: string;
  issuer: string;
  account: string;
  secret: string;       // Base32 编码
  uri?: string;          // otpauth:// URI（可选，优先解析）
  algorithm?: string;
  digits?: number;
  period?: number;
}

export interface TOTPCodeResponse {
  code: string;
  remaining: number;
}

// ---- 本地 PKI 类型 ----

/** 密钥存储位置 */
export type KeyStorage = 'database' | 'smartcard' | 'imported';

/** CSR 记录（数据库存储） */
export interface CSRRecord {
  uuid: string;
  common_name: string;
  organization?: string;
  org_unit?: string;
  country?: string;
  state?: string;
  locality?: string;
  email?: string;
  key_type: string;
  key_storage: KeyStorage;
  card_uuid?: string;
  san_dns?: string;
  san_ip?: string;
  san_email?: string;
  san_uri?: string;
  key_usage?: string;
  ext_key_usage?: string;
  csr_pem: string;
  has_private_key: boolean;
  remark?: string;
  created_at: string;
}

/** 创建 CSR 请求 */
export interface CreateCSRRequest {
  common_name: string;
  organization?: string;
  org_unit?: string;
  country?: string;
  state?: string;
  locality?: string;
  email?: string;
  key_type: string;
  key_storage: KeyStorage;
  card_uuid?: string;
  san_dns?: string;
  san_ip?: string;
  san_email?: string;
  san_uri?: string;
  key_usage?: string[];
  ext_key_usage?: string[];
  remark?: string;
}

/** 本地 CA 记录 */
export interface LocalCA {
  uuid: string;
  name: string;
  common_name: string;
  organization?: string;
  country?: string;
  key_type: string;
  cert_pem?: string;
  chain_pem?: string;
  has_priv_key: boolean;
  card_uuid?: string;
  not_before: string;
  not_after: string;
  issued_count: number;
  revoked: boolean;
  created_at: string;
}

/** 创建 CA 请求 */
export interface CreateCARequest {
  name: string;
  common_name: string;
  organization?: string;
  country?: string;
  key_type: string;
  validity_years: number;
  card_uuid?: string;
}

/** 导入 CA 请求 */
export interface ImportCARequest {
  name: string;
  cert_pem: string;
  key_pem?: string;
  chain_pem?: string;
  card_uuid?: string;
}

/** PKI 证书记录 */
export interface PKICert {
  uuid: string;
  common_name: string;
  serial_number?: string;
  ca_uuid?: string;
  ca_name?: string;
  csr_uuid?: string;
  key_type: string;
  key_storage: KeyStorage;
  card_uuid?: string;
  cert_pem?: string;
  has_private_key: boolean;
  not_before: string;
  not_after: string;
  key_usage?: string;
  ext_key_usage?: string;
  san_dns?: string;
  san_ip?: string;
  san_email?: string;
  revoked: boolean;
  remark?: string;
  created_at: string;
}

/** 签发证书请求 */
export interface IssueCertRequest {
  csr_uuid: string;
  ca_uuid: string;
  validity_days: number;
  remark?: string;
}

/** 导入证书模式 */
export type ImportCertMode = 'cert_only' | 'cert_key' | 'pkcs12' | 'key_only';

/** 导入证书请求 */
export interface ImportCertRequest {
  mode: ImportCertMode;
  cert_pem?: string;
  key_pem?: string;
  pkcs12_b64?: string;
  pkcs12_password?: string;
  card_uuid?: string;
  remark?: string;
}

/** 导出证书格式 */
export type ExportCertFormat = 'pem' | 'der' | 'pkcs12' | 'key_pem';

/** 自签名证书请求 */
export interface SelfSignRequest {
  common_name: string;
  organization?: string;
  org_unit?: string;
  country?: string;
  locality?: string;
  key_type: string;
  validity_days: number;
  card_uuid: string;
  san_dns?: string;
  san_ip?: string;
  san_email?: string;
  key_usage?: string[];
  ext_key_usage?: string[];
  export_also?: boolean;
}

// 旧版兼容（保留）
export interface CSRRequest {
  common_name: string;
  key_type: string;
  card_uuid: string;
  san_dns?: string;
}

export interface CSRResponse {
  csr_pem: string;
}

export interface CreateLocalCARequest {
  name: string;
  key_type: string;
  validity_years: number;
  card_uuid: string;
}

// ---- 认证类型 ----

export interface LoginRequest {
  username: string;
  password: string;
}

export interface AuthToken {
  token: string;
  user_uuid: string;
  username: string;
  role: 'admin' | 'user' | 'readonly';
}
