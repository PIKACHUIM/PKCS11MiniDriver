import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Typography, Row, Col, Card, Steps, Space, Tag } from 'antd';
import {
  SafetyCertificateOutlined, CreditCardOutlined, ApiOutlined,
  CloudServerOutlined, FileProtectOutlined, KeyOutlined,
  ArrowRightOutlined, CheckCircleOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../../store/auth';

const { Title, Text, Paragraph } = Typography;

const features = [
  { icon: <SafetyCertificateOutlined />, title: 'CA 管理', desc: '创建自签名 CA，管理证书链，支持 CRL/OCSP/AIA 吊销服务', color: '#1677ff' },
  { icon: <CreditCardOutlined />, title: '智能卡支持', desc: '支持本地虚拟卡、TPM2、云端智能卡，多安全等级密钥存储', color: '#722ed1' },
  { icon: <ApiOutlined />, title: 'PKCS#11 驱动', desc: '标准 PKCS#11 接口，兼容主流浏览器和应用程序', color: '#13c2c2' },
  { icon: <CloudServerOutlined />, title: 'ACME 服务', desc: '内置 ACME 协议支持，自动化证书申请与续期', color: '#52c41a' },
  { icon: <FileProtectOutlined />, title: '证书申请', desc: '支持 X.509/GPG/SSH 证书，完整的申请审批工作流', color: '#fa8c16' },
  { icon: <KeyOutlined />, title: 'TOTP 管理', desc: '云端 TOTP 密钥安全存储，支持 SHA1/SHA256/SHA512', color: '#eb2f96' },
];

const steps = [
  { title: '注册账号', desc: '创建您的 OpenCert 账号' },
  { title: '配置 CA', desc: '创建或导入证书颁发机构' },
  { title: '申请证书', desc: '选择模板，提交证书申请' },
  { title: '下载使用', desc: '将证书部署到您的应用' },
];

const Home: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();

  if (isAuthenticated) {
    navigate('/dashboard', { replace: true });
    return null;
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0d1117', color: '#e6edf3' }}>
      {/* 顶部导航 */}
      <div style={{
        position: 'fixed', top: 0, left: 0, right: 0, zIndex: 100,
        padding: '0 48px', height: 64,
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        background: 'rgba(13,17,23,0.85)', backdropFilter: 'blur(12px)',
        borderBottom: '1px solid rgba(48,54,61,0.6)',
      }}>
        <Space align="center">
          <div style={{
            width: 32, height: 32, borderRadius: 8,
            background: 'linear-gradient(135deg, #1677ff, #722ed1)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <SafetyCertificateOutlined style={{ color: '#fff', fontSize: 16 }} />
          </div>
          <Text strong style={{ color: '#e6edf3', fontSize: 18, letterSpacing: '-0.3px' }}>OpenCert</Text>
        </Space>
        <Space>
          <Button type="text" style={{ color: '#8b949e' }} onClick={() => navigate('/login')}>登录</Button>
          <Button type="primary" onClick={() => navigate('/login')}
            style={{ background: 'linear-gradient(135deg, #1677ff, #0958d9)', border: 'none', borderRadius: 8 }}>
            立即开始
          </Button>
        </Space>
      </div>

      {/* Hero 区域 */}
      <div style={{
        paddingTop: 160, paddingBottom: 100, textAlign: 'center',
        background: 'radial-gradient(ellipse at 50% 0%, rgba(22,119,255,0.12) 0%, transparent 60%)',
        position: 'relative',
      }}>
        <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none',
          backgroundImage: 'linear-gradient(rgba(48,54,61,0.3) 1px, transparent 1px), linear-gradient(90deg, rgba(48,54,61,0.3) 1px, transparent 1px)',
          backgroundSize: '60px 60px', opacity: 0.4 }} />
        <div style={{ position: 'relative', zIndex: 1, maxWidth: 800, margin: '0 auto', padding: '0 24px' }}>
          <Tag color="blue" style={{ marginBottom: 24, borderRadius: 20, padding: '4px 16px', fontSize: 13 }}>
            🔐 企业级 PKI 管理平台
          </Tag>
          <Title style={{ color: '#e6edf3', fontSize: 56, fontWeight: 800, lineHeight: 1.15, margin: '0 0 24px', letterSpacing: '-1.5px' }}>
            安全、简单的<br />
            <span style={{ background: 'linear-gradient(135deg, #1677ff, #722ed1)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
              证书管理平台
            </span>
          </Title>
          <Paragraph style={{ color: '#8b949e', fontSize: 18, maxWidth: 560, margin: '0 auto 40px', lineHeight: 1.7 }}>
            OpenCert 提供完整的 CA 管理、智能卡支持、PKCS#11 驱动和自动化证书申请，
            让企业 PKI 基础设施管理变得简单高效。
          </Paragraph>
          <Space size={16}>
            <Button type="primary" size="large" icon={<ArrowRightOutlined />}
              onClick={() => navigate('/login')}
              style={{ height: 48, padding: '0 32px', borderRadius: 10, fontWeight: 600, fontSize: 16,
                background: 'linear-gradient(135deg, #1677ff, #0958d9)', border: 'none',
                boxShadow: '0 8px 24px rgba(22,119,255,0.4)' }}>
              立即开始使用
            </Button>
            <Button size="large" ghost
              style={{ height: 48, padding: '0 32px', borderRadius: 10, fontWeight: 600, fontSize: 16,
                borderColor: 'rgba(48,54,61,0.8)', color: '#8b949e' }}>
              查看文档
            </Button>
          </Space>
        </div>
      </div>

      {/* 功能特性 */}
      <div style={{ padding: '80px 48px', maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ textAlign: 'center', marginBottom: 56 }}>
          <Title level={2} style={{ color: '#e6edf3', margin: '0 0 12px', fontWeight: 700 }}>核心功能</Title>
          <Text style={{ color: '#8b949e', fontSize: 16 }}>一站式 PKI 管理，覆盖证书全生命周期</Text>
        </div>
        <Row gutter={[24, 24]}>
          {features.map((f) => (
            <Col xs={24} sm={12} lg={8} key={f.title}>
              <Card hoverable style={{
                background: 'rgba(22,27,34,0.8)', border: '1px solid rgba(48,54,61,0.6)',
                borderRadius: 12, height: '100%',
                transition: 'border-color 0.3s, transform 0.3s',
              }}
                bodyStyle={{ padding: 28 }}
                onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = f.color; (e.currentTarget as HTMLElement).style.transform = 'translateY(-4px)'; }}
                onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'rgba(48,54,61,0.6)'; (e.currentTarget as HTMLElement).style.transform = 'translateY(0)'; }}
              >
                <div style={{ width: 48, height: 48, borderRadius: 12, background: `${f.color}20`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  fontSize: 22, color: f.color, marginBottom: 16 }}>
                  {f.icon}
                </div>
                <Title level={5} style={{ color: '#e6edf3', margin: '0 0 8px', fontWeight: 600 }}>{f.title}</Title>
                <Text style={{ color: '#8b949e', fontSize: 14, lineHeight: 1.6 }}>{f.desc}</Text>
              </Card>
            </Col>
          ))}
        </Row>
      </div>

      {/* 三层架构 */}
      <div style={{ padding: '80px 48px', background: 'rgba(22,27,34,0.4)', borderTop: '1px solid rgba(48,54,61,0.4)', borderBottom: '1px solid rgba(48,54,61,0.4)' }}>
        <div style={{ maxWidth: 1000, margin: '0 auto', textAlign: 'center' }}>
          <Title level={2} style={{ color: '#e6edf3', margin: '0 0 12px', fontWeight: 700 }}>三层架构设计</Title>
          <Text style={{ color: '#8b949e', fontSize: 16 }}>Platform + Manager + Driver，灵活部署，安全可靠</Text>
          <Row gutter={[0, 0]} style={{ marginTop: 48, border: '1px solid rgba(48,54,61,0.6)', borderRadius: 16, overflow: 'hidden' }}>
            {[
              { name: 'OpenCert Platform', port: ':1027', desc: '云端平台服务，CA 管理、证书颁发、用户管理、支付系统', color: '#1677ff', tag: '云端' },
              { name: 'OpenCert Manager', port: ':1026', desc: '本地管理端，智能卡管理、本地 PKI、TOTP 验证器', color: '#722ed1', tag: '本地' },
              { name: 'OpenCert Driver', port: 'PKCS#11', desc: '标准 PKCS#11 驱动，兼容浏览器和应用程序', color: '#13c2c2', tag: '驱动' },
            ].map((layer, i) => (
              <Col xs={24} md={8} key={layer.name} style={{
                padding: 32, background: i === 1 ? 'rgba(22,27,34,0.6)' : 'transparent',
                borderRight: i < 2 ? '1px solid rgba(48,54,61,0.6)' : 'none',
              }}>
                <Tag color={layer.color} style={{ marginBottom: 16, borderRadius: 20, padding: '2px 12px' }}>{layer.tag}</Tag>
                <Title level={4} style={{ color: '#e6edf3', margin: '0 0 8px', fontWeight: 700 }}>{layer.name}</Title>
                <Text style={{ color: layer.color, fontFamily: 'monospace', fontSize: 13 }}>{layer.port}</Text>
                <Paragraph style={{ color: '#8b949e', fontSize: 14, marginTop: 12, lineHeight: 1.6 }}>{layer.desc}</Paragraph>
              </Col>
            ))}
          </Row>
        </div>
      </div>

      {/* 快速开始 */}
      <div style={{ padding: '80px 48px', maxWidth: 800, margin: '0 auto', textAlign: 'center' }}>
        <Title level={2} style={{ color: '#e6edf3', margin: '0 0 12px', fontWeight: 700 }}>快速开始</Title>
        <Text style={{ color: '#8b949e', fontSize: 16 }}>四步完成证书申请</Text>
        <div style={{ marginTop: 48 }}>
          <Steps direction="vertical" current={-1} style={{ textAlign: 'left' }}
            items={steps.map((s, i) => ({
              title: <Text strong style={{ color: '#e6edf3' }}>{s.title}</Text>,
              description: <Text style={{ color: '#8b949e' }}>{s.desc}</Text>,
              icon: <div style={{ width: 32, height: 32, borderRadius: '50%',
                background: 'linear-gradient(135deg, #1677ff, #722ed1)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                color: '#fff', fontWeight: 700, fontSize: 14 }}>{i + 1}</div>,
            }))}
          />
        </div>
        <Button type="primary" size="large" icon={<CheckCircleOutlined />}
          onClick={() => navigate('/login')}
          style={{ marginTop: 48, height: 48, padding: '0 40px', borderRadius: 10, fontWeight: 600, fontSize: 16,
            background: 'linear-gradient(135deg, #1677ff, #722ed1)', border: 'none',
            boxShadow: '0 8px 24px rgba(22,119,255,0.3)' }}>
          免费注册，立即体验
        </Button>
      </div>

      {/* 页脚 */}
      <div style={{ padding: '32px 48px', borderTop: '1px solid rgba(48,54,61,0.4)', textAlign: 'center' }}>
        <Text style={{ color: '#8b949e', fontSize: 13 }}>© 2025 OpenCert · 企业级 PKI 管理平台 · 开源项目</Text>
      </div>
    </div>
  );
};

export default Home;
