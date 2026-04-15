import React, { useState } from 'react';
import {
  Card, Tabs, Form, Input, Select, Button, Space, Tag, message,
  InputNumber, Row, Col, Checkbox, Table, Typography, Divider, Upload,
  Modal, Descriptions,
} from 'antd';
import {
  SafetyCertificateOutlined, FileProtectOutlined, KeyOutlined,
  CloudUploadOutlined, DownloadOutlined, PlusOutlined,
  ImportOutlined, ExportOutlined, CopyOutlined, BankOutlined,
} from '@ant-design/icons';
import {
  generateSelfSigned, createLocalCA, generateCSR,
  importCert, exportCert, getLocalCAs,
} from '../../api';
import type { LocalCA, CSRRequest, SelfSignRequest } from '../../types';

const { TextArea } = Input;
const { Text } = Typography;

/** 密钥类型选项 */
const keyTypeOptions = [
  { label: 'RSA 2048', value: 'rsa2048' },
  { label: 'RSA 4096', value: 'rsa4096' },
  { label: 'RSA 8192', value: 'rsa8192' },
  { label: 'EC P-256', value: 'ec256' },
  { label: 'EC P-384', value: 'ec384' },
  { label: 'EC P-521', value: 'ec521' },
  { label: 'Ed25519', value: 'ed25519' },
  { label: 'SM2', value: 'sm2' },
];

/** 密钥用途选项 */
const keyUsageOptions = [
  { label: '数字签名', value: 'digitalSignature' },
  { label: '内容加密', value: 'keyEncipherment' },
  { label: '数据加密', value: 'dataEncipherment' },
  { label: '密钥协商', value: 'keyAgreement' },
  { label: '证书签名', value: 'certSign' },
  { label: 'CRL 签名', value: 'crlSign' },
];

/** 扩展密钥用途选项 */
const extKeyUsageOptions = [
  { label: 'TLS 服务器认证', value: 'serverAuth' },
  { label: 'TLS 客户端认证', value: 'clientAuth' },
  { label: '代码签名', value: 'codeSigning' },
  { label: '邮件保护', value: 'emailProtection' },
  { label: '时间戳', value: 'timeStamping' },
  { label: 'OCSP 签名', value: 'ocspSigning' },
];

// ---- 自签名证书 Tab ----
const SelfSignTab: React.FC = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      await generateSelfSigned(values);
      message.success('自签名证书已生成');
      form.resetFields();
    } catch (err: any) {
      if (err.message) message.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form form={form} layout="vertical" initialValues={{ key_type: 'ec256', validity_days: 365 }}>
      <Divider orientation="left">主体信息</Divider>
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
          <Form.Item name="locality" label="城市 (L)">
            <Input placeholder="Beijing" />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left">证书参数</Divider>
      <Row gutter={16}>
        <Col span={8}>
          <Form.Item name="key_type" label="密钥类型" rules={[{ required: true }]}>
            <Select options={keyTypeOptions} />
          </Form.Item>
        </Col>
        <Col span={8}>
          <Form.Item name="validity_days" label="有效期（天）" rules={[{ required: true }]}>
            <InputNumber min={1} max={3650} style={{ width: '100%' }} />
          </Form.Item>
        </Col>
        <Col span={8}>
          <Form.Item name="card_uuid" label="存储卡片 UUID" rules={[{ required: true }]}>
            <Input placeholder="目标智能卡 UUID" />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left">SAN 扩展</Divider>
      <Form.Item name="san_dns" label="DNS 名称（逗号分隔）">
        <Input placeholder="example.com, *.example.com" />
      </Form.Item>
      <Form.Item name="san_ip" label="IP 地址（逗号分隔）">
        <Input placeholder="192.168.1.1, 10.0.0.1" />
      </Form.Item>
      <Form.Item name="san_email" label="邮箱地址（逗号分隔）">
        <Input placeholder="admin@example.com" />
      </Form.Item>

      <Divider orientation="left">密钥用途</Divider>
      <Form.Item name="key_usage" label="密钥用途">
        <Checkbox.Group options={keyUsageOptions} />
      </Form.Item>
      <Form.Item name="ext_key_usage" label="扩展密钥用途">
        <Checkbox.Group options={extKeyUsageOptions} />
      </Form.Item>

      <Form.Item name="export_also" valuePropName="checked">
        <Checkbox>同时导出证书文件</Checkbox>
      </Form.Item>

      <Form.Item>
        <Button type="primary" icon={<SafetyCertificateOutlined />} onClick={handleSubmit} loading={loading}>
          生成自签名证书
        </Button>
      </Form.Item>
    </Form>
  );
};

