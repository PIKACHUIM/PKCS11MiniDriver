import DocPage from '../components/DocPage'

export default function QuickStartPage() {
  return (
    <DocPage
      title="快速开始"
      subtitle="从零开始部署 OpenCert Manager，包括云端平台、本地客户端和 PKCS#11 驱动"
      badge="Quick Start"
    >
      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>前置要求</strong>
          <ul style={{ margin: '0.5rem 0 0', paddingLeft: '1.2rem' }}>
            <li>Go ≥ 1.22</li>
            <li>Node.js ≥ 18 LTS（前端构建）</li>
            <li>GCC / MinGW（C 驱动编译）</li>
            <li>CMake ≥ 3.20（驱动构建）</li>
            <li>PostgreSQL 14+（云端平台）</li>
          </ul>
        </div>
      </div>

      {/* ── 一、云端平台 ── */}
      <h2>☁️ 一、部署云端平台（server-card）</h2>
      <div className="steps">
        <div className="step">
          <div className="step-num">1</div>
          <div className="step-body">
            <div className="step-title">克隆仓库</div>
            <pre>{`git clone https://github.com/PIKACHUIM/PKCS11MiniDriver.git
cd PKCS11MiniDriver`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">2</div>
          <div className="step-body">
            <div className="step-title">配置服务</div>
            <pre>{`cd servers
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml，配置数据库连接、JWT 密钥等
# 关键配置项：
#   api.port: 1027
#   database.dsn: postgres://opencert:password@localhost:5432/opencert
#   jwt.algorithm: ES256`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">编译并运行</div>
            <pre>{`go mod download
go build -o opencert-platform ./cmd/servers/
./opencert-platform --config configs/config.yaml`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">4</div>
          <div className="step-body">
            <div className="step-title">验证服务</div>
            <pre>{`curl http://localhost:1027/api/health
# 返回: {"data":{"status":"ok","version":"2.0.0"}}`}</pre>
          </div>
        </div>
      </div>

      {/* ── 二、本地客户端 ── */}
      <h2>💻 二、运行本地客户端（client-card）</h2>
      <div className="steps">
        <div className="step">
          <div className="step-num">1</div>
          <div className="step-body">
            <div className="step-title">构建前端</div>
            <pre>{`cd front
npm install
npm run build
# 构建产物在 front/dist/

# 将产物复制到 Go embed 目录
cp -r dist/* ../clients/ui/dist/`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">2</div>
          <div className="step-body">
            <div className="step-title">编译 Go 后端</div>
            <pre>{`cd clients
go mod download
go build -o opencert-manager ./cmd/clients/

# Windows 编译
GOOS=windows GOARCH=amd64 go build -o opencert-manager.exe ./cmd/clients/

# macOS（需要 CGO 支持 Secure Enclave）
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o opencert-manager-darwin ./cmd/clients/`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">启动客户端</div>
            <pre>{`# Web 模式（浏览器访问 http://localhost:1026）
./opencert-manager

# Electron 桌面模式
cd front && npm run electron`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">4</div>
          <div className="step-body">
            <div className="step-title">初始化配置</div>
            <div className="step-desc">
              打开 <code>http://localhost:1026</code>，创建第一个用户，
              可选配置云端服务地址（<code>https://your-server:1027</code>）。
            </div>
          </div>
        </div>
      </div>

      {/* ── 三、PKCS#11 驱动 ── */}
      <h2>🔌 三、安装 PKCS#11 驱动（pkcs11-mock）</h2>
      <div className="steps">
        <div className="step">
          <div className="step-num">1</div>
          <div className="step-body">
            <div className="step-title">编译驱动</div>
            <pre>{`cd drivers

# Windows（MSVC）
mkdir build && cd build
cmake .. -G "Visual Studio 17 2022" -A x64
cmake --build . --config Release
# 产物：build/Release/pkcs11-mock.dll

# Linux
mkdir build && cd build
cmake .. && make -j$(nproc)
# 产物：build/libpkcs11-mock.so

# macOS
mkdir build && cd build
cmake .. && make -j$(sysctl -n hw.ncpu)
# 产物：build/libpkcs11-mock.dylib`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">2</div>
          <div className="step-body">
            <div className="step-title">注册驱动</div>
            <pre>{`# Windows - Firefox 手动加载
# 在 Firefox 证书管理器中加载 pkcs11-mock.dll

# Linux - 复制到标准路径
sudo cp build/libpkcs11-mock.so /usr/lib/pkcs11/

# Linux - Firefox 注册
modutil -dbdir ~/.mozilla/firefox/<profile> \
        -add "OpenCert PKCS11" \
        -libfile /usr/lib/pkcs11/libpkcs11-mock.so

# SSH 配置（~/.ssh/config）
# PKCS11Provider /usr/lib/pkcs11/libpkcs11-mock.so`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">验证驱动</div>
            <pre>{`# 确保 client-card 已在运行（IPC 通道就绪）
# 使用 pkcs11-tool 验证
pkcs11-tool --module pkcs11-mock.dll --list-slots
pkcs11-tool --module pkcs11-mock.dll --list-objects
# 应显示已配置的智能卡槽和证书对象`}</pre>
          </div>
        </div>
      </div>

      {/* ── TPM2 环境 ── */}
      <h2>🔒 四、TPM2 环境准备（可选）</h2>
      <table>
        <thead>
          <tr><th>平台</th><th>TPM 方案</th><th>降级方案</th></tr>
        </thead>
        <tbody>
          {[
            ['Windows 10+', 'TPM 2.0 原生支持', '降级为 Local Slot（纯软件加密）'],
            ['Linux Kernel 4.14+', 'TPM 2.0 + tpm2-abrmd', '降级为 Local Slot'],
            ['macOS Apple Silicon / T2', 'Secure Enclave（仅 EC P-256）', '降级为 Keychain → Local Slot'],
          ].map(([p, t, f]) => (
            <tr key={p as string}>
              <td>{p}</td>
              <td><span className="badge badge-green">{t}</span></td>
              <td style={{ fontSize: '0.8rem', color: 'var(--color-text-muted)' }}>{f}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <pre>{`# Linux - 安装 TPM2 工具
sudo apt install tpm2-tools tpm2-abrmd libtss2-dev   # Debian/Ubuntu
sudo dnf install tpm2-tools tpm2-abrmd tpm2-tss-devel # Fedora/RHEL

# 启动 TPM2 资源管理器
sudo systemctl enable --now tpm2-abrmd

# 授予用户访问权限
sudo usermod -aG tss $USER

# Windows - 检查 TPM 状态
Get-Tpm`}</pre>

      {/* ── Docker 部署 ── */}
      <h2>🐳 五、Docker 快速部署（云端平台）</h2>
      <pre>{`# docker-compose.yml
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

volumes:
  pgdata:`}</pre>
      <pre>{`# 启动服务
docker compose up -d

# 验证
curl http://localhost:1027/api/health`}</pre>

      {/* ── systemd ── */}
      <h2>⚙️ 六、systemd 服务（Linux 生产环境）</h2>
      <pre>{`# /etc/systemd/system/opencert-platform.service
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
WantedBy=multi-user.target`}</pre>
      <pre>{`sudo systemctl daemon-reload
sudo systemctl enable --now opencert-platform
sudo systemctl status opencert-platform`}</pre>

      {/* ── 端口规划 ── */}
      <h2>🌐 七、端口规划</h2>
      <table>
        <thead>
          <tr><th>服务</th><th>端口</th><th>协议</th><th>说明</th></tr>
        </thead>
        <tbody>
          {[
            ['client-card API / UI', '1026', 'HTTP', '本地管理端，仅监听 127.0.0.1'],
            ['server-card API', '1027', 'HTTPS', '云端平台 REST API'],
            ['IPC 通道（Windows）', '—', 'Named Pipe', '\\\\.\\pipe\\opencert-pkcs11'],
            ['IPC 通道（Linux/macOS）', '—', 'Unix Socket', '/tmp/opencert-pkcs11.sock'],
            ['OCSP 服务', '1027', 'HTTP', '/ocsp/<path>'],
            ['CRL 服务', '1027', 'HTTP', '/crl/<path>'],
            ['ACME 服务', '1027', 'HTTPS', '/acme/<path>'],
          ].map(([s, p, proto, d]) => (
            <tr key={s as string}>
              <td><code>{s}</code></td>
              <td><span className="badge badge-blue">{p}</span></td>
              <td>{proto}</td>
              <td style={{ fontSize: '0.8rem' }}>{d}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* ── 安全加固 ── */}
      <h2>🛡️ 八、安全加固清单</h2>
      <div className="card-grid">
        <div className="info-card">
          <div className="info-card-title">💻 本地管理端</div>
          <div className="info-card-body">
            <ul style={{ paddingLeft: '1rem', margin: 0 }}>
              <li>API 绑定地址为 <code>127.0.0.1</code></li>
              <li>Bearer Token 文件权限 <code>0600</code></li>
              <li>SQLite 数据库文件权限 <code>0600</code></li>
              <li>IPC 通道权限正确设置</li>
              <li>审计日志已启用</li>
              <li>TPM 可用性已确认（高安全性模式）</li>
            </ul>
          </div>
        </div>
        <div className="info-card">
          <div className="info-card-title">☁️ 云端平台</div>
          <div className="info-card-body">
            <ul style={{ paddingLeft: '1rem', margin: 0 }}>
              <li>TLS 证书有效且自动续期</li>
              <li>数据库连接使用 SSL</li>
              <li>JWT 密钥长度 ≥ 256 位</li>
              <li>配置反向代理（Nginx/Caddy）</li>
              <li>防火墙规则已配置</li>
              <li>定期备份 PostgreSQL 数据库</li>
            </ul>
          </div>
        </div>
      </div>

      <div className="callout callout-success">
        <span className="callout-icon">✅</span>
        <div className="callout-body">
          <strong>部署完成</strong>
          完成以上步骤后，可通过浏览器访问 <code>http://localhost:1026</code> 使用本地管理端，
          创建智能卡、导入证书，并通过 PKCS#11 驱动在系统中使用这些证书。
        </div>
      </div>

      <h2>🔗 下一步</h2>
      <ul>
        <li>查看 <a href="#/overview">功能概览</a> 了解完整功能模块</li>
        <li>查看 <a href="#/architecture">系统架构</a> 了解组件设计原理</li>
        <li>查看 <a href="#/api">API 文档</a> 了解完整 REST API 接口</li>
        <li>查看 <a href="#/driver">PKCS#11 驱动</a> 了解驱动集成细节</li>
        <li>查看 <a href="#/security">安全设计</a> 了解密钥保护机制</li>
      </ul>
    </DocPage>
  )
}
