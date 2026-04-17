import React, { useEffect, useState } from 'react';
import {
  Table, Button, Space, Tag, Typography, Modal, Form, Input, Select,
  Popconfirm, message, Tooltip, Card, InputNumber, Row, Col, Divider,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, ReloadOutlined, BankOutlined,
  ImportOutlined, DownloadOutlined, StopOutlined,
} from '@ant-design/icons';
import { getLocalCAs, createLocalCA, importLocalCA, revokeLocalCA, deleteLocalCA, exportLocalCA, getCards } from '../../api';
import type { LocalCA, CreateCARequest, ImportCARequest, Card as CardType } from '../../types';
import { useAppStore } from '../../store/appStore';
import dayjs from 'dayjs';

const { Text } = Typography;
const { TextArea } = Input;

const KEY_TYPE_OPTIONS = [
  { label: 'RSA 2048', value: 'rsa2048' },
  { label: 'RSA 4096', value: 'rsa4096' },
  { label: 'EC P-256（推荐）', value: 'ec256' },
  { label: 'EC P-384', value: 'ec384' },
  { label: 'EC P-521', value: 'ec521' },
  { label: 'Ed25519', value: 'ed25519' },
  { label: 'SM2', value: 'sm2' },
];

const LocalCAPage: React.FC = () => {
  const { darkMode } = useAppStore();
  const [list, setList] = useState<LocalCA[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [cards, setCards] = useState<CardType[]>([]);

  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createForm] = Form.useForm();

  const [importOpen, setImportOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importForm] = Form.useForm();

  const load = async (p = page) => {
    setLoading(true);
    try {
      const res = await getLocalCAs({ page: p, page_size: 10 });
      setList(res.items);
      setTotal(res.total);
    } catch (e: any) { message.error(e.message); }
    finally { setLoading(false); }
  };

  useEffect(() => {
    load();
    getCards({ page: 1, page_size: 100 }).then((r) => setCards(r.items)).catch(() => {});
  }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setCreating(true);
      await createLocalCA(values as CreateCARequest);
      message.success('CA 已创建');
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e: any) { if (e.message) message.error(e.message); }
    finally { setCreating(false); }
  };

  const handleImport = async () => {
    try {
      const values = await importForm.validateFields();
      setImporting(true);
      await importLocalCA(values as ImportCARequest);
      message.success('CA 已导入');
      setImportOpen(false);
      importForm.resetFields();
      load();
    } catch (e: any) { if (e.message) message.error(e.message); }
    finally { setImporting(false); }
  };

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  const columns = [
    {
      title: 'CA 名称',
      dataIndex: 'name',
      render: (v: string) => <Text strong style={{ color: darkMode ? '#c9d1d9' : undefined }}>{v}</Text>,
    },
    {
      title: '密钥类型',
      dataIndex: 'key_type',
      width: 100,
      render: (v: string) => <Tag color="blue">{v?.toUpperCase()}</Tag>,
    },
    {
      title: '有效期',
      width: 220,
      render: (_: any, r: LocalCA) => (
        <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#666' }}>
          {dayjs(r.not_before).format('YYYY-MM-DD')} ~ {dayjs(r.not_after).format('YYYY-MM-DD')}
        </Text>
      ),
    },
    {
      title: '已签发',
      dataIndex: 'issued_count',
      width: 80,
      render: (v: number) => <Text>{v ?? 0}</Text>,
    },
    {
      title: '私钥',
      dataIndex: 'has_priv_key',
      width: 60,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '有' : '无'}</Tag>,
    },
    {
      title: '状态',
      width: 80,
      render: (_: any, r: LocalCA) => (
        <Tag color={r.revoked ? 'red' : 'green'}>{r.revoked ? '已吊销' : '有效'}</Tag>
      ),
    },
    {
      title: '操作',
      width: 170,
      render: (_: any, record: LocalCA) => (
        <Space>
          <Tooltip title="导出证书">
            <Button type="text" size="small" icon={<DownloadOutlined />}
              onClick={() => exportLocalCA(record.uuid, 'pem', record.name).catch((e) => message.error(e.message))} />
          </Tooltip>
          <Tooltip title="导出证书链">
            <Button type="text" size="small" icon={<DownloadOutlined />}
              onClick={() => exportLocalCA(record.uuid, 'chain', record.name).catch((e) => message.error(e.message))}>
              链
            </Button>
          </Tooltip>
          {!record.revoked && (
            <Popconfirm title="确认吊销此 CA？吊销后无法签发新证书。"
              onConfirm={() => revokeLocalCA(record.uuid).then(() => { message.success('已吊销'); load(); }).catch((e) => message.error(e.message))}
              okText="吊销" cancelText="取消" okButtonProps={{ danger: true }}>
              <Tooltip title="吊销">
                <Button type="text" size="small" danger icon={<StopOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
          <Popconfirm title="确认删除此 CA？"
            onConfirm={() => deleteLocalCA(record.uuid).then(() => { message.success('已删除'); load(); }).catch((e) => message.error(e.message))}
            okText="删除" cancelText="取消" okButtonProps={{ danger: true }}>
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 16, color: darkMode ? '#c9d1d9' : undefined }}>
          <BankOutlined style={{ marginRight: 8 }} />CA 机构管理
        </Text>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => load()}>刷新</Button>
          <Button icon={<ImportOutlined />} onClick={() => setImportOpen(true)}>导入 CA</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>创建 CA</Button>
        </Space>
      </div>

      <Card style={cardStyle} bodyStyle={{ padding: 0 }}>
        <Table dataSource={list} columns={columns} rowKey="uuid" loading={loading}
          pagination={{ current: page, total, pageSize: 10, onChange: (p) => { setPage(p); load(p); }, showTotal: (t) => `共 ${t} 条` }} />
      </Card>

      {/* 创建 CA 弹窗 */}
      <Modal title={<Space><BankOutlined />创建本地 CA</Space>} open={createOpen}
        onOk={handleCreate} onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        okText="创建" cancelText="取消" confirmLoading={creating} width={560}>
        <Form form={createForm} layout="vertical" style={{ marginTop: 16 }}
          initialValues={{ key_type: 'ec256', validity_years: 10 }}>
          <Form.Item name="name" label="CA 名称" rules={[{ required: true }]}>
            <Input placeholder="My Root CA" />
          </Form.Item>
          <Divider titlePlacement="left" style={{ fontSize: 13 }}>主体信息</Divider>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="common_name" label="通用名称 (CN)" rules={[{ required: true }]}>
                <Input placeholder="My Root CA" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="organization" label="组织 (O)">
                <Input placeholder="My Organization" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="country" label="国家 (C)">
                <Input placeholder="CN" maxLength={2} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="key_type" label="密钥类型">
                <Select options={KEY_TYPE_OPTIONS} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="validity_years" label="有效期（年）">
                <InputNumber min={1} max={30} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="card_uuid" label="存储智能卡（可选）">
            <Select allowClear placeholder="选择存储密钥的智能卡（留空则存数据库）"
              options={cards.map((c) => ({ value: c.uuid, label: `${c.card_name} (${c.slot_type})` }))} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 导入 CA 弹窗 */}
      <Modal title={<Space><ImportOutlined />导入 CA 证书</Space>} open={importOpen}
        onOk={handleImport} onCancel={() => { setImportOpen(false); importForm.resetFields(); }}
        okText="导入" cancelText="取消" confirmLoading={importing} width={600}>
        <Form form={importForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="name" label="CA 名称" rules={[{ required: true }]}>
            <Input placeholder="导入的 CA 名称" />
          </Form.Item>
          <Form.Item name="cert_pem" label="CA 证书（PEM 格式）" rules={[{ required: true }]}>
            <TextArea rows={6} placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
              style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
          <Form.Item name="key_pem" label="CA 私钥（PEM 格式，可选）">
            <TextArea rows={5} placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;（有私钥才能用此 CA 签发证书）"
              style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
          <Form.Item name="chain_pem" label="证书链（PEM 格式，可选）">
            <TextArea rows={4} placeholder="中间 CA 证书链（可选）"
              style={{ fontFamily: 'monospace', fontSize: 12 }} />
          </Form.Item>
          <Form.Item name="card_uuid" label="私钥存储智能卡（可选）">
            <Select allowClear placeholder="若私钥需存储到智能卡，请选择"
              options={cards.map((c) => ({ value: c.uuid, label: `${c.card_name} (${c.slot_type})` }))} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default LocalCAPage;
