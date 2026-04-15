# PKCS11Driver — 本次对话思路总结

> 创建时间：2026-04-14
> 本文档汇总本次对话中产生的所有设计思路、方案决策和实现规划

---

## 一、项目背景与目标

### 项目定位

构建一套完整的虚拟智能卡 PKCS#11 驱动系统，架构如下：

```
pkcs11-mock (C DLL)
      │  Named Pipe / Unix Socket (IPC)
      ▼
 client-card (Go 服务)          ←→  前端 (React + Antd)
      │  REST API :1026
      │
      ├── Local Slot   (本地 SQLite + AES256)
      ├── TPM2 Slot    (TPM2/T2 芯片 + AES256)
      └── Cloud Slot   (server-card REST API :1027)
                              │
                         server-card (Go 服务)
                              │
                         云端数据库 (PostgreSQL)
```

### 核心设计原则

1. **接口驱动**：所有 Slot 类型实现统一的 `SlotProvider` 接口，后期可无缝扩展新卡片类型
2. **最小权限**：私钥永不明文传输，本地加密存储，云端签名不离开服务器
3. **零依赖本地**：client-card 使用 SQLite，无需额外数据库服务
4. **标准兼容**：严格遵循 PKCS#11 v2.40 标准，确保与现有 PKI 生态兼容

---

## 二、核心数据模型设计思路

### 加密层次结构（本地/TPM2）

```
用户密码
    │
    ▼ HMAC(password, salt32) → AES-256-GCM
卡片主密钥 (32字节随机)
    │
    ▼ HMAC(masterKey, salt32) → AES-256-GCM
临时密钥 (32字节随机，每个证书独立)
    │
    ▼ AES-256-GCM
私钥 DER 数据
```

**关键设计决策**：
- 卡片密码以 JSON 列表形式存储多种加密信息，支持多用户权限共享同一张卡
- `HMAC(用户密码, salt)` 加密主密钥 → 多个用户可各自解锁
- `HMAC(卡片密码, salt)` 加密主密钥 → 持有卡密码可绕过用户权限访问

### TPM2 额外保护层

```
用户密码
    │
    ▼ HMAC(password, salt) → AES-256-GCM
卡片主密钥 (32字节)
    │
    ▼ TPM2 RSA 公钥加密（TPM 内部密钥，不可导出）
TPM 封装的主密钥 blob
    │
    ▼ HMAC(masterKey, salt) → AES-256-GCM
临时密钥
    │
    ▼ AES-256-GCM
私钥 DER 数据
```

### 数据库表设计

| 表 | 关键字段 | 说明 |
|----|---------|------|
| `users` | uuid, user_type, display_name, email, password_hash, cloud_url, auth_key, enabled | 支持本地和云端用户 |
| `cards` | uuid, slot_type, card_name, user_uuid, card_keys(JSON), expires_at | card_keys 存储多用户加密主密钥 |
| `certificates` | uuid, slot_type, card_uuid, cert_type, key_type, cert_content, temp_key_salt, temp_key_enc, private_data | 证书和私钥分离存储 |
| `logs` | uuid, log_type, slot_type, card_uuid, user_uuid, level, title, content | 完整审计日志 |

---

## 三、IPC 协议设计思路

### 传输层选择

| 平台 | 方案 | 原因 |
|------|------|------|
| Windows | Named Pipe | 系统原生，无需网络端口，安全隔离 |
| Linux/macOS | Unix Domain Socket | 文件系统权限控制，高性能 |

### 帧格式

```
Frame: [4字节长度(LE)] + [JSON Payload]

Request:  { "cmd": "C_Sign", "seq": 1, "payload": {...} }
Response: { "seq": 1, "rv": 0, "payload": {...} }
```

**选择 JSON 而非二进制协议的原因**：调试方便，性能对于本地 IPC 足够，无需额外序列化库

---

## 四、PKCS#11 功能映射思路

### 层级关系

```
Library → Slot → Token → Session → Object
```

### 会话状态机

```
C_OpenSession(只读)  →  RO_PUBLIC
C_OpenSession(读写)  →  RW_PUBLIC

RO_PUBLIC + C_Login(USER)  →  RO_USER
RW_PUBLIC + C_Login(USER)  →  RW_USER
RW_PUBLIC + C_Login(SO)    →  RW_SO

已登录状态再次 C_Login  →  ALREADY_LOGGED_IN
```

