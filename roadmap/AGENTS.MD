# PKCS11Driver — 执行计划与方案文档

> 最后更新：2026-03-04
> 状态说明：✅ 已完成 | 🚧 进行中 | ⬜ 待开始

---

## 一、项目全景

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

### 组件职责

| 组件 | 语言 | 端口/路径 | 职责 |
|------|------|-----------|------|
| `pkcs11-mock` | C | — | PKCS#11 标准 DLL，转发调用到 client-card |
| `client-card` | Go | IPC + :1026 | 虚拟智能卡管理，三种 Slot 实现 |
| `server-card` | Go | :1027 | 云端卡片服务，管理云端证书和密钥 |
| 前端 | React/TS | :1026 (内嵌) | 管理界面，支持 Web 和 Electron 桌面 |

---

## 二、整体分阶段计划

| Phase | 内容 | 状态 |
|-------|------|------|
| Phase 1 | 项目骨架、数据模型、IPC 协议、存储层 | ✅ 已完成 |
| Phase 2 | Local Slot 实现（密钥管理 + 签名/解密） | ✅ 已完成 |
| Phase 3 | REST API 服务（用户/卡片/证书 CRUD） | ✅ 已完成 |
| Phase 4 | TPM2 Slot 实现（Windows TPM2 + macOS T2） | ✅ 已完成 |
| Phase 5 | Cloud Slot + server-card 协议设计与实现 | ✅ 已完成 |
| Phase 6 | 前端管理界面（React + Antd + Electron） | ✅ 已完成 |
| Phase 7 | pkcs11-mock C 代码完善与联调 | ✅ 已完成 |
| Phase 8 | 集成测试、安全审计、CI/CD | 🚧 进行中 |

---

## 三、已完成内容（Phase 1-3）

### Phase 1：项目骨架

**目录结构**
```
client-card/
├── cmd/client-card/main.go          # 启动入口
├── configs/config.go                # 配置加载（TOML/ENV）
├── internal/
│   ├── card/
│   │   ├── engine.go                # SlotProvider 接口定义
│   │   ├── manager.go               # Slot/Session 管理器
│   │   └── local/
│   │       ├── slot.go              # Local Slot 实现
│   │       └── keymgr.go            # 密钥管理（生成/导入）
│   ├── crypto/
│   │   ├── aes.go                   # AES-256-GCM 加解密
│   │   ├── bcrypt.go                # bcrypt 密码哈希
│   │   └── hmac.go                  # HMAC-SHA256
│   ├── ipc/
│   │   ├── protocol.go              # IPC 帧格式定义
│   │   ├── server.go                # IPC 服务器（命令分发）
│   │   ├── handler.go               # PKCS#11 命令处理器
│   │   ├── pipe_windows.go          # Windows Named Pipe
│   │   └── pipe_unix.go             # Unix Domain Socket
│   ├── storage/
│   │   ├── db.go                    # SQLite 初始化 + 迁移
│   │   ├── models.go                # 数据模型定义
│   │   ├── repo.go                  # 用户/卡片/日志 Repository
│   │   └── cert_repo.go             # 证书 Repository
│   └── api/
│       ├── server.go                # HTTP 服务器 + 路由注册
│       ├── response.go              # 统一响应格式
│       ├── user_card_handler.go     # 用户/卡片 CRUD Handler
│       ├── cert_handler.go          # 证书/密钥 Handler
│       └── helpers.go               # 工具函数
├── pkg/pkcs11types/types.go         # PKCS#11 常量和类型
└── test/
    ├── storage_test.go              # 存储层测试
    └── local_slot_test.go           # Local Slot 测试
```

**数据模型**（SQLite）

| 表 | 关键字段 |
|----|---------|
| `users` | uuid, user_type, display_name, email, password_hash, cloud_url, auth_key, enabled |
| `cards` | uuid, slot_type, card_name, user_uuid, card_keys(JSON), expires_at |
| `certificates` | uuid, slot_type, card_uuid, cert_type, key_type, cert_content, temp_key_salt, temp_key_enc, private_data |
| `logs` | uuid, log_type, slot_type, card_uuid, user_uuid, level, title, content |

**IPC 协议**（Named Pipe / Unix Socket + 二进制帧头 + JSON）
```
Frame: Magic(4B "PK11") + Cmd(4B) + Len(4B) + JSON Payload
       BigEndian 字节序，帧头共 12 字节

特殊命令：
- CmdPing      (0x0000) : 心跳，30秒空闲发送，3次无响应重连
- CmdHandshake (0x00FF) : 版本协商，C_Initialize 阶段交换协议版本号

Response JSON: { "rv": 0, "data": {...} }
```

