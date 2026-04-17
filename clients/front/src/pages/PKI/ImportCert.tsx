import React from 'react';
import { Card, Space, Upload, message } from 'antd';
import { ImportOutlined } from '@ant-design/icons';
import { importCert } from '../../api';

const ImportCertPage: React.FC = () => {
  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <Space>
            <ImportOutlined />
            <span>导入证书</span>
          </Space>
        }
      >
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
          <p className="ant-upload-drag-icon">
            <ImportOutlined style={{ fontSize: 48, color: '#1677ff' }} />
          </p>
          <p className="ant-upload-text">拖拽证书文件到此处，或点击选择</p>
          <p className="ant-upload-hint">支持 PEM / DER / PKCS#12 / PKCS#7 格式</p>
        </Upload.Dragger>
      </Card>
    </div>
  );
};

export default ImportCertPage;
