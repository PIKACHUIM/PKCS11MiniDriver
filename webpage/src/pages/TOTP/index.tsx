import React, { useEffect, useState, useRef, useCallback } from 'react';
import {
  Card, Table, Button, Space, Tag, Modal, Form, Input, Select,
  message, Progress, Typography, Tooltip, InputNumber, Row, Col, Tabs,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, CopyOutlined, ClockCircleOutlined,
  ReloadOutlined, QrcodeOutlined, KeyOutlined, CloudOutlined,
} from '@ant-design/icons';
import { getTOTPList, getTOTPCode, createTOTP, deleteTOTP, listCloudTOTPs, createCloudTOTP, getCloudTOTPCode, deleteCloudTOTP } from '../../api';
import type { TOTPEntry, CloudTOTPEntry } from '../../types';

const { Text } = Typography;

interface TOTPWithCode extends TOTPEntry {
  code?: string;
  remaining?: number;
}

interface CloudTOTPWithCode extends CloudTOTPEntry {
  code?: string;
  remaining?: number;
}

// ---- 本地 TOTP Tab ----
const LocalTOTPTab: React.FC = () => {
  const [entries, setEntries] = useState<TOTPWithCode[]>([]);
  const [loading, setLoading] = useState(false);
  const [addVisible, setAddVisible] = useState(false);
  const [form] = Form.useForm();
  const timerRef = useRef<ReturnType<typeof setInterval>>();

  const loadEntries = async () => {
    setLoading(true);
    try {
      const data = await getTOTPList();
      setEntries((data || []).map((e: TOTPEntry) => ({ ...e })));
    } catch (err: any) {
      message.error(err.message || '加载 TOTP 列表失败');
    } finally {
      setLoading(false);
    }
  };

  const refreshCodes = useCallback(async () => {
    setEntries(prev => prev.map(e => {
      const period = e.period || 30;
      const now = Math.floor(Date.now() / 1000);
      const remaining = period - (now % period);
      return { ...e, remaining };
    }));
    for (const entry of entries) {
      try {
        const resp = await getTOTPCode(entry.uuid);
        setEntries(prev => prev.map(e => e.uuid === entry.uuid ? { ...e, code: resp.code } : e));
      } catch {}
    }
  }, [entries]);

  useEffect(() => { loadEntries(); }, []);

  useEffect(() => {
    timerRef.current = setInterval(() => {
      const now = Math.floor(Date.now() / 1000);
      setEntries(prev => prev.map(e => {
        const period = e.period || 30;
        const remaining = period - (now % period);
        if (remaining === period) {
          getTOTPCode(e.uuid).then(resp => {
            setEntries(p => p.map(x => x.uuid === e.uuid ? { ...x, code: resp.code, remaining } : x));
          }).catch(() => {});
        }
        return { ...e, remaining };
      }));
    }, 1000);
    return () => clearInterval(timerRef.current);
  }, []);

  useEffect(() => {
    if (entries.length > 0 && !entries[0].code) refreshCodes();
  }, [entries.length]);

  const handleAdd = async () => {
    try {
      const values = await form.validateFields();
      await createTOTP(values);
      message.success('TOTP 条目已添加');
      setAddVisible(false);
      form.resetFields();
      loadEntries();
    } catch (err: any) {
      if (err.message) message.error(err.message);
    }
  };

  const handleDelete = (entry: TOTPWithCode) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除 ${entry.issuer}:${entry.account} 的 TOTP 条目吗？`,
      okType: 'danger', okText: '确认删除',
      onOk: async () => {
        try { await deleteTOTP(entry.uuid); message.success('已删除'); loadEntries(); }
        catch (err: any) { message.error(err.message || '删除失败'); }
      },
    });
  };

  const columns = [
    { title: '发行者', dataIndex: 'issuer', width: 160, render: (v: string) => <Text strong>{v || '-'}</Text> },
    { title: '账户', dataIndex: 'account', width: 200, ellipsis: true },
    {
      title: '验证码', width: 200,
      render: (_: unknown, record: TOTPWithCode) => {
        const code = record.code || '------';
        const period = record.period || 30;
        const remaining = record.remaining || 0;
        const percent = (remaining / period) * 100;
        const isUrgent = remaining <= 5;
        return (
          <Space>
            <Tooltip title="点击复制">
              <Button type="text" size="large"
                style={{ fontFamily: 'monospace', fontSize: 22, fontWeight: 700, letterSpacing: 4, color: isUrgent ? '#ff4d4f' : '#1677ff' }}
                icon={<CopyOutlined style={{ fontSize: 14 }} />}
                onClick={() => { if (record.code) { navigator.clipboard.writeText(record.code); message.success('已复制'); } }}
              >{code}</Button>
            </Tooltip>
            <Progress type="circle" percent={percent} size={28} format={() => `${remaining}`} strokeColor={isUrgent ? '#ff4d4f' : '#1677ff'} />
          </Space>
        );
      },
    },
    { title: '算法', dataIndex: 'algorithm', width: 80, render: (v: string) => <Tag>{v || 'SHA1'}</Tag> },
    { title: '位数', dataIndex: 'digits', width: 60, render: (v: number) => v || 6 },
    {
      title: '操作', width: 80,
      render: (_: unknown, record: TOTPWithCode) => (
        <Tooltip title="删除"><Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} /></Tooltip>
      ),
    },
  ];

  return (
    <Card
      title={<Space><ClockCircleOutlined /><span>本地 TOTP</span><Tag color="blue">{entries.length} 个条目</Tag></Space>}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={refreshCodes}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAddVisible(true)}>添加 TOTP</Button>
        </Space>
      }
    >
      <Table rowKey="uuid" columns={columns} dataSource={entries} loading={loading} pagination={false} />
      <Modal title="添加 TOTP 条目" open={addVisible} onOk={handleAdd}
        onCancel={() => { setAddVisible(false); form.resetFields(); }} width={520}>
        <Form form={form} layout="vertical" initialValues={{ algorithm: 'SHA1', digits: 6, period: 30 }}>
          <Form.Item name="issuer" label="发行者" rules={[{ required: true }]}><Input placeholder="GitHub、Google" prefix={<KeyOutlined />} /></Form.Item>
          <Form.Item name="account" label="账户名" rules={[{ required: true }]}><Input placeholder="user@example.com" /></Form.Item>
          <Form.Item name="secret" label="密钥 (Base32)" rules={[{ required: true }]}><Input.TextArea placeholder="JBSWY3DPEHPK3PXP..." rows={2} style={{ fontFamily: 'monospace' }} /></Form.Item>
          <Form.Item name="uri" label="或粘贴 otpauth:// URI（可选）"><Input placeholder="otpauth://totp/..." prefix={<QrcodeOutlined />} /></Form.Item>
          <Row gutter={16}>
            <Col span={8}><Form.Item name="algorithm" label="算法"><Select options={[{ value: 'SHA1', label: 'SHA1' }, { value: 'SHA256', label: 'SHA256' }, { value: 'SHA512', label: 'SHA512' }]} /></Form.Item></Col>
            <Col span={8}><Form.Item name="digits" label="位数"><Select options={[{ value: 6, label: '6 位' }, { value: 8, label: '8 位' }]} /></Form.Item></Col>
            <Col span={8}><Form.Item name="period" label="周期（秒）"><InputNumber min={15} max={120} style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Card>
  );
};

// ---- 云端 TOTP Tab ----
const CloudTOTPTab: React.FC = () => {
  const [entries, setEntries] = useState<CloudTOTPWithCode[]>([]);
  const [loading, setLoading] = useState(false);
  const [addVisible, setAddVisible] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try { setEntries(await listCloudTOTPs()); }
    catch (err: any) { message.error(err.message || '加载失败'); }
    finally { setLoading(false); }
  };

  useEffect(() => { load(); }, []);

  const handleGetCode = async (entry: CloudTOTPWithCode) => {
    try {
      const resp = await getCloudTOTPCode(entry.uuid);
      setEntries(prev => prev.map(e => e.uuid === entry.uuid ? { ...e, code: resp.code, remaining: resp.remaining } : e));
    } catch (err: any) { message.error(err.message || '获取验证码失败'); }
  };

  const handleAdd = async () => {
    try {
      const values = await form.validateFields();
      await createCloudTOTP(values);
      message.success('云端 TOTP 已添加');
      setAddVisible(false); form.resetFields(); load();
    } catch (err: any) { if (err.message) message.error(err.message); }
  };

  const handleDelete = (entry: CloudTOTPWithCode) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除云端 TOTP「${entry.issuer}:${entry.account}」吗？`,
      okType: 'danger', okText: '确认删除',
      onOk: async () => {
        try { await deleteCloudTOTP(entry.uuid); message.success('已删除'); load(); }
        catch (err: any) { message.error(err.message || '删除失败'); }
      },
    });
  };

  const columns = [
    { title: '发行者', dataIndex: 'issuer', width: 160, render: (v: string) => <Text strong>{v || '-'}</Text> },
    { title: '账户', dataIndex: 'account', width: 200, ellipsis: true },
    { title: '算法', dataIndex: 'algorithm', width: 80, render: (v: string) => <Tag>{v || 'SHA1'}</Tag> },
    { title: '位数', dataIndex: 'digits', width: 60, render: (v: number) => v || 6 },
    { title: '周期', dataIndex: 'period', width: 70, render: (v: number) => `${v || 30}s` },
    {
      title: '验证码', width: 220,
      render: (_: unknown, record: CloudTOTPWithCode) => (
        record.code ? (
          <Space>
            <Text style={{ fontFamily: 'monospace', fontSize: 20, fontWeight: 700, color: '#1677ff', letterSpacing: 4 }}>{record.code}</Text>
            {record.remaining !== undefined && (
              <Progress type="circle" percent={((record.remaining) / (record.period || 30)) * 100} size={28} format={() => `${record.remaining}`} strokeColor="#1677ff" />
            )}
            <Button size="small" icon={<CopyOutlined />} onClick={() => { navigator.clipboard.writeText(record.code!); message.success('已复制'); }} />
          </Space>
        ) : (
          <Button size="small" type="primary" onClick={() => handleGetCode(record)}>获取验证码</Button>
        )
      ),
    },
    {
      title: '操作', width: 80,
      render: (_: unknown, record: CloudTOTPWithCode) => (
        <Tooltip title="删除"><Button type="text" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)} /></Tooltip>
      ),
    },
  ];

  return (
    <Card
      title={<Space><CloudOutlined /><span>云端 TOTP</span><Tag color="purple">{entries.length} 个条目</Tag></Space>}
      extra={<Button type="primary" icon={<PlusOutlined />} onClick={() => setAddVisible(true)}>添加云端 TOTP</Button>}
    >
      <Table rowKey="uuid" columns={columns} dataSource={entries} loading={loading} pagination={false} />
      <Modal title="添加云端 TOTP" open={addVisible} onOk={handleAdd}
        onCancel={() => { setAddVisible(false); form.resetFields(); }} width={520}>
        <Form form={form} layout="vertical" initialValues={{ algorithm: 'SHA1', digits: 6, period: 30 }}>
          <Form.Item name="issuer" label="发行者" rules={[{ required: true }]}><Input placeholder="GitHub、Google" /></Form.Item>
          <Form.Item name="account" label="账户名" rules={[{ required: true }]}><Input placeholder="user@example.com" /></Form.Item>
          <Form.Item name="secret" label="密钥 (Base32)" rules={[{ required: true }]}><Input.TextArea placeholder="JBSWY3DPEHPK3PXP..." rows={2} style={{ fontFamily: 'monospace' }} /></Form.Item>
          <Row gutter={16}>
            <Col span={8}><Form.Item name="algorithm" label="算法"><Select options={[{ value: 'SHA1', label: 'SHA1' }, { value: 'SHA256', label: 'SHA256' }, { value: 'SHA512', label: 'SHA512' }]} /></Form.Item></Col>
            <Col span={8}><Form.Item name="digits" label="位数"><Select options={[{ value: 6, label: '6 位' }, { value: 8, label: '8 位' }]} /></Form.Item></Col>
            <Col span={8}><Form.Item name="period" label="周期（秒）"><InputNumber min={15} max={120} style={{ width: '100%' }} /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>
    </Card>
  );
};

// ---- 主页面 ----
const TOTPPage: React.FC = () => (
  <div>
    <Tabs items={[
      { key: 'local', label: <Space><ClockCircleOutlined />本地 TOTP</Space>, children: <LocalTOTPTab /> },
      { key: 'cloud', label: <Space><CloudOutlined />云端 TOTP</Space>, children: <CloudTOTPTab /> },
    ]} />
  </div>
);

export default TOTPPage;