支持的 PKCS#11 命令：
- `C_GetInfo` / `C_GetSlotList` / `C_GetSlotInfo` / `C_GetTokenInfo`
- `C_GetMechanismList` / `C_GetMechanismInfo`
- `C_OpenSession` / `C_CloseSession` / `C_CloseAllSessions` / `C_GetSessionInfo`
- `C_Login` / `C_Logout` / `C_InitPIN` / `C_SetPIN`
- `C_FindObjectsInit` / `C_FindObjects` / `C_FindObjectsFinal`
- `C_GetAttributeValue` / `C_CreateObject` / `C_DestroyObject`
- `C_SignInit` / `C_Sign`
- `C_DecryptInit` / `C_Decrypt`
- `C_EncryptInit` / `C_Encrypt`
- `C_GenerateKeyPair`

### Phase 2：Local Slot

**加密层次结构**
```
用户密码/卡片密码
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

**支持的密钥类型**：
- RSA: 1024 / 2048 / 4096 / 8192
- EC: P-256 / P-384 / P-521 / Brainpool P-256r1 / P-384r1 / P-512r1
- EdDSA: Ed25519 / X25519
- 国密: SM2

**支持的签名/摘要算法**：
- RSA: PKCS1v15 (SHA1/256/384/512), PSS (SHA256/384/512)
- ECDSA: SHA256 / SHA384 / SHA512 / 原始摘要
- EdDSA: Ed25519 纯签名
- 摘要: SHA-1 / SHA-256 / SHA-384 / SHA-512 / SHA3-256 / SHA3-384 / SHA3-512 / MD5 / SM3

**支持的加密算法**：
- 非对称: RSA-PKCS1v15 / RSA-OAEP
- 对称: AES-128/256 (GCM/CBC) / ChaCha20-Poly1305 / SM4

### Phase 3：REST API

**端口**：1026（可配置）

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

---

## 四、Phase 4：TPM2 Slot 实现

### 目标
- Windows：使用 TPM 2.0 芯片（通过 `go-tpm` 库）
- macOS：使用 T2/Apple Silicon Secure Enclave（通过 CryptoKit / Security.framework CGO）
- Linux：使用 TPM 2.0 芯片（通过 `go-tpm` 库）

### 方案设计

```
TPM2 Slot 加密层次：

用户密码
    │
    ▼ HMAC(password, salt) → AES-256-GCM
卡片主密钥 (32字节)
    │
    ▼ TPM2 RSA 公钥加密（TPM 内部密钥）
TPM 封装的主密钥
    │
    ▼ HMAC(masterKey, salt) → AES-256-GCM
临时密钥
    │
    ▼ AES-256-GCM
私钥 DER 数据
```

### 实现步骤

1. **抽象 TPM 接口** `internal/tpm/provider.go`
   ```go
   type TPMProvider interface {
       Seal(data []byte) (blob []byte, err error)
       Unseal(blob []byte) (data []byte, err error)
       Available() bool
   }
   ```

2. **Windows/Linux 实现** `internal/tpm/tpm2_linux_windows.go`
   - 使用 `github.com/google/go-tpm/tpm2` 库
   - 创建 Primary Key → 用于 Seal/Unseal 主密钥

3. **macOS 实现** `internal/tpm/tpm2_darwin.go`
   - 使用 CGO 调用 `Security.framework`
   - 通过 Secure Enclave 生成 EC P-256 密钥
   - 用 ECIES 加密主密钥

4. **TPM2 Slot** `internal/card/tpm2/slot.go`
   - 继承 Local Slot 大部分逻辑
   - 重写 `unlockMasterKey`：先 TPM Unseal，再 AES 解密

5. **数据库扩展**：`certificates.tpm_platform` 字段记录 TPM 封装的主密钥 blob

### 文件规划
```
internal/
├── tpm/
│   ├── provider.go              # TPM 接口定义
│   ├── tpm2_linux_windows.go    # go-tpm 实现（build tag: linux windows）
│   ├── tpm2_darwin.go           # macOS Secure Enclave（build tag: darwin）
│   └── mock.go                  # 测试用 Mock 实现
└── card/tpm2/
    ├── slot.go                  # TPM2 Slot 实现
    └── keymgr.go                # TPM2 密钥管理
