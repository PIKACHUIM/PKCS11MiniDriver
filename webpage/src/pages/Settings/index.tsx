import React, { useState } from 'react';
import { Card, Form, Input, Button, Switch, Typography, Space, Divider, message, Tag, Alert } from 'antd';
import { SaveOutlined, ApiOutlined, BulbOutlined } from '@ant-design/icons';
import { useAppStore } from '../../store/appStore';

const { Title, Text } = Typography;

const Settings: React.FC = () => {
  const { darkMode, toggleDarkMode, connected, serverVersion, checkConnection } = useAppStore();
  const [apiBase, setApiBase] = useState(
    localStorage.getItem('apiBase') || 'http://localhost:1026'
  );
  const [testing, setTesting] = useState(false);

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
    marginBottom: 16,
  };

  const handleSaveApiBase = () => {
    localStorage.setItem('apiBase', apiBase);
    message.success('已保存，刷新页面后生效');
  };

  const handleTestConnection = async () => {
    setTesting(true);
    try {
      await checkConnection();
      if (connected) {
        message.success(`连接成功，服务版本 v${serverVersion}`);
      } else {
        message.error('连接失败，请检查 client-card 是否已启动');
      }
    } finally {
      setTesting(false);
    }
  };

  return (
    <div style={{ padding: 24, maxWidth: 680 }}>
      <Title level={4} style={{ marginBottom: 24, color: darkMode ? '#c9d1d9' : undefined }}>
        系统设置
      </Title>

      {/* 连接设置 */}
      <Card
        title={<Space><ApiOutlined /><span>服务连接</span></Space>}
        style={cardStyle}
        headStyle={{ borderBottom: darkMode ? '1px solid #21262d' : undefined }}
      >
        <Alert
          message={connected ? `已连接 client-card v${serverVersion}` : '未连接到 client-card 服务'}
          type={connected ? 'success' : 'warning'}
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form layout="vertical">
          <Form.Item
            label={<Text style={{ color: darkMode ? '#c9d1d9' : undefined }}>client-card 服务地址</Text>}
            extra="修改后需刷新页面生效"
          >
            <Space.Compact style={{ width: '100%' }}>
              <Input
                value={apiBase}
                onChange={(e) => setApiBase(e.target.value)}
                placeholder="http://localhost:1026"
                style={{ flex: 1 }}
              />
              <Button onClick={handleSaveApiBase} icon={<SaveOutlined />}>保存</Button>
            </Space.Compact>
          </Form.Item>

          <Button
            onClick={handleTestConnection}
            loading={testing}
            icon={<ApiOutlined />}
          >
            测试连接
          </Button>
        </Form>
      </Card>

      {/* 外观设置 */}
      <Card
        title={<Space><BulbOutlined /><span>外观</span></Space>}
        style={cardStyle}
        headStyle={{ borderBottom: darkMode ? '1px solid #21262d' : undefined }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <Text strong style={{ color: darkMode ? '#c9d1d9' : undefined }}>暗色模式</Text>
            <br />
            <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#999' }}>
              切换界面主题
            </Text>
          </div>
          <Switch
            checked={darkMode}
            onChange={toggleDarkMode}
            checkedChildren="暗色"
            unCheckedChildren="亮色"
          />
        </div>
      </Card>

      {/* 关于 */}
      <Card
        title="关于"
        style={cardStyle}
        headStyle={{ borderBottom: darkMode ? '1px solid #21262d' : undefined }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>前端版本</Text>
            <Tag color="blue">v1.0.0</Tag>
          </div>
          <Divider style={{ margin: '8px 0', borderColor: darkMode ? '#21262d' : undefined }} />
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>client-card 版本</Text>
            <Tag color={connected ? 'success' : 'default'}>
              {connected ? `v${serverVersion}` : '未连接'}
            </Tag>
          </div>
          <Divider style={{ margin: '8px 0', borderColor: darkMode ? '#21262d' : undefined }} />
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>技术栈</Text>
            <Space>
              <Tag>React 18</Tag>
              <Tag>Ant Design 5</Tag>
              <Tag>Vite</Tag>
            </Space>
          </div>
          <Divider style={{ margin: '8px 0', borderColor: darkMode ? '#21262d' : undefined }} />
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>项目</Text>
            <Text style={{ color: darkMode ? '#8b949e' : '#666', fontSize: 12 }}>
              GlobalTrusts PKCS11Driver
            </Text>
          </div>
        </Space>
      </Card>
    </div>
  );
};

export default Settings;