### 操作状态机（双操作叠加）

同一时刻支持两种操作并行（如 SIGN_ENCRYPT、DECRYPT_DIGEST），通过全局状态变量管理，任一操作完成后退回单操作状态。

---

## 五、三种 Slot 实现对比

| 特性 | Local Slot | TPM2 Slot | Cloud Slot |
|------|-----------|-----------|-----------|
| 私钥存储位置 | 本地 SQLite（加密） | 本地 SQLite（TPM 封装加密） | 云端服务器 |
| 签名操作 | 本地执行 | 本地执行（TPM 参与解密） | 云端执行，私钥不离开服务器 |
| 离线可用 | ✅ 完全离线 | ✅ 完全离线 | ❌ 需要网络 |
| 硬件保护 | ❌ 纯软件 | ✅ TPM 芯片保护 | ✅ 服务器端保护 |
| 跨设备使用 | ❌ 仅本机 | ❌ 仅本机 | ✅ 任意设备 |
| 实现复杂度 | 低 | 中 | 高 |

---

## 六、REST API 设计思路

### client-card API（端口 1026）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET/POST | `/api/users` | 用户列表/创建 |
| GET/PUT/DELETE | `/api/users/{uuid}` | 用户详情/更新/删除 |
| GET/POST | `/api/cards` | 卡片列表/创建 |
| GET/PUT/DELETE | `/api/cards/{uuid}` | 卡片详情/更新/删除 |
| GET/POST | `/api/cards/{uuid}/certs` | 证书列表/导入 |
| GET/DELETE | `/api/cards/{uuid}/certs/{uuid}` | 证书详情/删除 |
| POST | `/api/cards/{uuid}/keygen` | 生成密钥对 |
| GET | `/api/logs` | 日志查询（分页） |
| GET | `/api/slots` | Slot 状态列表 |

### server-card API（端口 1027）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/api/auth/login` | 登录，返回 JWT |
| POST | `/api/auth/refresh` | 刷新 Token |
| DELETE | `/api/auth/logout` | 登出 |
| GET/POST | `/api/cards` | 云端卡片列表/创建 |
| GET/DELETE | `/api/cards/{uuid}` | 卡片详情/删除 |
| GET/POST | `/api/cards/{uuid}/certs` | 证书列表/上传 |
| DELETE | `/api/cards/{uuid}/certs/{uuid}` | 删除证书 |
| POST | `/api/cards/{uuid}/sign` | 云端签名 |
| POST | `/api/cards/{uuid}/decrypt` | 云端解密 |

**技术选型**：Go 1.22 标准库 `net/http` + 新 `ServeMux`，零依赖，路由已足够

---

## 七、前端设计思路

### 技术栈选择

| 技术 | 选择 | 原因 |
|------|------|------|
| 框架 | React 18 + TypeScript | 成熟生态，类型安全 |
| UI 库 | Ant Design 5.x | 组件丰富，企业级 |
| 状态管理 | Zustand | 轻量，无样板代码 |
| 路由 | React Router v6 | 标准方案 |
| 国际化 | i18next | 支持中/英切换 |
| 桌面 | Electron | 跨平台，复用 Web 代码 |
| 构建 | Vite | 快速热更新 |

### 页面规划

```
├── 登录页          /login
├── 仪表盘          /dashboard        (Slot 状态总览)
├── 用户管理        /users            (CRUD)
├── 卡片管理        /cards            (CRUD + 密码解锁)
│   └── 证书管理    /cards/:id/certs  (CRUD + 密钥生成)
├── 日志查看        /logs             (分页 + 筛选)
└── 系统设置        /settings         (端口/主题/语言)
```

### 内嵌方案

前端静态文件通过 Go `embed.FS` 内嵌到二进制，访问 `http://localhost:1026` 即可打开管理界面，Electron 模式加载同一地址并添加系统托盘。

---

## 八、TPM2 实现思路

### 抽象接口设计

```go
type TPMProvider interface {
    Seal(data []byte) (blob []byte, err error)
    Unseal(blob []byte) (data []byte, err error)
    Available() bool
}
```

### 平台实现策略