```

---

## 五、Phase 5：Cloud Slot + server-card

### server-card 服务设计

**端口**：1027（可配置）

**认证方式**：JWT（登录后返回，内含有效期）

**API 设计**
```
POST   /api/auth/login           # 登录，返回 JWT
POST   /api/auth/refresh         # 刷新 Token
DELETE /api/auth/logout          # 登出

GET    /api/cards                # 列出用户的云端卡片
POST   /api/cards                # 创建云端卡片
GET    /api/cards/{uuid}         # 卡片详情
DELETE /api/cards/{uuid}         # 删除卡片

GET    /api/cards/{uuid}/certs   # 证书列表（不含私钥）
POST   /api/cards/{uuid}/certs   # 上传证书（私钥加密存储）
DELETE /api/cards/{uuid}/certs/{uuid}

POST   /api/cards/{uuid}/sign    # 云端签名（私钥不离开服务器）
POST   /api/cards/{uuid}/decrypt # 云端解密
```

### Cloud Slot 设计（client-card 侧）

```
Cloud Slot 工作流：

1. Login：POST /api/auth/login → 获取 JWT，存入 auth_key 字段
2. FindObjects：GET /api/cards/{uuid}/certs → 缓存到本地 SQLite
3. Sign：POST /api/cards/{uuid}/sign → 发送数据，服务端签名返回
4. Decrypt：POST /api/cards/{uuid}/decrypt → 服务端解密返回
```

### 文件规划
```
server-card/
├── cmd/server-card/main.go
├── configs/config.go
├── internal/
│   ├── api/                     # REST API Handler
│   ├── auth/                    # JWT 认证
│   ├── card/                    # 云端卡片业务逻辑
│   └── storage/                 # PostgreSQL 存储层
└── go.mod

client-card/internal/card/cloud/
├── slot.go                      # Cloud Slot 实现
└── client.go                    # server-card HTTP 客户端
```

---

## 六、Phase 6：前端管理界面

### 技术栈
- **框架**：React 18 + TypeScript
- **UI 库**：Ant Design 5.x
- **状态管理**：Zustand（轻量）
- **路由**：React Router v6
- **国际化**：i18next（中/英）
- **主题**：Antd 暗黑/透明模式切换
- **桌面**：Electron（托盘图标 + 系统通知）
- **构建**：Vite

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

### 与 client-card 集成
- 前端静态文件内嵌到 Go 二进制（`embed.FS`）
- 访问 `http://localhost:1026` 即可打开管理界面
- Electron 模式：加载 `http://localhost:1026`，添加系统托盘

---

## 七、Phase 7：pkcs11-mock C 代码完善

### 当前状态
- `drivers/src/` 已有基础 C 代码框架
- 需要完善 IPC 客户端（Named Pipe / Unix Socket）
- 需要实现所有 PKCS#11 标准函数的转发

### 关键函数映射

| PKCS#11 函数 | IPC 命令 | 状态 |
|-------------|---------|------|
| `C_Initialize` | Handshake (版本协商) | ✅ |
| `C_GetInfo` | `CmdGetInfo` | ✅ |
| `C_GetSlotList` | `CmdGetSlotList` | ✅ |
| `C_GetSlotInfo` | `CmdGetSlotInfo` | ✅ |
| `C_GetTokenInfo` | `CmdGetTokenInfo` | ✅ |
| `C_GetMechanismList` | `CmdGetMechanismList` | ✅ |
| `C_GetMechanismInfo` | `CmdGetMechanismInfo` | ✅ |
| `C_OpenSession` | `CmdOpenSession` | ✅ |
| `C_CloseSession` | `CmdCloseSession` | ✅ |
| `C_CloseAllSessions` | `CmdCloseAllSessions` | ✅ |
| `C_GetSessionInfo` | `CmdGetSessionInfo` | ✅ |
| `C_Login` | `CmdLogin` | ✅ |
| `C_Logout` | `CmdLogout` | ✅ |
| `C_InitPIN` | `CmdInitPIN` | ✅ |
| `C_SetPIN` | `CmdSetPIN` | ✅ |
| `C_FindObjectsInit` | `CmdFindObjectsInit` | ✅ |
| `C_FindObjects` | `CmdFindObjects` | ✅ |
| `C_FindObjectsFinal` | `CmdFindObjectsFinal` | ✅ |
| `C_GetAttributeValue` | `CmdGetAttributeValue` | ✅ |
| `C_CreateObject` | `CmdCreateObject` | ✅ |
| `C_DestroyObject` | `CmdDestroyObject` | ✅ |
| `C_SignInit` | `CmdSignInit` | ✅ |
| `C_Sign` | `CmdSign` | ✅ |
| `C_DecryptInit` | `CmdDecryptInit` | ✅ |
| `C_Decrypt` | `CmdDecrypt` | ✅ |
| `C_EncryptInit` | `CmdEncryptInit` | ✅ |
| `C_Encrypt` | `CmdEncrypt` | ✅ |
| `C_GenerateKeyPair` | `CmdGenerateKeyPair` | ✅ |

