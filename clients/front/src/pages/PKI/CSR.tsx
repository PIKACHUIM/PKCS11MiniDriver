import React, { useEffect, useState } from 'react';
import {
  Table, Button, Space, Tag, Typography, Modal, Form, Input, Select,
  Popconfirm, message, Tooltip, Card, Drawer, Divider, Row, Col, Checkbox,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, ReloadOutlined, DownloadOutlined,
  CopyOutlined, EyeOutlined, KeyOutlined,
} from '@ant-design/icons';
import { getCSRList, createCSR, deleteCSR, downloadCSRFile, getCards } from '../../api';
import type { CSRRecord, CreateCSRRequest, Card as CardType } from '../../types';
import { useAppStore } from '../../store/appStore';
import dayjs from 'dayjs';

const { Text } = Typography;
const { TextArea } = Input;

const KEY_TYPE_OPTIONS = [
  { label: 'RSA 2048', value: 'rsa2048' },
  { label: 'RSA 4096', value: 'rsa4096' },
  { label: 'RSA 8192', value: 'rsa8192' },
  { label: 'EC P-256（推荐）', value: 'ec256' },
  { label: 'EC P-384', value: 'ec384' },
  { label: 'EC P-521', value: 'ec521' },
  { label: 'Ed25519', value: 'ed25519' },
  { label: 'SM2', value: 'sm2' },
];

const KEY_USAGE_OPTIONS = [
  { label: '数字签名', value: 'digitalSignature' },
  { label: '内容加密', value: 'keyEncipherment' },
  { label: '数据加密', value: 'dataEncipherment' },
  { label: '密钥协商', value: 'keyAgreement' },
  { label: '证书签名', value: 'certSign' },
  { label: 'CRL 签名', value: 'crlSign' },
];

const EXT_KEY_USAGE_OPTIONS = [
  { label: 'TLS 服务器认证', value: 'serverAuth' },
  { label: 'TLS 客户端认证', value: 'clientAuth' },
  { label: '代码签名', value: 'codeSigning' },
  { label: '邮件保护', value: 'emailProtection' },
  { label: '时间戳', value: 'timeStamping' },
  { label: 'OCSP 签名', value: 'ocspSigning' },
];

