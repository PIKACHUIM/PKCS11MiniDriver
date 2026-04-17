import DocPage from '../components/DocPage'

const algorithms = [
  { category: '非对称密钥', items: ['RSA 1024–8192', 'ECC P-256/P-384/P-521', 'Brainpool P256/P384/P512', 'Ed25519 / X25519', 'SM2'] },
  { category: '摘要算法', items: ['SHA-1 / SHA-256 / SHA-512', 'SHA3-256 / SHA3-512', 'MD5 / MD4', 'SM3'] },
  { category: '对称加密', items: ['AES-128 / AES-256 (GCM/CBC)', 'RC4', 'ChaCha20-Poly1305', 'SM4'] },
  { category: '证书格式', items: ['X.509 v3', 'OpenPGP / GPG', 'SSH 公钥格式'] },
]

const pkcs11Functions = [
  { func: 'C_GetSlotList', desc: '列出所有 Slot（每张智能卡对应一个 Slot）' },
  { func: 'C_GetSlotInfo', desc: '获取 Slot 信息（名称、厂商、标志位）' },
  { func: 'C_GetTokenInfo', desc: '获取 Token 信息（标签、序列号、容量）' },
  { func: 'C_OpenSession', desc: '打开会话（支持 RO/RW 模式）' },
  { func: 'C_CloseSession', desc: '关闭会话' },
  { func: 'C_Login', desc: '用户登录（USER_PIN / SO_PIN）' },
  { func: 'C_Logout', desc: '用户登出' },
  { func: 'C_FindObjectsInit', desc: '初始化对象搜索（按属性过滤）' },
  { func: 'C_FindObjects', desc: '获取搜索结果（证书/密钥对象）' },
  { func: 'C_FindObjectsFinal', desc: '结束搜索' },
  { func: 'C_GetAttributeValue', desc: '获取对象属性（证书内容、密钥类型等）' },
  { func: 'C_SignInit', desc: '初始化签名操作' },
  { func: 'C_Sign', desc: '执行签名' },
  { func: 'C_DecryptInit', desc: '初始化解密操作' },
  { func: 'C_Decrypt', desc: '执行解密' },
  { func: 'C_GenerateKeyPair', desc: '片上生成密钥对' },
  { func: 'C_GenerateRandom', desc: '生成随机数' },
]

export default function DriverPage() {
  return (
    <DocPage
      title="PKCS#11 驱动"
      subtitle="pkcs11-mock 是符合 PKCS#11 v2.40 标准的动态库，通过 HTTP 与 client-card 通信，将虚拟智能卡注册到操作系统"
      badge="PKCS#11"
    >
      <h2>⚙ 驱动架构</h2>
      <pre>{`应用程序 (Firefox / Chrome / SSH / GPG)
    │
    │  PKCS#11 标准 C 接口
    ▼
pkcs11-mock.dll / pkcs11-mock.so
    │
    │  HTTP REST API (localhost:1026)
    ▼
client-card (Go 后端)
    │
    ├── local slot  → SQLite 本地数据库
    ├── tpmv2 slot  → TPM2 硬件
    └── cloud slot  → server-card 云端 API`}</pre>

      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>通信安全</strong>
          pkcs11-mock 与 client-card 之间通过 localhost 通信，
          使用启动时自动生成的随机 Bearer Token 认证，防止其他进程伪造请求。
        </div>
      </div>

      <h2>🔧 支持的算法</h2>
      <div className="card-grid">
        {algorithms.map((a) => (
          <div key={a.category} className="info-card">
            <div className="info-card-title">🔐 {a.category}</div>
            <div className="info-card-body">
              <ul style={{ paddingLeft: '1rem', margin: 0 }}>
                {a.items.map((item) => (
                  <li key={item} style={{ marginBottom: '2px' }}>{item}</li>
                ))}
              </ul>
            </div>
          </div>
        ))}
      </div>

      <h2>📋 实现的 PKCS#11 函数</h2>
      <table>
        <thead>
          <tr><th>函数</th><th>说明</th></tr>
        </thead>
        <tbody>
          {pkcs11Functions.map((f) => (
            <tr key={f.func}>
              <td><code>{f.func}</code></td>
              <td>{f.desc}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>🔑 片上密钥生成与 CSR 签名</h2>
      <p>
        为确保私钥真正在智能卡上生成（而非用户自行生成后导入），
        pkcs11-mock 实现了 CSR 签名机制：
      </p>
      <pre>{`# 流程
1. 应用调用 C_GenerateKeyPair → client-card 生成密钥对
2. 应用调用 C_Sign 对 CSR 数据签名
3. client-card 使用卡片私钥对 CSR 进行二次签名
4. 云端验证二次签名，确认 CSR 来自合法智能卡
5. 颁发证书`}</pre>

      <h2>🖥️ 安装配置</h2>
      <h3>Windows</h3>
      <pre>{`# 方式一：直接注册
regsvr32 C:\\path\\to\\pkcs11-mock.dll

# 方式二：Firefox 配置
# 打开 about:config → security.osclientcerts.autoload = true
# 或在 Firefox 证书管理器中手动加载 DLL

# 方式三：Chrome/Edge（通过 Windows 证书存储）
# 驱动会自动将证书同步到 Windows 证书存储`}</pre>

      <h3>Linux / macOS</h3>
      <pre>{`# 复制到标准路径
sudo cp pkcs11-mock.so /usr/lib/x86_64-linux-gnu/pkcs11/

# Firefox 配置
# 编辑 ~/.mozilla/firefox/<profile>/pkcs11.txt
# 或使用 modutil 工具
modutil -dbdir ~/.mozilla/firefox/<profile> \\
        -add "OpenCert PKCS11" \\
        -libfile /usr/lib/pkcs11/pkcs11-mock.so

# SSH 配置
# ~/.ssh/config
PKCS11Provider /usr/lib/pkcs11/pkcs11-mock.so`}</pre>

      <h3>验证安装</h3>
      <pre>{`# 使用 pkcs11-tool 验证
pkcs11-tool --module pkcs11-mock.dll --list-slots
pkcs11-tool --module pkcs11-mock.dll --list-objects
pkcs11-tool --module pkcs11-mock.dll --test`}</pre>

      <h2>🔐 密钥安全级别</h2>
      <table>
        <thead>
          <tr><th>级别</th><th>存储</th><th>TPM 保护</th><th>可导出</th><th>云端备份</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><span className="badge badge-green">高安全</span></td>
            <td>TPM 内部</td>
            <td>✅ 片上不可导出</td>
            <td>❌</td>
            <td>❌</td>
          </tr>
          <tr>
            <td><span className="badge badge-orange">中安全</span></td>
            <td>本地加密文件</td>
            <td>✅ TPM 密钥加密</td>
            <td>✅（需 TPM）</td>
            <td>可选</td>
          </tr>
          <tr>
            <td><span className="badge badge-blue">低安全</span></td>
            <td>本地加密文件</td>
            <td>❌ 密码加密</td>
            <td>✅</td>
            <td>可选</td>
          </tr>
        </tbody>
      </table>

      <div className="callout callout-warning">
        <span className="callout-icon">⚠️</span>
        <div className="callout-body">
          <strong>TPM2 要求</strong>
          TPM2 高安全模式需要系统具备 TPM 2.0 芯片，并且已在 BIOS 中启用。
          Windows 11 设备通常已满足此要求。
        </div>
      </div>
    </DocPage>
  )
}