// ---- 本地 CA 管理 Tab ----
const LocalCATab: React.FC = () => {
  const [cas, setCAs] = useState<LocalCA[]>([]);
  const [loading, setLoading] = useState(false);
  const [createVisible, setCreateVisible] = useState(false);
  const [form] = Form.useForm();

  const loadCAs = async () => {
    setLoading(true);
    try {
      const data = await getLocalCAs();
      setCAs(data || []);
    } catch (err: any) {
      message.error(err.message || '加载 CA 列表失败');
    } finally {
      setLoading(false);
    }
  };

  React.useEffect(() => { loadCAs(); }, []);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await createLocalCA(values);
      message.success('本地 CA 已创建');
      setCreateVisible(false);
      form.resetFields();
      loadCAs();
    } catch (err: any) {
      if (err.message) message.error(err.message);
    }
  };

  const columns = [
    { title: 'CA 名称', dataIndex: 'name', width: 200 },
    {
      title: '有效期',
      render: (_: unknown, r: LocalCA) => (
        <Text>{r.not_before} ~ {r.not_after}</Text>
      ),
    },
    { title: '已签发', dataIndex: 'issued_count', width: 80 },
    {
      title: '状态',
      width: 80,
      render: (_: unknown, r: LocalCA) => (
        <Tag color={r.revoked ? 'red' : 'green'}>{r.revoked ? '已吊销' : '有效'}</Tag>
      ),
    },
  ];

  return (
    <>
      <Space style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateVisible(true)}>
          创建本地 CA
        </Button>
      </Space>
      <Table rowKey="uuid" columns={columns} dataSource={cas} loading={loading} pagination={false} />

      <Modal title="创建本地 CA" open={createVisible} onOk={handleCreate}
        onCancel={() => { setCreateVisible(false); form.resetFields(); }} width={480}>
        <Form form={form} layout="vertical" initialValues={{ key_type: 'ec256', validity_years: 10 }}>
          <Form.Item name="name" label="CA 名称" rules={[{ required: true }]}>
            <Input placeholder="My Root CA" />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="key_type" label="密钥类型">
                <Select options={keyTypeOptions} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="validity_years" label="有效期（年）">
                <InputNumber min={1} max={10} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="card_uuid" label="存储卡片 UUID" rules={[{ required: true }]}>
            <Input placeholder="目标智能卡 UUID" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---- CSR 生成 Tab ----
const CSRTab: React.FC = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [csrResult, setCSRResult] = useState<string>('');

  const handleGenerate = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const resp = await generateCSR(values);
      setCSRResult(resp.csr_pem || '');
      message.success('CSR 已生成');
    } catch (err: any) {
      if (err.message) message.error(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form form={form} layout="vertical" initialValues={{ key_type: 'ec256' }}>
      <Row gutter={16}>
        <Col span={12}>
          <Form.Item name="common_name" label="通用名称 (CN)" rules={[{ required: true }]}>
            <Input placeholder="example.com" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="key_type" label="密钥类型" rules={[{ required: true }]}>
            <Select options={keyTypeOptions} />
          </Form.Item>
        </Col>
      </Row>
      <Form.Item name="card_uuid" label="智能卡 UUID（片上生成密钥）" rules={[{ required: true }]}>
        <Input placeholder="密钥将在此智能卡上生成" />
      </Form.Item>
      <Form.Item name="san_dns" label="DNS 名称（逗号分隔）">
        <Input placeholder="example.com, *.example.com" />
      </Form.Item>

      <Form.Item>
        <Space>
          <Button type="primary" icon={<KeyOutlined />} onClick={handleGenerate} loading={loading}>
            生成 CSR
          </Button>
          {csrResult && (
            <>
              <Button icon={<CopyOutlined />} onClick={() => {
                navigator.clipboard.writeText(csrResult);
                message.success('CSR 已复制到剪贴板');
              }}>复制 CSR</Button>
              <Button icon={<DownloadOutlined />} onClick={() => {
                const blob = new Blob([csrResult], { type: 'application/x-pem-file' });
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url; a.download = 'request.csr'; a.click();
                URL.revokeObjectURL(url);
              }}>下载 CSR</Button>
            </>
          )}
        </Space>
      </Form.Item>

      {csrResult && (
        <Form.Item label="CSR 内容">
          <TextArea value={csrResult} rows={8} readOnly style={{ fontFamily: 'monospace', fontSize: 12 }} />
        </Form.Item>
      )}
    </Form>
  );
};

// ---- 主页面 ----
const PKIPage: React.FC = () => {
  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <Space>
            <FileProtectOutlined />
            <span>本地 PKI 工具</span>
          </Space>
        }
      >
        <Tabs
          items={[
            {
              key: 'selfsign',
              label: <Space><SafetyCertificateOutlined />自签名证书</Space>,
              children: <SelfSignTab />,
            },
            {
              key: 'ca',
              label: <Space><BankOutlined />本地 CA</Space>,
              children: <LocalCATab />,
            },
            {
              key: 'csr',
              label: <Space><KeyOutlined />生成 CSR</Space>,
              children: <CSRTab />,
            },
            {
              key: 'import',
              label: <Space><ImportOutlined />导入证书</Space>,
              children: (
                <Upload.Dragger
                  accept=".pem,.der,.p12,.pfx,.p7b,.cer,.crt"
                  multiple={false}
                  showUploadList={false}
                  customRequest={async ({ file, onSuccess, onError }) => {
                    try {
                      const formData = new FormData();
                      formData.append('file', file as File);
                      await importCert(formData);
                      message.success('证书导入成功');
                      onSuccess?.({});
                    } catch (err: any) {
                      message.error(err.message || '导入失败');
                      onError?.(err);
                    }
                  }}
                >
                  <p className="ant-upload-drag-icon"><ImportOutlined style={{ fontSize: 48, color: '#1677ff' }} /></p>
                  <p className="ant-upload-text">拖拽证书文件到此处，或点击选择</p>
                  <p className="ant-upload-hint">支持 PEM / DER / PKCS#12 / PKCS#7 格式</p>
                </Upload.Dragger>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
};

export default PKIPage;
