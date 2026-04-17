import DocPage from '../components/DocPage'

interface EndpointProps {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE'
  path: string
  desc: string
  body?: string
  response?: string
}

function Endpoint({ method, path, desc, body, response }: EndpointProps) {
  return (
    <div className="api-endpoint">
      <div className="api-endpoint-header">
        <span className={`method-badge method-${method.toLowerCase()}`}>{method}</span>
        <span className="api-path">{path}</span>
        <span className="api-desc">{desc}</span>
      </div>
      {(body || response) && (
        <div className="api-endpoint-body">
          {body && (
            <>
              <p style={{ fontSize: '0.8rem', color: 'var(--color-text-muted)', marginBottom: '0.5rem' }}>请求体：</p>
              <pre style={{ margin: 0 }}>{body}</pre>
            </>
          )}
          {response && (
            <>
              <p style={{ fontSize: '0.8rem', color: 'var(--color-text-muted)', margin: '0.75rem 0 0.5rem' }}>响应示例：</p>
              <pre style={{ margin: 0 }}>{response}</pre>
            </>
          )}
        </div>
      )}
    </div>
  )
}

export default function ApiPage() {
  return (
    <DocPage
      title="API 文档"
      subtitle="OpenCert Manager REST API 参考，基础路径 /api，使用 Bearer Token 认证"
      badge="API v1.1"
    >
      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>通用说明</strong>
          基础路径：<code>http://127.0.0.1:1026/api</code>（client-card）或 <code>https://your-server/api</code>（server-card）
          <br />
          认证方式：<code>Authorization: Bearer &lt;token&gt;</code>
          <br />
          响应格式：成功 <code>{"{ \"data\": ... }"}</code>，失败 <code>{"{ \"error\": \"描述\" }"}</code>
        </div>
      </div>

      <h2>🏥 健康检查</h2>
      <Endpoint
        method="GET"
        path="/api/health"
        desc="服务健康状态"
        response={`{ "data": { "status": "ok", "version": "1.0.0", "time": "1713100000" } }`}
      />

      <h2>👤 用户管理</h2>
      <Endpoint method="GET"    path="/api/users"        desc="用户列表" />
      <Endpoint method="POST"   path="/api/users"        desc="创建用户"
        body={`{
  "user_type": "local",
  "display_name": "张三",
  "email": "zhangsan@example.com",
  "password": "secure_password",
  "role": "user"
}`} />
      <Endpoint method="GET"    path="/api/users/{uuid}" desc="用户详情" />
      <Endpoint method="PUT"    path="/api/users/{uuid}" desc="更新用户" />
      <Endpoint method="DELETE" path="/api/users/{uuid}" desc="删除用户" />

      <h3>用户认证</h3>
      <Endpoint method="POST" path="/api/auth/login" desc="用户登录"
        body={`{ "email": "user@example.com", "password": "password", "totp_code": "123456" }`}
        response={`{ "data": { "token": "eyJ...", "expires_at": 1713200000 } }`}
      />
      <Endpoint method="POST" path="/api/auth/refresh" desc="刷新 Token" />
      <Endpoint method="POST" path="/api/auth/logout"  desc="登出" />

      <h2>💳 卡片管理</h2>
      <Endpoint method="GET"    path="/api/cards"        desc="卡片列表" />
      <Endpoint method="POST"   path="/api/cards"        desc="创建卡片"
        body={`{
  "slot_type": "local",
  "card_name": "我的智能卡",
  "user_uuid": "user-uuid-here",
  "password": "card_password"
}`} />
      <Endpoint method="GET"    path="/api/cards/{uuid}" desc="卡片详情" />
      <Endpoint method="PUT"    path="/api/cards/{uuid}" desc="更新卡片" />
      <Endpoint method="DELETE" path="/api/cards/{uuid}" desc="删除卡片" />

      <h2>📜 证书管理</h2>
      <Endpoint method="GET"    path="/api/cards/{card_uuid}/certs"       desc="证书列表" />
      <Endpoint method="POST"   path="/api/cards/{card_uuid}/certs"       desc="导入证书"
        body={`{
  "cert_type": "x509",
  "key_type": "ec256",
  "cert_data": "-----BEGIN CERTIFICATE-----...",
  "key_data": "-----BEGIN PRIVATE KEY-----...",
  "password": "card_password"
}`} />
      <Endpoint method="GET"    path="/api/cards/{card_uuid}/certs/{uuid}" desc="证书详情" />
      <Endpoint method="DELETE" path="/api/cards/{card_uuid}/certs/{uuid}" desc="删除证书" />

      <h2>🔑 密钥操作</h2>
      <Endpoint method="POST" path="/api/cards/{card_uuid}/keygen" desc="片上生成密钥"
        body={`{ "key_type": "ec256", "password": "card_password" }`}
        response={`{ "data": { "key_id": "key-uuid", "public_key": "-----BEGIN PUBLIC KEY-----..." } }`}
      />
      <Endpoint method="POST" path="/api/cards/{card_uuid}/sign"   desc="数据签名"
        body={`{ "key_id": "key-uuid", "data": "base64-encoded-data", "mechanism": "SHA256withECDSA", "password": "card_password" }`}
      />
      <Endpoint method="POST" path="/api/cards/{card_uuid}/csr"    desc="生成 CSR（驱动签名）"
        body={`{ "key_id": "key-uuid", "subject": { "CN": "example.com", "O": "My Org" }, "password": "card_password" }`}
      />

      <h2>🔐 TOTP 管理</h2>
      <Endpoint method="GET"    path="/api/cards/{card_uuid}/totp" desc="TOTP 条目列表" />
      <Endpoint method="POST"   path="/api/cards/{card_uuid}/totp" desc="添加 TOTP 条目"
        body={`{ "name": "GitHub", "secret": "JBSWY3DPEHPK3PXP", "digits": 6, "period": 30 }`}
      />
      <Endpoint method="GET"    path="/api/totp/{id}/code"         desc="获取当前验证码"
        response={`{ "data": { "code": "123456", "expires_in": 15 } }`}
      />
      <Endpoint method="DELETE" path="/api/totp/{id}"              desc="删除 TOTP 条目" />

      <h2>🏛️ CA 管理（云端）</h2>
      <Endpoint method="GET"  path="/api/ca"        desc="CA 列表" />
      <Endpoint method="POST" path="/api/ca"        desc="导入或创建 CA" />
      <Endpoint method="GET"  path="/api/ca/{uuid}" desc="CA 详情" />
      <Endpoint method="POST" path="/api/ca/{uuid}/issue" desc="颁发证书"
        body={`{
  "template_uuid": "template-uuid",
  "subject": { "CN": "example.com", "O": "My Org" },
  "san": { "dns": ["example.com", "www.example.com"] },
  "validity_days": 365,
  "csr": "-----BEGIN CERTIFICATE REQUEST-----..."
}`}
      />
      <Endpoint method="POST" path="/api/ca/{uuid}/revoke/{cert_uuid}" desc="吊销证书" />

      <h2>📋 证书模板</h2>
      <Endpoint method="GET"    path="/api/templates"        desc="模板列表" />
      <Endpoint method="POST"   path="/api/templates"        desc="创建模板" />
      <Endpoint method="GET"    path="/api/templates/{uuid}" desc="模板详情" />
      <Endpoint method="PUT"    path="/api/templates/{uuid}" desc="更新模板" />
      <Endpoint method="DELETE" path="/api/templates/{uuid}" desc="删除模板" />

      <h2>🤖 ACME 服务</h2>
      <Endpoint method="GET"  path="/acme/{path}/directory"                    desc="ACME 目录" />
      <Endpoint method="POST" path="/acme/{path}/new-account"                  desc="创建账户" />
      <Endpoint method="POST" path="/acme/{path}/new-order"                    desc="创建订单" />
      <Endpoint method="POST" path="/acme/{path}/order/{id}/finalize"          desc="完成订单" />
      <Endpoint method="GET"  path="/acme/{path}/certificate/{id}"             desc="下载证书" />

      <h2>🔍 吊销服务</h2>
      <Endpoint method="GET"  path="/ocsp/{path}"  desc="OCSP 查询（RFC 6960）" />
      <Endpoint method="POST" path="/ocsp/{path}"  desc="OCSP 查询（POST 方式）" />
      <Endpoint method="GET"  path="/crl/{path}"   desc="CRL 下载（DER 格式）" />
      <Endpoint method="GET"  path="/ca/{path}"    desc="CA 证书下载" />
    </DocPage>
  )
}
