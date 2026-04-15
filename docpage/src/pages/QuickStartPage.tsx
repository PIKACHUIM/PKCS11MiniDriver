import DocPage from '../components/DocPage'

export default function QuickStartPage() {
  return (
    <DocPage
      title="快速开始"
      subtitle="从零开始部署 OpenCert Manager，包括云端服务、本地客户端和 PKCS#11 驱动"
      badge="Quick Start"
    >
      <div className="callout callout-info">
        <span className="callout-icon">ℹ️</span>
        <div className="callout-body">
          <strong>前置要求</strong>
          Go 1.22+、Node.js 18+、Git。Windows 用户需要 MSVC 编译工具链（用于 C 驱动编译）。
        </div>
      </div>

      <h2>▶ 一、部署云端服务（server-card）</h2>
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
# 编辑 config.yaml，配置数据库、JWT 密钥等`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">编译并运行</div>
            <pre>{`go mod download
go build -o server-card ./cmd/server
./server-card --config configs/config.yaml`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">4</div>
          <div className="step-body">
            <div className="step-title">验证服务</div>
            <pre>{`curl http://localhost:8080/api/health
# 返回: {"data":{"status":"ok","version":"1.0.0"}}`}</pre>
          </div>
        </div>
      </div>

      <h2>▶ 二、运行本地客户端（client-card）</h2>
      <div className="steps">
        <div className="step">
          <div className="step-num">1</div>
          <div className="step-body">
            <div className="step-title">编译 Go 后端</div>
            <pre>{`cd clients
go mod download
go build -o client-card ./cmd/client`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">2</div>
          <div className="step-body">
            <div className="step-title">安装前端依赖并构建</div>
            <pre>{`cd webpage
npm install
npm run build`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">启动客户端</div>
            <pre>{`# 方式一：Web 模式（浏览器访问）
./client-card --mode web --port 1026

# 方式二：Electron 桌面模式
npm run electron`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">4</div>
          <div className="step-body">
            <div className="step-title">初始化配置</div>
            <div className="step-desc">
              打开 <code>http://localhost:5173</code>，创建第一个用户，配置云端服务地址（可选）。
            </div>
          </div>
        </div>
      </div>

      <h2>▶ 三、安装 PKCS#11 驱动</h2>
      <div className="steps">
        <div className="step">
          <div className="step-num">1</div>
          <div className="step-body">
            <div className="step-title">编译驱动（Windows）</div>
            <pre>{`cd drivers
# 使用 Visual Studio 或 MinGW
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release
# 输出: build/Release/pkcs11-mock.dll`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">2</div>
          <div className="step-body">
            <div className="step-title">注册驱动</div>
            <pre>{`# Windows - 注册到系统
regsvr32 pkcs11-mock.dll

# Linux - 复制到标准路径
sudo cp pkcs11-mock.so /usr/lib/pkcs11/`}</pre>
          </div>
        </div>
        <div className="step">
          <div className="step-num">3</div>
          <div className="step-body">
            <div className="step-title">验证驱动</div>
            <pre>{`# 使用 pkcs11-tool 验证
pkcs11-tool --module pkcs11-mock.dll --list-slots
# 应显示已配置的智能卡槽`}</pre>
          </div>
        </div>
      </div>

      <h2>🐳 Docker 快速部署</h2>
      <pre>{`# 使用 Docker Compose 一键部署云端服务
version: '3.8'
services:
  server-card:
    image: ghcr.io/pikachuim/pkcs11minidriver/server:latest
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
      - ./data:/app/data
    environment:
      - CONFIG_PATH=/app/configs/config.yaml`}</pre>

      <div className="callout callout-success">
        <span className="callout-icon">✅</span>
        <div className="callout-body">
          <strong>部署完成</strong>
          完成以上步骤后，你可以通过浏览器访问 client-card Web UI，
          创建智能卡、导入证书，并通过 PKCS#11 驱动在系统中使用这些证书。
        </div>
      </div>

      <h2>🔗 下一步</h2>
      <ul>
        <li>查看 <a href="#/api">API 文档</a> 了解完整的 REST API 接口</li>
        <li>查看 <a href="#/driver">PKCS#11 驱动</a> 了解驱动集成细节</li>
        <li>查看 <a href="#/security">安全设计</a> 了解密钥保护机制</li>
      </ul>
    </DocPage>
  )
}