| 平台 | 实现方案 | Build Tag |
|------|---------|-----------|
| Windows | `go-tpm/tpm2` 库，创建 Primary Key | `windows` |
| Linux | `go-tpm/tpm2` 库，打开 `/dev/tpm0` | `linux` |
| macOS | CGO 调用 `Security.framework`，Secure Enclave EC P-256 | `darwin` |
| 测试 | Mock 实现，内存存储 | — |

### 待解决问题

- macOS Secure Enclave 的 CGO 绑定需要验证，备选方案是使用 `keychain` 存储主密钥

---

## 九、测试策略

```
单元测试
├── test/storage_test.go         用户/卡片/日志 CRUD
└── test/local_slot_test.go      创建卡片/登录/密钥生成/签名

集成测试
├── test/api_test.go             REST API 端到端测试
├── test/ipc_test.go             IPC 协议端到端测试
└── test/cloud_slot_test.go      Cloud Slot 集成测试

E2E 测试
└── test/pkcs11_e2e_test.go      pkcs11-mock + client-card 联调（待实现）
```

---

## 十、CI/CD 流水线

```
.github/workflows/
├── ci.yml              主 CI（测试/构建/安全扫描，三平台矩阵：ubuntu/windows/macos）
└── release-beta.yml    Beta 发布（tag v*-beta* 触发，自动创建 pre-release）
```

---

## 十一、关键设计决策汇总

| 决策点 | 选择 | 原因 |
|--------|------|------|
| IPC 协议格式 | JSON | 调试方便，性能足够 |
| IPC 传输层 | Named Pipe (Win) / Unix Socket (Unix) | 无需网络端口，系统级安全隔离 |
| 本地数据库 | SQLite | 零依赖，单文件，Go 原生支持 |
| 云端数据库 | PostgreSQL | 高并发，成熟生态 |
| 加密方案 | AES-256-GCM + HMAC-SHA256 | 标准、安全、Go 原生支持 |
| 密码存储 | bcrypt (cost=12) | 抗暴力破解 |
| API 框架 | Go 1.22 标准库 net/http | 零依赖，新 ServeMux 已足够 |
| 前端框架 | React + Antd | 成熟生态，组件丰富 |
| 桌面方案 | Electron | 跨平台，复用 Web 代码 |
| 密钥类型支持 | rsa2048/rsa4096/ec256/ec384/ec521 | 覆盖主流场景 |
| 签名算法 | RSA PKCS1v15/PSS + ECDSA | 标准兼容 |

---

## 十二、当前进度与待办

### 已完成（Phase 1-7）

- ✅ Phase 1：项目骨架、数据模型、IPC 协议、存储层
- ✅ Phase 2：Local Slot 实现（密钥管理 + 签名/解密）
- ✅ Phase 3：REST API 服务（用户/卡片/证书 CRUD）
- ✅ Phase 4：TPM2 Slot 实现（Windows TPM2 + macOS T2）
- ✅ Phase 5：Cloud Slot + server-card 协议设计与实现
- ✅ Phase 6：前端管理界面（React + Antd + Electron）
- ✅ Phase 7：pkcs11-mock C 代码完善与联调

### 进行中（Phase 8）

- 🚧 集成测试、安全审计、CI/CD

### 待解决问题

1. **TPM2 macOS 方案**：Secure Enclave CGO 绑定验证，备选 `keychain`
2. **前端内嵌**：`embed.FS` 与 Vite 构建产物集成方式确认
3. **pkcs11-mock 连接重试**：C 代码处理 client-card 未启动时的重连逻辑
4. **Cloud Slot 离线缓存**：证书缓存 TTL 和强制刷新策略设计
5. **多用户权限**：同一张卡片多用户共享时的 CardKeys 管理逻辑
6. **E2E 测试**：pkcs11-mock + client-card 完整联调测试

---

## 十三、相关文档索引

| 文档 | 路径 | 内容 |
|------|------|------|
| 初始需求设计 | `roadmap/DESIGN.MD` | 项目需求和初始方案讨论 |
| 完整执行计划 | `roadmap/AGENTS.MD` | Phase 1-8 详细实现计划 |
| PKCS#11 功能映射 | `roadmap/DRIVER.md` | PKCS#11 标准函数映射关系网 |
| 本次对话总结 | `roadmap/SESSION.MD` | 本文档 |
