# OpenCert Manager — 部署与运维

> 文档版本：v2.0.0
> 最后更新：2026-04-17

---

## 一、部署架构

```mermaid
mindmap
  root((部署架构))
    编译环境
      Go ≥ 1.22
      Node.js ≥ 18 LTS
      GCC/MinGW C编译器
      CMake ≥ 3.20
    运行平台
      Windows 10+
        TPM 2.0 可选
      Linux Kernel 4.14+
        TPM 2.0 可选
      macOS 11 Big Sur+
        Secure Enclave Apple Silicon/T2
    组件部署
      OpenCert Manager 桌面端
        client-card Go二进制
        前端 embed.FS内嵌
        Electron可选
      OpenCert Platform 服务端
        server-card Go二进制
        PostgreSQL数据库
        Nginx反向代理
        Docker可选
      OpenCert Driver
        pkcs11-mock DLL/SO/DYLIB
    端口规划
      client-card :1026
      server-card :1027
      IPC Named Pipe/Unix Socket
```

---

## 二、环境要求

### 2.1 编译环境

| 组件 | 要求 |
|------|------|
| Go | ≥ 1.22 |
| Node.js | ≥ 18 LTS（前端构建） |
| GCC/MinGW | C 编译器（OpenCert Driver） |
| CMake | ≥ 3.20（OpenCert Driver 构建） |

### 2.2 运行环境

| 平台 | 最低版本 | TPM 支持 |
|------|---------|---------|
| Windows | 10 / Server 2016 | TPM 2.0（可选） |
| Linux | Kernel 4.14+ | TPM 2.0（可选） |
| macOS | 11 Big Sur+ | Secure Enclave（Apple Silicon / T2） |

---

## 三、构建步骤

### 3.1 OpenCert Manager（client-card + 前端）

```bash
# 1. 构建前端
cd front
npm install
npm run build
# 构建产物在 front/dist/

# 2. 将前端产物复制到 Go embed 目录
cp -r front/dist/* clients/ui/dist/

# 3. 构建 Go 二进制
cd clients
go build -o opencert-manager ./cmd/clients/

# Windows 交叉编译
GOOS=windows GOARCH=amd64 go build -o opencert-manager.exe ./cmd/clients/

# macOS 交叉编译（需要 CGO 支持 Secure Enclave）
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o opencert-manager-darwin ./cmd/clients/
```

### 3.2 OpenCert Platform（server-card）

```bash
cd servers
go build -o opencert-platform ./cmd/servers/

# Docker 构建
docker build -t opencert-platform:latest .
```

### 3.3 OpenCert Driver（pkcs11-mock）

```bash
cd drivers

# Windows（MSVC）
mkdir build && cd build
cmake .. -G "Visual Studio 17 2022" -A x64
cmake --build . --config Release
# 产物：build/Release/pkcs11-mock.dll

# Linux
mkdir build && cd build
cmake ..
make -j$(nproc)
# 产物：build/libpkcs11-mock.so

# macOS
mkdir build && cd build
cmake ..
make -j$(sysctl -n hw.ncpu)
# 产物：build/libpkcs11-mock.dylib
```

---

## 四、配置说明

### 4.1 OpenCert Manager 配置（config.yaml）

```yaml
# API 服务配置
api:
  host: "127.0.0.1"       # 绑定地址（建议保持 127.0.0.1）
  port: 1026               # 服务端口
  tls_enabled: false       # 是否启用 TLS
  tls_cert: ""             # TLS 证书路径
  tls_key: ""              # TLS 私钥路径

# IPC 配置
ipc:
  pipe_path: ""            # 留空使用默认路径
                           # Windows: \\.\pipe\opencert-pkcs11
                           # Unix: /tmp/opencert-pkcs11.sock

# 数据存储
storage:
  data_dir: ""             # 留空使用默认路径
                           # Windows: %APPDATA%/OpenCert/
                           # Linux: ~/.config/opencert/
                           # macOS: ~/Library/Application Support/OpenCert/
  db_name: "opencert.db"

# Cloud Slot 配置
cloud:
  allow_insecure: false    # 开发环境可设为 true 允许 HTTP
  cache_ttl: "24h"         # 离线缓存 TTL

# 安全配置
security:
  bcrypt_cost: 13
  argon2_time: 3
  argon2_memory: 65536     # 64MB
  argon2_threads: 4
  login_max_attempts: 5
  login_lockout_minutes: 15
  rate_limit_rpm: 100      # 每分钟每 IP 请求数
```

### 4.2 OpenCert Platform 配置（config.yaml）

```yaml
# API 服务配置
api:
  host: "0.0.0.0"
  port: 1027
  tls_enabled: true
  tls_cert: "/etc/opencert/tls.crt"
  tls_key: "/etc/opencert/tls.key"

# 数据库
database:
  driver: "postgres"
  dsn: "postgres://opencert:password@localhost:5432/opencert?sslmode=require"

# JWT 配置
jwt:
  algorithm: "ES256"       # HS256 / RS256 / ES256
  secret: ""               # HS256 密钥（≥256位）
  private_key: ""          # RS256/ES256 私钥路径
  access_ttl: "15m"
  refresh_ttl: "7d"

# 支付插件
payment:
  providers:
    - name: "stripe"
      enabled: true
      config:
        api_key: "sk_live_xxx"
        webhook_secret: "whsec_xxx"
```

