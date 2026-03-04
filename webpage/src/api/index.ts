import axios from 'axios';
import type {
  User, Card, Certificate, Log, SlotInfo,
  CreateUserRequest, CreateCardRequest, KeyGenRequest,
  PageResult,
} from '../types';

// API 基础地址，开发时代理到 client-card :1026
const BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:1026';

const http = axios.create({
  baseURL: BASE_URL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
});

// 响应拦截器：统一错误处理
http.interceptors.response.use(
  (res) => res,
  (err) => {
    const msg = err.response?.data?.message || err.message || '请求失败';
    return Promise.reject(new Error(msg));
  }
);

// ---- 健康检查 ----
export const getHealth = () =>
  http.get<{ status: string; version: string }>('/api/health').then((r) => r.data);

// ---- Slot 状态 ----
export const getSlots = () =>
  http.get<SlotInfo[]>('/api/slots').then((r) => r.data);

// ---- 用户管理 ----
export const getUsers = (params?: { page?: number; page_size?: number }) =>
  http.get<PageResult<User>>('/api/users', { params }).then((r) => r.data);

export const getUser = (uuid: string) =>
  http.get<User>(`/api/users/${uuid}`).then((r) => r.data);

export const createUser = (data: CreateUserRequest) =>
  http.post<User>('/api/users', data).then((r) => r.data);

export const updateUser = (uuid: string, data: Partial<CreateUserRequest & { enabled: boolean }>) =>
  http.put<User>(`/api/users/${uuid}`, data).then((r) => r.data);

export const deleteUser = (uuid: string) =>
  http.delete(`/api/users/${uuid}`).then((r) => r.data);

// ---- 卡片管理 ----
export const getCards = (params?: { user_uuid?: string; page?: number; page_size?: number }) =>
  http.get<PageResult<Card>>('/api/cards', { params }).then((r) => r.data);

export const getCard = (uuid: string) =>
  http.get<Card>(`/api/cards/${uuid}`).then((r) => r.data);

export const createCard = (data: CreateCardRequest) =>
  http.post<Card>('/api/cards', data).then((r) => r.data);

export const updateCard = (uuid: string, data: Partial<Card>) =>
  http.put<Card>(`/api/cards/${uuid}`, data).then((r) => r.data);

export const deleteCard = (uuid: string) =>
  http.delete(`/api/cards/${uuid}`).then((r) => r.data);

// ---- 证书管理 ----
export const getCerts = (cardUUID: string) =>
  http.get<Certificate[]>(`/api/cards/${cardUUID}/certs`).then((r) => r.data);

export const getCert = (cardUUID: string, certUUID: string) =>
  http.get<Certificate>(`/api/cards/${cardUUID}/certs/${certUUID}`).then((r) => r.data);

export const deleteCert = (cardUUID: string, certUUID: string) =>
  http.delete(`/api/cards/${cardUUID}/certs/${certUUID}`).then((r) => r.data);

export const generateKey = (cardUUID: string, data: KeyGenRequest) =>
  http.post<Certificate>(`/api/cards/${cardUUID}/keygen`, data).then((r) => r.data);

// ---- 日志查询 ----
export const getLogs = (params?: {
  card_uuid?: string;
  user_uuid?: string;
  level?: string;
  page?: number;
  page_size?: number;
}) => http.get<PageResult<Log>>('/api/logs', { params }).then((r) => r.data);
