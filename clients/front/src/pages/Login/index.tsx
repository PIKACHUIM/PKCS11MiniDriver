import React, { useState } from 'react';
import { Form, Input, Button, message, Typography, Space } from 'antd';
import { UserOutlined, LockOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { login } from '../../api';
import { useAuthStore } from '../../store/auth';
import { useAppStore } from '../../store/appStore';

const { Title, Text } = Typography;

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { setAuth } = useAuthStore();
  const { darkMode } = useAppStore();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  // 登录成功后跳转到原目标页，默认 /dashboard
  const from = (location.state as any)?.from?.pathname || '/dashboard';

  const handleLogin = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const auth = await login(values);
      setAuth(auth);
      message.success('登录成功');
      navigate(from, { replace: true });
    } catch (e: any) {
      if (e.message) message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  const bg = darkMode ? '#0d1117' : '#f0f2f5';
  const cardBg = darkMode ? '#161b22' : '#fff';
  const border = darkMode ? '1px solid rgba(255,255,255,0.08)' : '1px solid rgba(0,0,0,0.06)';

  return (
    <div style={{
      minHeight: '100vh',
      background: bg,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    }}>
      <div style={{
        width: 380,
        padding: '40px 36px',
        background: cardBg,
        border,
        borderRadius: 16,
        boxShadow: darkMode
          ? '0 8px 40px rgba(0,0,0,0.5)'
          : '0 8px 40px rgba(0,0,0,0.1)',
      }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{
            width: 56,
            height: 56,
            borderRadius: 16,
            background: 'linear-gradient(135deg, #1677ff, #722ed1)',
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 6px 20px rgba(22,119,255,0.4)',
            marginBottom: 16,
          }}>
            <SafetyCertificateOutlined style={{ fontSize: 28, color: '#fff' }} />
          </div>
          <Title level={4} style={{ margin: 0, color: darkMode ? '#c9d1d9' : '#1a1a2e' }}>
            OpenCert Manager
          </Title>
          <Text style={{ color: darkMode ? '#8b949e' : '#999', fontSize: 13 }}>
            本地智能卡管理系统
          </Text>
        </div>

        <Form
          form={form}
          layout="vertical"
          onFinish={handleLogin}
          initialValues={{ username: 'admin' }}
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined style={{ color: '#8b949e' }} />}
              placeholder="用户名"
              size="large"
              autoComplete="username"
            />
          </Form.Item>
          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: '#8b949e' }} />}
              placeholder="密码"
              size="large"
              autoComplete="current-password"
            />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, marginTop: 8 }}>
            <Button
              type="primary"
              htmlType="submit"
              size="large"
              block
              loading={loading}
              style={{
                background: 'linear-gradient(135deg, #1677ff, #722ed1)',
                border: 'none',
                height: 44,
                fontSize: 15,
                fontWeight: 600,
              }}
            >
              登 录
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center', marginTop: 20 }}>
          <Text style={{ fontSize: 12, color: darkMode ? '#6e7681' : '#bbb' }}>
            默认账号：admin / admin
          </Text>
        </div>
      </div>
    </div>
  );
};

export default LoginPage;