---

## 五、TPM2 环境准备

### 5.1 Windows

```powershell
# 检查 TPM 是否可用
Get-Tpm

# 确认 TPM 2.0 版本
(Get-Tpm).ManufacturerVersion

# 无需额外安装，Windows 10+ 原生支持 TPM 2.0 API
```

### 5.2 Linux

```bash
# 安装 TPM2 工具和库
sudo apt install tpm2-tools tpm2-abrmd libtss2-dev       # Debian/Ubuntu
sudo dnf install tpm2-tools tpm2-abrmd tpm2-tss-devel     # Fedora/RHEL

# 启动 TPM2 资源管理器
sudo systemctl enable --now tpm2-abrmd

# 检查 TPM 设备
ls -la /dev/tpm*

# 确认用户有权限访问
sudo usermod -aG tss $USER
```

### 5.3 macOS

```bash
# Apple Silicon / T2 芯片自带 Secure Enclave
# 无需额外安装
# 注意：Secure Enclave 仅支持 EC P-256 密钥，不支持 RSA 硬件保护
```

### 5.4 TPM 降级策略

| 平台 | TPM 方案 | 降级方案 |
|------|---------|---------|
| Windows | TPM 2.0 | 降级为 Local Slot（纯软件加密） |
| Linux | TPM 2.0 | 降级为 Local Slot |
| macOS + CGO | Secure Enclave | 降级为 Keychain |
| macOS - CGO | Keychain | 降级为 Local Slot |

启动时自动检测 TPM 可用性，不可用时自动降级并在日志中输出警告。

---

## 六、生产部署建议

### 6.1 OpenCert Manager（桌面端）

- 以当前用户权限运行，不需要管理员/root 权限
- 数据目录权限设置为 0700（仅所有者可访问）
- 建议配合系统开机自启动（Windows 服务 / systemd / launchd）

### 6.2 OpenCert Platform（服务端）

- 使用 Docker 或 systemd 部署
- 强制启用 TLS（建议使用 Let's Encrypt 或内部 CA）
- PostgreSQL 启用 SSL 连接
- 配置反向代理（Nginx/Caddy）处理 TLS 终止
- 定期备份数据库
- 监控 `/api/health` 端点

### 6.3 Docker 部署示例

```yaml
# docker-compose.yml
version: '3.8'
services:
  opencert-platform:
    image: opencert-platform:latest
    ports:
      - "1027:1027"
    volumes:
      - ./config.yaml:/etc/opencert/config.yaml
      - ./tls:/etc/opencert/tls
    environment:
      - DATABASE_DSN=postgres://opencert:password@db:5432/opencert?sslmode=require
    depends_on:
      - db

  db:
    image: postgres:16
    environment:
      POSTGRES_USER: opencert
      POSTGRES_PASSWORD: password
      POSTGRES_DB: opencert
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

volumes:
  pgdata:
```

### 6.4 systemd 服务示例

```ini
# /etc/systemd/system/opencert-platform.service
[Unit]
Description=OpenCert Platform
After=network.target postgresql.service

[Service]
Type=simple
User=opencert
ExecStart=/usr/local/bin/opencert-platform
WorkingDirectory=/etc/opencert
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 七、安全加固清单

### 7.1 本地管理端

- [ ] API 绑定地址为 127.0.0.1
- [ ] Bearer Token 文件权限 0600
- [ ] SQLite 数据库文件权限 0600
- [ ] IPC 通道权限正确设置
- [ ] 审计日志已启用
- [ ] TPM 可用性已确认（如需高安全性模式）

### 7.2 云端平台

- [ ] TLS 证书有效且自动续期
- [ ] 数据库连接使用 SSL
- [ ] JWT 密钥长度 ≥ 256 位
- [ ] 支付插件配置参数已加密存储
- [ ] 审计日志已启用
- [ ] 定期验证审计日志完整性
- [ ] 配置反向代理（Nginx/Caddy）
- [ ] 防火墙规则已配置
- [ ] 定期备份数据库

---

## 八、监控与运维

### 8.1 健康检查

| 端点 | 说明 |
|------|------|
| `GET /api/health` | 服务健康状态 |

建议使用 Prometheus + Grafana 或类似工具监控。

### 8.2 日志管理

- 本地管理端：日志存储在 SQLite，支持 API 查询
- 云端平台：建议输出 JSON 格式日志，接入 ELK/Loki 等日志系统
- 审计日志：链式哈希保护，定期验证完整性

### 8.3 备份策略

| 组件 | 备份内容 | 建议频率 |
|------|---------|---------|
| 本地 SQLite | 数据库文件 | 用户自行备份 |
| 云端 PostgreSQL | 全量 + WAL 增量 | 每日全量 + 持续 WAL |
| CA 私钥 | 加密备份到安全存储 | 创建时备份 |
| 配置文件 | config.yaml | 变更时备份 |