const CSRPage: React.FC = () => {
  const { darkMode } = useAppStore();
  const [list, setList] = useState<CSRRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [cards, setCards] = useState<CardType[]>([]);

  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();
  const [keyStorage, setKeyStorage] = useState<'database' | 'smartcard'>('database');

  const [viewOpen, setViewOpen] = useState(false);
  const [viewRecord, setViewRecord] = useState<CSRRecord | null>(null);

  const load = async (p = page) => {
    setLoading(true);
    try {
      const res = await getCSRList({ page: p, page_size: 10 });
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
      const values = await form.validateFields();
      setCreating(true);
      await createCSR(values as CreateCSRRequest);
      message.success('CSR 已生成');
      setCreateOpen(false);
      form.resetFields();
      setKeyStorage('database');
      load();
    } catch (e: any) { if (e.message) message.error(e.message); }
    finally { setCreating(false); }
  };

  const cardStyle = {
    background: darkMode ? '#161b22' : '#fff',
    border: darkMode ? '1px solid #21262d' : '1px solid #f0f0f0',
    borderRadius: 12,
  };

  const columns = [
    {
      title: '通用名称 (CN)',
      dataIndex: 'common_name',
      render: (v: string) => <Text strong style={{ color: darkMode ? '#c9d1d9' : undefined }}>{v}</Text>,
    },
    {
      title: '组织 (O)',
      dataIndex: 'organization',
      render: (v: string) => <Text style={{ color: darkMode ? '#8b949e' : '#666' }}>{v || '-'}</Text>,
    },
    {
      title: '密钥类型',
      dataIndex: 'key_type',
      width: 100,
      render: (v: string) => <Tag color="blue">{v?.toUpperCase()}</Tag>,
    },
    {
      title: '密钥存储',
      dataIndex: 'key_storage',
      width: 100,
      render: (v: string) => (
        <Tag color={v === 'smartcard' ? 'purple' : 'green'}>
          {v === 'smartcard' ? '智能卡' : '数据库'}
        </Tag>
      ),
    },
    {
      title: '私钥',
      dataIndex: 'has_private_key',
      width: 60,
      render: (v: boolean) => <Tag color={v ? 'green' : 'default'}>{v ? '有' : '无'}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 150,
      render: (v: string) => (
        <Text style={{ fontSize: 12, color: darkMode ? '#8b949e' : '#999' }}>
          {dayjs(v).format('YYYY-MM-DD HH:mm')}
        </Text>
      ),
    },
    {
      title: '操作',
      width: 130,
      render: (_: any, record: CSRRecord) => (
        <Space>
          <Tooltip title="查看 CSR">
            <Button type="text" size="small" icon={<EyeOutlined />}
              onClick={() => { setViewRecord(record); setViewOpen(true); }} />
          </Tooltip>
          <Tooltip title="下载 CSR">
            <Button type="text" size="small" icon={<DownloadOutlined />}
              onClick={() => downloadCSRFile(record.uuid, `${record.common_name}.csr`).catch((e) => message.error(e.message))} />
          </Tooltip>
          <Popconfirm title="确认删除此 CSR？若有私钥将一并删除。"
            onConfirm={() => deleteCSR(record.uuid).then(() => { message.success('已删除'); load(); }).catch((e) => message.error(e.message))}
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
          <KeyOutlined style={{ marginRight: 8 }} />CSR 管理
        </Text>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => load()}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>生成 CSR</Button>
        </Space>
      </div>

      <Card style={cardStyle} bodyStyle={{ padding: 0 }}>
        <Table dataSource={list} columns={columns} rowKey="uuid" loading={loading}
          pagination={{ current: page, total, pageSize: 10, onChange: (p) => { setPage(p); load(p); }, showTotal: (t) => `共 ${t} 条` }} />
      </Card>

      {/* 生成 CSR 弹窗 */}
      <Modal title={<Space><KeyOutlined />生成 CSR</Space>} open={createOpen}
        onOk={handleCreate} onCancel={() => { setCreateOpen(false); form.resetFields(); setKeyStorage('database'); }}
        okText="生成" cancelText="取消" confirmLoading={creating} width={700}>
        <Form form={form} layout="vertical" style={{ marginTop: 16 }}
          initialValues={{ key_type: 'ec256', key_storage: 'database' }}>
          <Divider orientation="left" style={{ fontSize: 13 }}>主体信息</Divider>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="common_name" label="通用名称 (CN)" rules={[{ required: true }]}>
                <Input placeholder="example.com" />
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
              <Form.Item name="org_unit" label="部门 (OU)">
                <Input placeholder="IT Department" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="country" label="国家 (C)">
                <Input placeholder="CN" maxLength={2} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="state" label="省份 (ST)">
                <Input placeholder="Beijing" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="locality" label="城市 (L)">
                <Input placeholder="Beijing" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="email" label="邮箱 (E)">
                <Input placeholder="admin@example.com" />
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left" style={{ fontSize: 13 }}>密钥参数</Divider>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="key_type" label="密钥类型" rules={[{ required: true }]}>
                <Select options={KEY_TYPE_OPTIONS} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="key_storage" label="密钥存储位置" rules={[{ required: true }]}>
                <Select onChange={(v) => setKeyStorage(v)} options={[
                  { label: '存储到数据库（可导出私钥）', value: 'database' },
                  { label: '片上生成（智能卡，不可导出）', value: 'smartcard' },
                ]} />
              </Form.Item>
            </Col>
          </Row>
          {keyStorage === 'smartcard' && (
            <Form.Item name="card_uuid" label="目标智能卡" rules={[{ required: true, message: '请选择智能卡' }]}>
              <Select placeholder="选择智能卡（密钥将在卡上生成）"
                options={cards.map((c) => ({ value: c.uuid, label: `${c.card_name} (${c.slot_type})` }))} />
            </Form.Item>
          )}

          <Divider orientation="left" style={{ fontSize: 13 }}>SAN 扩展</Divider>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="san_dns" label="DNS 名称（逗号分隔）">
                <Input placeholder="example.com, *.example.com" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="san_ip" label="IP 地址（逗号分隔）">
                <Input placeholder="192.168.1.1, 10.0.0.1" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="san_email" label="邮箱 SAN">
                <Input placeholder="user@example.com" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="san_uri" label="URI SAN">
                <Input placeholder="https://example.com" />
              </Form.Item>
            </Col>
          </Row>

          <Divider orientation="left" style={{ fontSize: 13 }}>密钥用途</Divider>
          <Form.Item name="key_usage" label="密钥用途 (Key Usage)">
            <Checkbox.Group options={KEY_USAGE_OPTIONS} />
          </Form.Item>
          <Form.Item name="ext_key_usage" label="扩展密钥用途 (Extended Key Usage)">
            <Checkbox.Group options={EXT_KEY_USAGE_OPTIONS} />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input placeholder="可选备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 查看 CSR 抽屉 */}
      <Drawer title={<Space><EyeOutlined />查看 CSR — {viewRecord?.common_name}</Space>}
        open={viewOpen} onClose={() => setViewOpen(false)} width={600}
        extra={
          <Space>
            <Button size="small" icon={<CopyOutlined />} onClick={() => {
              if (viewRecord?.csr_pem) { navigator.clipboard.writeText(viewRecord.csr_pem); message.success('已复制'); }
            }}>复制</Button>
            <Button size="small" icon={<DownloadOutlined />}
              onClick={() => viewRecord && downloadCSRFile(viewRecord.uuid, `${viewRecord.common_name}.csr`).catch((e) => message.error(e.message))}>
              下载
            </Button>
          </Space>
        }>
        {viewRecord && (
          <>
            <Row gutter={[16, 8]} style={{ marginBottom: 16 }}>
              <Col span={12}><Text type="secondary">通用名称：</Text><Text strong>{viewRecord.common_name}</Text></Col>
              <Col span={12}><Text type="secondary">组织：</Text><Text>{viewRecord.organization || '-'}</Text></Col>
              <Col span={12}><Text type="secondary">密钥类型：</Text><Tag color="blue">{viewRecord.key_type}</Tag></Col>
              <Col span={12}><Text type="secondary">存储位置：</Text>
                <Tag color={viewRecord.key_storage === 'smartcard' ? 'purple' : 'green'}>
                  {viewRecord.key_storage === 'smartcard' ? '智能卡' : '数据库'}
                </Tag>
              </Col>
              <Col span={12}><Text type="secondary">含私钥：</Text><Tag color={viewRecord.has_private_key ? 'green' : 'default'}>{viewRecord.has_private_key ? '是' : '否'}</Tag></Col>
              <Col span={12}><Text type="secondary">创建时间：</Text><Text>{dayjs(viewRecord.created_at).format('YYYY-MM-DD HH:mm')}</Text></Col>
            </Row>
            <Divider style={{ margin: '8px 0' }} />
            <Text type="secondary" style={{ fontSize: 12 }}>CSR 内容（PEM）</Text>
            <TextArea value={viewRecord.csr_pem} rows={14} readOnly
              style={{ fontFamily: 'monospace', fontSize: 11, marginTop: 8 }} />
          </>
        )}
      </Drawer>
    </div>
  );
};

export default CSRPage;