---

## 八、Phase 8：集成测试与 CI/CD

### 测试策略

```
单元测试（已有）
├── test/storage_test.go         ✅ 用户/卡片/日志 CRUD
└── test/local_slot_test.go      ✅ 创建卡片/登录/密钥生成/签名

集成测试（已补充）
├── test/api_test.go             ✅ REST API 端到端测试（用户/卡片/日志/Slot/参数校验）
├── test/ipc_test.go             ✅ IPC 协议端到端测试（帧协议/会话/登录/FindObjects）
└── test/cloud_slot_test.go      ✅ Cloud Slot 集成测试

E2E 测试
└── test/pkcs11_e2e_test.go      ⬜ pkcs11-mock + client-card 联调
```

### CI/CD 流水线（GitHub Actions）
```
.github/workflows/
├── ci.yml              ✅ 主 CI（测试/构建/安全扫描，三平台矩阵）
└── release-beta.yml    ✅ Beta 发布（tag v*-beta* 触发，自动创建 pre-release）
```

---

## 九、关键设计决策记录

| 决策 | 选择 | 原因 |
|------|------|------|
| IPC 协议格式 | JSON | 调试方便，性能足够 |
| IPC 传输 | Named Pipe (Win) / Unix Socket (Unix) | 方案A，无需网络端口 |
| 数据库 | SQLite (client) / PostgreSQL (server) | 本地零依赖，云端高并发 |
| 加密方案 | AES-256-GCM + HMAC-SHA256 | 标准、安全、Go 原生支持 |
| 密码存储 | bcrypt (cost=12) | 抗暴力破解 |
| API 框架 | Go 1.22 标准库 net/http | 零依赖，ServeMux 已足够 |
| 前端框架 | React + Antd | 成熟生态，组件丰富 |
| 桌面方案 | Electron | 跨平台，复用 Web 代码 |
| client 端口 | 1026 | 可配置 |
| server 端口 | 1027 | 可配置 |

---

## 十、当前待解决问题

1. **TPM2 macOS 方案**：Secure Enclave 的 CGO 绑定需要验证，备选方案是使用 `keychain` 存储主密钥
2. **前端内嵌**：`embed.FS` 与 Vite 构建产物的集成方式需要确认
3. **pkcs11-mock 连接重试**：C 代码需要处理 client-card 未启动时的重连逻辑
4. **Cloud Slot 离线缓存**：证书缓存策略（TTL、强制刷新）需要设计
5. **多用户权限**：同一张卡片多个用户共享时的 CardKeys 管理逻辑

---

## 十一、下一步行动（Phase 4）

### 立即执行

1. 添加 `go-tpm` 依赖
   ```bash
   go get github.com/google/go-tpm/tpm2
   go get github.com/google/go-tpm-tools/client
   ```

2. 创建 TPM 抽象接口 `internal/tpm/provider.go`

3. 实现 Windows/Linux TPM2 `internal/tpm/tpm2_linux_windows.go`
   - `C_Initialize`：打开 TPM 设备，创建 Primary Key
   - `Seal(data)`：用 Primary Key 封装数据
   - `Unseal(blob)`：解封数据

4. 实现 macOS Secure Enclave `internal/tpm/tpm2_darwin.go`
   - 通过 CGO 调用 `Security.framework`
   - 生成 Secure Enclave EC P-256 密钥
   - 用 ECIES 加密/解密主密钥

5. 实现 TPM2 Slot `internal/card/tpm2/slot.go`
   - 复用 Local Slot 的签名/解密逻辑
   - 重写 `unlockMasterKey`：TPM Unseal → AES 解密

6. 更新 `main.go` 加载 TPM2 Slot

7. 补充测试 `test/tpm2_slot_test.go`（使用 Mock TPM）