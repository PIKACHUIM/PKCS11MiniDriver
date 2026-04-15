import React, { useState, useEffect } from 'react';
import {
  Card, Form, Input, Button, Typography, Space, Divider, message, Tag, Alert,
  Segmented, Tabs, Table, Modal, Select, InputNumber, Switch, Popconfirm,
} from 'antd';
import {
  SaveOutlined, ApiOutlined, BulbOutlined, DesktopOutlined, MoonOutlined, SunOutlined,
  PlusOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { useAppStore } from '../../store/appStore';
import { useAuthStore } from '../../store/auth';
import type { ThemeMode } from '../../store/appStore';
import {
  listStorageZones, createStorageZone, deleteStorageZone,
  listOIDs, createOID, deleteOID,
  listRevocationServices, createRevocationService, deleteRevocationService,
  listACMEConfigs, createACMEConfig, deleteACMEConfig,
  listPaymentPlugins, createPaymentPlugin, deletePaymentPlugin,
  listCAs, approveRefund, rejectRefund, listPaymentOrders,
} from '../../api';
import type { StorageZone, CustomOID, RevocationService, ACMEConfig, PaymentPlugin, CA, PaymentOrder } from '../../types';

const { Title, Text } = Typography;
const { Option } = Select;

const THEME_OPTIONS: { label: React.ReactNode; value: ThemeMode }[] = [
  { label: <Space><SunOutlined />亮色</Space>, value: 'light' },
  { label: <Space><MoonOutlined />暗黑</Space>, value: 'dark' },
  { label: <Space><DesktopOutlined />跟随系统</Space>, value: 'system' },
];

// ---- 存储区域 Tab ----
const StorageZoneTab: React.FC = () => {
  const [data, setData] = useState<StorageZone[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();
  const [storageType, setStorageType] = useState('database');

  const load = async () => { setLoading(true); try { setData(await listStorageZones()); } catch {} finally { setLoading(false); } };
  useEffect(() => { load(); }, []);

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '存储类型', dataIndex: 'storage_type', key: 'storage_type', render: (v: string) => <Tag>{v}</Tag> },
    { title: 'HSM 驱动', dataIndex: 'hsm_driver', key: 'hsm_driver', render: (v: string) => v || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', render: (v: string) => <Tag color={v === 'active' ? 'green' : 'default'}>{v}</Tag> },
    {
      title: '操作', key: 'action', render: (_: any, r: StorageZone) => (
        <Popconfirm title="确认删除？" onConfirm={async () => { await deleteStorageZone(r.uuid); load(); }}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>添加存储区域</Button>
      </div>
      <Table rowKey="uuid" columns={columns} dataSource={data} loading={loading} size="small" />
      <Modal title="添加存储区域" open={open} onCancel={() => setOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={async (v) => {
          setSaving(true);
          try { await createStorageZone(v); message.success('创建成功'); setOpen(false); form.resetFields(); load(); }
          catch (e: any) { message.error(e.message); } finally { setSaving(false); }
        }} style={{ marginTop: 16 }}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="存储类型" name="storage_type" rules={[{ required: true }]} initialValue="database">
            <Select onChange={setStorageType}><Option value="database">数据库</Option><Option value="hsm">HSM</Option></Select>
          </Form.Item>
          {storageType === 'hsm' && (
            <Form.Item label="HSM 驱动名称" name="hsm_driver" rules={[{ required: true, message: 'HSM 类型必须填写驱动名称' }]}><Input /></Form.Item>
          )}
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space><Button onClick={() => setOpen(false)}>取消</Button><Button type="primary" htmlType="submit" loading={saving}>创建</Button></Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- OID 管理 Tab ----
const OIDTab: React.FC = () => {
  const [data, setData] = useState<CustomOID[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const load = async () => { setLoading(true); try { setData(await listOIDs()); } catch {} finally { setLoading(false); } };
  useEffect(() => { load(); }, []);

  const columns = [
    { title: 'OID 值', dataIndex: 'oid', key: 'oid', render: (v: string) => <Text code style={{ fontSize: 12 }}>{v}</Text> },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', render: (v: string) => v || '-' },
    { title: '用途类型', dataIndex: 'usage_type', key: 'usage_type', render: (v: string) => <Tag>{v}</Tag> },
    {
      title: '操作', key: 'action', render: (_: any, r: CustomOID) => (
        <Popconfirm title="确认删除？" onConfirm={async () => { await deleteOID(r.uuid); load(); }}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>添加 OID</Button>
      </div>
      <Table rowKey="uuid" columns={columns} dataSource={data} loading={loading} size="small" />
      <Modal title="添加 OID" open={open} onCancel={() => setOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={async (v) => {
          setSaving(true);
          try { await createOID(v); message.success('创建成功'); setOpen(false); form.resetFields(); load(); }
          catch (e: any) { message.error(e.message); } finally { setSaving(false); }
        }} style={{ marginTop: 16 }}>
          <Form.Item label="OID 值" name="oid" rules={[{ required: true }, { pattern: /^\d+(\.\d+)+$/, message: 'OID 格式不正确，如：2.5.4.3' }]}><Input placeholder="2.5.4.3" /></Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="描述" name="description"><Input /></Form.Item>
          <Form.Item label="用途类型" name="usage_type" rules={[{ required: true }]}>
            <Select>
              <Option value="ext_key_usage">扩展密钥用法</Option>
              <Option value="subject_field">主体字段</Option>
              <Option value="ev_policy">EV 策略</Option>
              <Option value="asn1_extension">ASN.1 扩展</Option>
            </Select>
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space><Button onClick={() => setOpen(false)}>取消</Button><Button type="primary" htmlType="submit" loading={saving}>创建</Button></Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- 吊销服务 Tab ----
const RevocationTab: React.FC = () => {
  const [data, setData] = useState<RevocationService[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [cas, setCAs] = useState<CA[]>([]);
  const [form] = Form.useForm();

  const load = async () => { setLoading(true); try { setData(await listRevocationServices()); } catch {} finally { setLoading(false); } };
  useEffect(() => {
    load();
    listCAs({ page: 1, page_size: 100 }).then(r => setCAs(r.items || [])).catch(() => {});
  }, []);

  const columns = [
    { title: '关联 CA', dataIndex: 'ca_name', key: 'ca_name', render: (v: string) => v || '-' },
    { title: '服务类型', dataIndex: 'service_type', key: 'service_type', render: (v: string) => <Tag>{v?.toUpperCase()}</Tag> },
    { title: '路径', dataIndex: 'path', key: 'path', render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text> },
    { title: '启用', dataIndex: 'enabled', key: 'enabled', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
    { title: 'CRL 间隔', dataIndex: 'crl_interval_minutes', key: 'crl_interval', render: (v: number) => v ? `${v} 分钟` : '-' },
    {
      title: '操作', key: 'action', render: (_: any, r: RevocationService) => (
        <Popconfirm title="确认删除？" onConfirm={async () => { await deleteRevocationService(r.uuid); load(); }}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>添加吊销服务</Button>
      </div>
      <Table rowKey="uuid" columns={columns} dataSource={data} loading={loading} size="small" />
      <Modal title="添加吊销服务" open={open} onCancel={() => setOpen(false)} footer={null} width={520}>
        <Form form={form} layout="vertical" onFinish={async (v) => {
          setSaving(true);
          try { await createRevocationService(v); message.success('创建成功'); setOpen(false); form.resetFields(); load(); }
          catch (e: any) { message.error(e.message); } finally { setSaving(false); }
        }} style={{ marginTop: 16 }}>
          <Form.Item label="关联 CA" name="ca_uuid" rules={[{ required: true }]}>
            <Select>{cas.map(c => <Option key={c.uuid} value={c.uuid}>{c.name}</Option>)}</Select>
          </Form.Item>
          <Form.Item label="服务类型" name="service_type" rules={[{ required: true }]}>
            <Select><Option value="crl">CRL</Option><Option value="ocsp">OCSP</Option><Option value="caissuer">CA Issuer</Option></Select>
          </Form.Item>
          <Form.Item label="服务路径" name="path" rules={[{ required: true }]}><Input placeholder="/crl/ca-uuid" /></Form.Item>
          <Form.Item label="是否启用" name="enabled" valuePropName="checked" initialValue={true}><Switch /></Form.Item>
          <Form.Item label="CRL 更新间隔（分钟）" name="crl_interval_minutes" initialValue={60}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space><Button onClick={() => setOpen(false)}>取消</Button><Button type="primary" htmlType="submit" loading={saving}>创建</Button></Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- ACME 配置 Tab ----
const ACMETab: React.FC = () => {
  const [data, setData] = useState<ACMEConfig[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [cas, setCAs] = useState<CA[]>([]);
  const [form] = Form.useForm();

  const load = async () => { setLoading(true); try { setData(await listACMEConfigs()); } catch {} finally { setLoading(false); } };
  useEffect(() => {
    load();
    listCAs({ page: 1, page_size: 100 }).then(r => setCAs(r.items || [])).catch(() => {});
  }, []);

  const columns = [
    { title: '路径', dataIndex: 'path', key: 'path', render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text> },
    { title: '关联 CA', dataIndex: 'ca_name', key: 'ca_name', render: (v: string) => v || '-' },
    { title: '关联模板', dataIndex: 'template_name', key: 'template_name', render: (v: string) => v || '-' },
    { title: '启用', dataIndex: 'enabled', key: 'enabled', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
    {
      title: '操作', key: 'action', render: (_: any, r: ACMEConfig) => (
        <Popconfirm title="确认删除？" onConfirm={async () => { await deleteACMEConfig(r.uuid); load(); }}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>添加 ACME 配置</Button>
      </div>
      <Table rowKey="uuid" columns={columns} dataSource={data} loading={loading} size="small" />
      <Modal title="添加 ACME 配置" open={open} onCancel={() => setOpen(false)} footer={null}>
        <Form form={form} layout="vertical" onFinish={async (v) => {
          setSaving(true);
          try { await createACMEConfig(v); message.success('创建成功'); setOpen(false); form.resetFields(); load(); }
          catch (e: any) { message.error(e.message); } finally { setSaving(false); }
        }} style={{ marginTop: 16 }}>
          <Form.Item label="路径" name="path" rules={[{ required: true }]}><Input placeholder="/acme/default" /></Form.Item>
          <Form.Item label="关联 CA" name="ca_uuid" rules={[{ required: true }]}>
            <Select>{cas.map(c => <Option key={c.uuid} value={c.uuid}>{c.name}</Option>)}</Select>
          </Form.Item>
          <Form.Item label="关联颁发模板 UUID" name="template_uuid"><Input placeholder="可选" /></Form.Item>
          <Form.Item label="是否启用" name="enabled" valuePropName="checked" initialValue={true}><Switch /></Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space><Button onClick={() => setOpen(false)}>取消</Button><Button type="primary" htmlType="submit" loading={saving}>创建</Button></Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- 支付插件 Tab ----
const PaymentPluginTab: React.FC = () => {
  const [plugins, setPlugins] = useState<PaymentPlugin[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pendingRefunds, setPendingRefunds] = useState<PaymentOrder[]>([]);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      setPlugins(await listPaymentPlugins());
      const res = await listPaymentOrders({ page: 1, page_size: 20, status: 'refunding' });
      setPendingRefunds(res.items || []);
    } catch {} finally { setLoading(false); }
  };
  useEffect(() => { load(); }, []);

  const pluginColumns = [
    { title: '插件名称', dataIndex: 'name', key: 'name' },
    { title: '插件类型', dataIndex: 'plugin_type', key: 'plugin_type', render: (v: string) => <Tag>{v}</Tag> },
    { title: '启用', dataIndex: 'enabled', key: 'enabled', render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '启用' : '禁用'}</Tag> },
    { title: '排序权重', dataIndex: 'sort_weight', key: 'sort_weight' },
    {
      title: '操作', key: 'action', render: (_: any, r: PaymentPlugin) => (
        <Popconfirm title="确认删除？" onConfirm={async () => { await deletePaymentPlugin(r.uuid); load(); }}>
          <Button size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  const refundColumns = [
    { title: '订单号', dataIndex: 'uuid', key: 'uuid', render: (v: string) => <Text code style={{ fontSize: 11 }}>{v.slice(0, 16)}...</Text> },
    { title: '金额', dataIndex: 'amount', key: 'amount', render: (v: number) => `¥${(v / 100).toFixed(2)}` },
    { title: '渠道', dataIndex: 'channel', key: 'channel' },
    {
      title: '操作', key: 'action', render: (_: any, r: PaymentOrder) => (
        <Space size={4}>
          <Button size="small" type="primary" onClick={async () => { await approveRefund(r.uuid); load(); }}>通过</Button>
          <Button size="small" danger onClick={async () => { await rejectRefund(r.uuid); load(); }}>拒绝</Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>添加支付插件</Button>
      </div>
      <Table rowKey="uuid" columns={pluginColumns} dataSource={plugins} loading={loading} size="small" style={{ marginBottom: 24 }} />

      {pendingRefunds.length > 0 && (
        <>
          <Title level={5} style={{ margin: '16px 0 8px' }}>待审批退款</Title>
          <Table rowKey="uuid" columns={refundColumns} dataSource={pendingRefunds} size="small" pagination={false} />
        </>
      )}

      <Modal title="添加支付插件" open={open} onCancel={() => setOpen(false)} footer={null} width={520}>
        <Form form={form} layout="vertical" onFinish={async (v) => {
          setSaving(true);
          try { await createPaymentPlugin(v); message.success('创建成功'); setOpen(false); form.resetFields(); load(); }
          catch (e: any) { message.error(e.message); } finally { setSaving(false); }
        }} style={{ marginTop: 16 }}>
          <Form.Item label="插件名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="插件类型" name="plugin_type" rules={[{ required: true }]}>
            <Select><Option value="alipay">支付宝</Option><Option value="wechat">微信支付</Option><Option value="stripe">Stripe</Option><Option value="custom">自定义</Option></Select>
          </Form.Item>
          <Form.Item label="是否启用" name="enabled" valuePropName="checked" initialValue={true}><Switch /></Form.Item>
          <Form.Item label="排序权重" name="sort_weight" initialValue={0}><InputNumber style={{ width: '100%' }} /></Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space><Button onClick={() => setOpen(false)}>取消</Button><Button type="primary" htmlType="submit" loading={saving}>创建</Button></Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- 主页面 ----
const Settings: React.FC = () => {
  const { darkMode, themeMode, setThemeMode, connected, serverVersion, checkConnection } = useAppStore();
  const { role } = useAuthStore();
  const [apiBase, setApiBase] = useState(localStorage.getItem('apiBase') || 'http://localhost:1026');
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
      if (connected) message.success(`连接成功，服务版本 v${serverVersion}`);
      else message.error('连接失败，请检查服务是否已启动');
    } finally { setTesting(false); }
  };

  const baseTabs = [
    {
      key: 'connection',
      label: '服务连接',
      children: (
        <div style={{ maxWidth: 560 }}>
          <Alert
            message={connected ? `已连接 v${serverVersion}` : '未连接到服务'}
            type={connected ? 'success' : 'warning'} showIcon style={{ marginBottom: 16 }}
          />
          <Form layout="vertical">
            <Form.Item label="client-card 服务地址" extra="修改后需刷新页面生效">
              <Space.Compact style={{ width: '100%' }}>
                <Input value={apiBase} onChange={(e) => setApiBase(e.target.value)} placeholder="http://localhost:1026" />
                <Button onClick={handleSaveApiBase} icon={<SaveOutlined />}>保存</Button>
              </Space.Compact>
            </Form.Item>
            <Button onClick={handleTestConnection} loading={testing} icon={<ApiOutlined />}>测试连接</Button>
          </Form>
        </div>
      ),
    },
    {
      key: 'theme',
      label: '外观',
      children: (
        <div style={{ maxWidth: 480 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
            <div>
              <Text strong>界面主题</Text>
              <br />
              <Text style={{ fontSize: 12, color: '#999' }}>选择亮色、暗黑或跟随系统自动切换</Text>
            </div>
            <Segmented<ThemeMode> value={themeMode} onChange={setThemeMode} options={THEME_OPTIONS} size="middle" />
          </div>
        </div>
      ),
    },
    {
      key: 'about',
      label: '关于',
      children: (
        <div style={{ maxWidth: 480 }}>
          <Space direction="vertical" style={{ width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}><Text type="secondary">前端版本</Text><Tag color="blue">v1.0.0</Tag></div>
            <Divider style={{ margin: '8px 0' }} />
            <div style={{ display: 'flex', justifyContent: 'space-between' }}><Text type="secondary">服务版本</Text><Tag color={connected ? 'success' : 'default'}>{connected ? `v${serverVersion}` : '未连接'}</Tag></div>
            <Divider style={{ margin: '8px 0' }} />
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary">技术栈</Text>
              <Space><Tag>React 19</Tag><Tag>Ant Design 6</Tag><Tag>Vite</Tag></Space>
            </div>
          </Space>
        </div>
      ),
    },
  ];

  const adminTabs = role === 'admin' ? [
    { key: 'storage_zones', label: '存储区域', children: <StorageZoneTab /> },
    { key: 'oids', label: 'OID 管理', children: <OIDTab /> },
    { key: 'revocation', label: '吊销服务', children: <RevocationTab /> },
    { key: 'acme', label: 'ACME 配置', children: <ACMETab /> },
    { key: 'payment_plugins', label: '支付插件', children: <PaymentPluginTab /> },
  ] : [];

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px' }}>系统设置</Title>
      <Tabs items={[...baseTabs, ...adminTabs]} />
    </div>
  );
};

export default Settings;
