import React, { useEffect, useState } from 'react';
import {
  Table, Button, Space, Tag, Typography, Modal, Form, Input,
  Select, Popconfirm, message, Tooltip, Card, Drawer, Descriptions,
  Empty,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, ReloadOutlined,
  KeyOutlined, SafetyCertificateOutlined, EyeOutlined,
} from '@ant-design/icons';
import { getCards, createCard, deleteCard, getCerts, generateKey, deleteCert, getUsers } from '../../api';
import type { Card as CardType, Certificate, User, CreateCardRequest } from '../../types';
import { useAppStore } from '../../store/appStore';
import dayjs from 'dayjs';

const { Title, Text } = Typography;

// 密钥类型选项
const KEY_TYPE_OPTIONS = [
  { value: 'ec256', label: 'EC P-256（推荐）' },
  { value: 'ec384', label: 'EC P-384' },
  { value: 'ec521', label: 'EC P-521' },
  { value: 'rsa2048', label: 'RSA 2048' },
  { value: 'rsa4096', label: 'RSA 4096' },
];

// Slot 类型颜色
const slotColor = (t: string) =>
  t === 'cloud' ? 'purple' : t === 'tpm2' ? 'cyan' : 'green';

const Cards: React.FC = () => {
  const { darkMode } = useAppStore();
  const [cards, setCards] = useState<CardType[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [users, setUsers] = useState<User[]>([]);

  // 新建卡片弹窗
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm] = Form.useForm();
  const [creating, setCreating] = useState(false);

  // 证书抽屉
  const [certDrawerOpen, setCertDrawerOpen] = useState(false);
  const [selectedCard, setSelectedCard] = useState<CardType | null>(null);
  const [certs, setCerts] = useState<Certificate[]>([]);
  const [certsLoading, setCertsLoading] = useState(false);

  // 密钥生成弹窗
  const [keygenOpen, setKeygenOpen] = useState(false);
  const [keygenForm] = Form.useForm();
  const [keygening, setKeygening] = useState(false);

  const load = async (p = page) => {
    setLoading(true);
    try {
      const res = await getCards({ page: p, page_size: 10 });
      setCards(res?.items ?? []);
      setTotal(res?.total ?? 0);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setLoading(false);
    }
  };

  const loadUsers = async () => {
    try {
      const res = await getUsers({ page: 1, page_size: 100 });
      setUsers(res?.items ?? []);
    } catch {}
  };

  const loadCerts = async (cardUUID: string) => {
    setCertsLoading(true);
    try {
      const res = await getCerts(cardUUID);
      setCerts(Array.isArray(res) ? res : []);
    } catch (e: any) {
      message.error(e.message);
    } finally {
      setCertsLoading(false);
    }
  };

  useEffect(() => { load(); loadUsers(); }, []);

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setCreating(true);
      await createCard(values as CreateCardRequest);
      message.success('卡片已创建');
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e: any) {
      if (e.message) message.error(e.message);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (uuid: string) => {
    try {
      await deleteCard(uuid);
      message.success('卡片已删除');
      load();
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const openCerts = (card: CardType) => {
    setSelectedCard(card);
    setCertDrawerOpen(true);
    loadCerts(card.uuid);
  };

  const handleKeygen = async () => {
    if (!selectedCard) return;
    try {
      const values = await keygenForm.validateFields();
      setKeygening(true);
      await generateKey(selectedCard.uuid, values);
      message.success('密钥对已生成');
      setKeygenOpen(false);
      keygenForm.resetFields();
      loadCerts(selectedCard.uuid);
    } catch (e: any) {
      if (e.message) message.error(e.message);
    } finally {
      setKeygening(false);
    }
  };

  const handleDeleteCert = async (certUUID: string) => {
    if (!selectedCard) return;
    try {
      await deleteCert(selectedCard.uuid, certUUID);
      message.success('证书已删除');
      loadCerts(selectedCard.uuid);
    } catch (e: any) {
      message.error(e.message);
    }
  };

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  const columns = [
    {
      title: '卡片名称',
      dataIndex: 'card_name',
      render: (v: string, record: CardType) => (
        <Space>
          <SafetyCertificateOutlined style={{ color: slotColor(record.slot_type) === 'purple' ? '#722ed1' : slotColor(record.slot_type) === 'cyan' ? '#13c2c2' : '#52c41a' }} />
          <Text strong style={{ color: darkMode ? '#c9d1d9' : undefined }}>{v}</Text>
        </Space>
      ),
    },
    {
      title: 'Slot 类型',
      dataIndex: 'slot_type',
      width: 100,
      render: (v: string) => (
        <Tag color={slotColor(v)}>{v?.toUpperCase() || 'LOCAL'}</Tag>
      ),
    },
    {
      title: '所属用户',
      dataIndex: 'user_uuid',
      render: (v: string) => {
        const user = users.find((u) => u.uuid === v);
        return <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>{user?.display_name || v?.slice(0, 8) + '...'}</Text>;
      },
    },
    {
      title: '备注',
      dataIndex: 'remark',
      render: (v: string) => <Text style={{ color: darkMode ? '#8b949e' : '#999', fontSize: 12 }}>{v || '-'}</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (v: string) => (
        <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#999' }}>
          {dayjs(v).format('YYYY-MM-DD HH:mm')}
        </Text>
      ),
    },
    {
      title: '操作',
      width: 140,
      render: (_: any, record: CardType) => (
        <Space>
          <Tooltip title="查看证书">
            <Button type="text" size="small" icon={<EyeOutlined />} onClick={() => openCerts(record)} />
          </Tooltip>
          <Tooltip title="生成密钥">
            <Button
              type="text" size="small" icon={<KeyOutlined />}
              onClick={() => { setSelectedCard(record); setKeygenOpen(true); }}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除此卡片？此操作将同时删除所有证书。"
            onConfirm={() => handleDelete(record.uuid)}
            okText="删除" cancelText="取消" okButtonProps={{ danger: true }}
          >
            <Tooltip title="删除">
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const certColumns = [
    {
      title: '密钥类型',
      dataIndex: 'key_type',
      width: 100,
      render: (v: string) => <Tag color="blue">{v?.toUpperCase()}</Tag>,
    },
    {
      title: '证书类型',
      dataIndex: 'cert_type',
      width: 80,
      render: (v: string) => <Tag>{v || 'x509'}</Tag>,
    },
    {
      title: '备注',
      dataIndex: 'remark',
      render: (v: string) => <Text style={{ fontSize: 12 }}>{v || '-'}</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 150,
      render: (v: string) => (
        <Text style={{ fontSize: 12, color: '#999' }}>{dayjs(v).format('MM-DD HH:mm:ss')}</Text>
      ),
    },
    {
      title: '操作',
      width: 80,
      render: (_: any, record: Certificate) => (
        <Popconfirm
          title="确认删除此证书？"
          onConfirm={() => handleDeleteCert(record.uuid)}
          okText="删除" cancelText="取消" okButtonProps={{ danger: true }}
        >
          <Button type="text" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0, color: darkMode ? '#c9d1d9' : undefined }}>
          卡片管理
        </Title>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => load()}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建卡片
          </Button>
        </Space>
      </div>

      <Card style={cardStyle} bodyStyle={{ padding: 0 }}>
        <Table
          dataSource={cards}
          columns={columns}
          rowKey="uuid"
          loading={loading}
          pagination={{
            current: page,
            total,
            pageSize: 10,
            onChange: (p) => { setPage(p); load(p); },
            showTotal: (t) => `共 ${t} 条`,
          }}
        />
      </Card>

      {/* 新建卡片弹窗 */}
      <Modal
        title="新建卡片"
        open={createOpen}
        onOk={handleCreate}
        onCancel={() => { setCreateOpen(false); createForm.resetFields(); }}
        okText="创建" cancelText="取消"
        confirmLoading={creating}
        width={480}
      >
        <Form form={createForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="card_name" label="卡片名称" rules={[{ required: true, message: '请输入卡片名称' }]}>
            <Input placeholder="例如：我的工作证书" />
          </Form.Item>
          <Form.Item name="slot_type" label="Slot 类型" initialValue="local" rules={[{ required: true }]}>
            <Select options={[
              { value: 'local', label: '本地 (Local)' },
              { value: 'tpm2', label: 'TPM2 (硬件安全)' },
              { value: 'cloud', label: '云端 (Cloud)' },
            ]} />
          </Form.Item>
          <Form.Item name="user_uuid" label="所属用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select
              options={users.map((u) => ({ value: u.uuid, label: u.display_name }))}
              placeholder="选择用户"
            />
          </Form.Item>
          <Form.Item name="password" label="卡片密码" rules={[{ required: true, message: '请设置卡片密码' }]}>
            <Input.Password placeholder="用于保护卡片主密钥" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} placeholder="可选备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 证书抽屉 */}
      <Drawer
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>{selectedCard?.card_name} — 证书列表</span>
            {selectedCard && <Tag color={slotColor(selectedCard.slot_type)}>{selectedCard.slot_type?.toUpperCase()}</Tag>}
          </Space>
        }
        open={certDrawerOpen}
        onClose={() => setCertDrawerOpen(false)}
        width={700}
        extra={
          <Button
            type="primary" size="small" icon={<KeyOutlined />}
            onClick={() => setKeygenOpen(true)}
          >
            生成密钥对
          </Button>
        }
      >
        {selectedCard && (
          <Descriptions size="small" style={{ marginBottom: 16 }} column={2}>
            <Descriptions.Item label="卡片 UUID">
              <Text copyable style={{ fontSize: 12 }}>{selectedCard.uuid}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="类型">
              <Tag color={slotColor(selectedCard.slot_type)}>{selectedCard.slot_type}</Tag>
            </Descriptions.Item>
          </Descriptions>
        )}

        {certs.length === 0 && !certsLoading ? (
          <Empty description="暂无证书，点击「生成密钥对」创建" />
        ) : (
          <Table
            dataSource={certs}
            columns={certColumns}
            rowKey="uuid"
            loading={certsLoading}
            pagination={false}
            size="small"
          />
        )}
      </Drawer>

      {/* 密钥生成弹窗 */}
      <Modal
        title={
          <Space>
            <KeyOutlined />
            <span>生成密钥对 — {selectedCard?.card_name}</span>
          </Space>
        }
        open={keygenOpen}
        onOk={handleKeygen}
        onCancel={() => { setKeygenOpen(false); keygenForm.resetFields(); }}
        okText="生成" cancelText="取消"
        confirmLoading={keygening}
        width={420}
      >
        <Form form={keygenForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item name="key_type" label="密钥类型" initialValue="ec256" rules={[{ required: true }]}>
            <Select options={KEY_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="password" label="卡片密码" rules={[{ required: true, message: '请输入卡片密码' }]}>
            <Input.Password placeholder="验证卡片身份" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input placeholder="例如：TLS 签名密钥" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Cards;
