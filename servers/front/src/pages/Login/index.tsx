import React, { useState } from 'react';
import { useNavigate, Navigate } from 'react-router-dom';
import { Form, Input, Button, Tabs, message, Typography, Space } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, IdcardOutlined, SafetyOutlined } from '@ant-design/icons';
import { login, register } from '../../api';
import { useAuthStore } from '../../store/auth';

const { Title, Text } = Typography;

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { setAuth, isAuthenticated } = useAuthStore();
  const [loginLoading, setLoginLoading] = useState(false);
  const [registerLoading, setRegisterLoading] = useState(false);

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  const handleLogin = async (values: { username: string; password: string }) => {
    setLoginLoading(true);
    try {
      const auth = await login(values);
      setAuth(auth);
      message.success('登录成功');
      navigate('/dashboard', { replace: true });
    } catch (err: any) {
      message.error(err.message || '用户名或密码错误');
    } finally {
      setLoginLoading(false);
    }
  };

  const handleRegister = async (values: { username: string; password: string; confirm: string; email: string; display_name: string }) => {
    if (values.password !== values.confirm) { message.error('两次密码不一致'); return; }
    setRegisterLoading(true);
    try {
      const auth = await register({ username: values.username, password: values.password, email: values.email, display_name: values.display_name });
      setAuth(auth);
      message.success('注册成功');
      navigate('/dashboard', { replace: true });
    } catch (err: any) {
      message.error(err.message || '注册失败');
    } finally {
      setRegisterLoading(false);
    }
  };

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: 'linear-gradient(135deg, #0d1117 0%, #161b22 50%, #0d1117 100%)',
      position: 'relative', overflow: 'hidden',
    }}>
      <div style={{ position: 'absolute', inset: 0, pointerEvents: 'none',
        background: 'radial-gradient(ellipse at 20% 50%, rgba(22,119,255,0.15) 0%, transparent 60%), radial-gradient(ellipse at 80% 20%, rgba(114,46,209,0.1) 0%, transparent 50%)' }} />
      <div style={{
        width: 420, padding: '40px 40px 32px',
        background: 'rgba(22,27,34,0.85)', backdropFilter: 'blur(20px)',
        border: '1px solid rgba(48,54,61,0.8)', borderRadius: 16,
        boxShadow: '0 24px 64px rgba(0,0,0,0.5)', position: 'relative', zIndex: 1,
      }}>
        <Space direction="vertical" align="center" style={{ width: '100%', marginBottom: 32 }}>
          <div style={{
            width: 56, height: 56, borderRadius: 14,
            background: 'linear-gradient(135deg, #1677ff, #722ed1)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 28, boxShadow: '0 8px 24px rgba(22,119,255,0.4)',
          }}>
            <SafetyOutlined style={{ color: '#fff' }} />
          </div>
          <Title level={3} style={{ margin: 0, color: '#e6edf3', fontWeight: 700, letterSpacing: '-0.5px' }}>OpenCert Platform</Title>
          <Text style={{ color: '#8b949e', fontSize: 13 }}>企业级证书管理云平台</Text>
        </Space>

        <Tabs centered items={[
          {
            key: 'login', label: '登录',
            children: (
              <Form layout="vertical" onFinish={handleLogin} size="large" style={{ marginTop: 8 }}>
                <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
                  <Input prefix={<UserOutlined style={{ color: '#8b949e' }} />} placeholder="用户名"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
                  <Input.Password prefix={<LockOutlined style={{ color: '#8b949e' }} />} placeholder="密码"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item style={{ marginBottom: 0, marginTop: 8 }}>
                  <Button type="primary" htmlType="submit" block loading={loginLoading}
                    style={{ height: 44, borderRadius: 8, fontWeight: 600, fontSize: 15,
                      background: 'linear-gradient(135deg, #1677ff, #0958d9)', border: 'none',
                      boxShadow: '0 4px 16px rgba(22,119,255,0.4)' }}>
                    登录
                  </Button>
                </Form.Item>
              </Form>
            ),
          },
          {
            key: 'register', label: '注册',
            children: (
              <Form layout="vertical" onFinish={handleRegister} size="large" style={{ marginTop: 8 }}>
                <Form.Item name="username" rules={[{ required: true }, { min: 3, message: '用户名至少3位' }]}>
                  <Input prefix={<UserOutlined style={{ color: '#8b949e' }} />} placeholder="用户名（至少3位）"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item name="display_name" rules={[{ required: true, message: '请输入显示名称' }]}>
                  <Input prefix={<IdcardOutlined style={{ color: '#8b949e' }} />} placeholder="显示名称"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item name="email" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
                  <Input prefix={<MailOutlined style={{ color: '#8b949e' }} />} placeholder="邮箱"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item name="password" rules={[{ required: true }, { min: 8, message: '密码至少8位' }]}>
                  <Input.Password prefix={<LockOutlined style={{ color: '#8b949e' }} />} placeholder="密码（至少8位）"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item name="confirm" rules={[{ required: true, message: '请确认密码' }]}>
                  <Input.Password prefix={<LockOutlined style={{ color: '#8b949e' }} />} placeholder="确认密码"
                    style={{ background: 'rgba(13,17,23,0.6)', border: '1px solid rgba(48,54,61,0.8)', color: '#e6edf3', borderRadius: 8 }} />
                </Form.Item>
                <Form.Item style={{ marginBottom: 0, marginTop: 8 }}>
                  <Button type="primary" htmlType="submit" block loading={registerLoading}
                    style={{ height: 44, borderRadius: 8, fontWeight: 600, fontSize: 15,
                      background: 'linear-gradient(135deg, #722ed1, #531dab)', border: 'none',
                      boxShadow: '0 4px 16px rgba(114,46,209,0.4)' }}>
                    注册账号
                  </Button>
                </Form.Item>
              </Form>
            ),
          },
        ]} />

        <div style={{ textAlign: 'center', marginTop: 24, paddingTop: 16, borderTop: '1px solid rgba(48,54,61,0.5)' }}>
          <Text style={{ color: '#8b949e', fontSize: 12 }}>© 2025 OpenCert Platform · 连接 server-card :1027</Text>
        </div>
      </div>
    </div>
  );
};

export default Login;
