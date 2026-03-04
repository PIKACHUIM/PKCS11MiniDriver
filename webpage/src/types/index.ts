// API 数据类型定义，与 client-card 后端模型对应

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
  password: string; // 卡片密码
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
