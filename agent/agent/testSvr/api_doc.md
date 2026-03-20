# Log Data Receiver API 文档

## 服务信息
- **服务名称**: Log Data Receiver
- **版本**: 1.0
- **运行地址**: http://0.0.0.0:8081

## API 端点列表

### 1. 接收日志数据
- **路径**: `/logs`
- **方法**: `POST`
- **功能**: 接收并处理日志数据
- **请求体**:
  ```json
  {
    "task_name": "任务名称",
    "content": "base64编码的日志内容",
    "read_time": "ISO8601格式的读取时间",
    "offset": 0,
    "size": 1024,
    "content_hash": "内容哈希值"
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "message": "日志数据接收成功",
    "data": {
      "task_name": "任务名称",
      "content": "base64编码的日志内容",
      "read_time": "ISO8601格式的读取时间",
      "offset": 0,
      "size": 1024,
      "content_hash": "内容哈希值"
    }
  }
  ```

### 2. 注册
- **路径**: `/api/env-node-registration/register`
- **方法**: `POST`
- **功能**: 注册代理节点
- **请求体**:
  ```json
  {
    "agentId": "代理ID",
    "data": "注册信息"
  }
  ```
- **响应**:
  ```json
  {
    "code": "0",
    "msg": "注册成功"
  }
  ```

### 3. 状态更新
- **路径**: `/task-status`
- **方法**: `POST`
- **功能**: 更新任务状态
- **请求体**:
  ```json
  {
    "client_id": "代理ID",
    "task_id": "任务ID",
    "status": "状态",
    "message": "状态消息（可选）"
  }
  ```
- **响应**:
  ```json
  {
    "code": 0,
    "message": "状态成功"
  }
  ```

### 4. 断点续传检查
- **路径**: `/api/upload/check`
- **方法**: `POST`
- **功能**: 查询文件已上传的大小
- **请求参数** (Form表单):
  - `file_md5`: 文件MD5值
  - `file_name`: 文件名
- **响应**:
  ```json
  {
    "code": 200,
    "uploaded_size": 1024,
    "msg": "文件已存在部分分片" // 或 "文件未上传过"
  }
  ```

### 5. 断点续传
- **路径**: `/api/upload/continue`
- **方法**: `POST`
- **功能**: 接收分片数据，实现断点续传
- **请求参数** (Form表单):
  - `file_md5`: 文件MD5值
  - `file_name`: 文件名
  - `start_pos`: 开始位置（默认0）
  - `file_chunk`: 文件分片（二进制）
- **响应**:
  ```json
  {
    "code": 200,
    "current_size": 2048,
    "msg": "分片上传成功"
  }
  ```

### 6. 结果同步
- **路径**: `/agent/result_sync`
- **方法**: `POST`
- **功能**: 同步任务执行结果
- **请求体**:
  ```json
  {
    "client_id": "代理ID",
    "task_id": "任务ID",
    "step": 1,
    "file_content": "文件内容",
    "create_at": "ISO8601格式的创建时间"
  }
  ```
- **响应**:
  ```json
  {
    "code": 0,
    "message": "结果同步成功"
  }
  ```

## 数据模型

### LogDataRequest
| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| task_name | string | 是 | 任务名称 |
| content | string | 是 | 日志内容（base64编码） |
| read_time | string | 是 | 读取时间（建议ISO8601格式） |
| offset | integer | 是 | 偏移量 |
| size | integer | 是 | 读取内容大小（字节） |
| content_hash | string | 是 | 内容哈希值 |

### RegDataRequest
| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| agentId | string | 是 | 代理ID |
| data | string | 是 | 注册信息 |

### StatusDataRequest
| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| client_id | string | 是 | 代理ID |
| task_id | string | 是 | 任务ID |
| status | string | 是 | 状态 |
| message | string | 否 | 状态消息 |

### ResultSync
| 字段 | 类型 | 必填 | 描述 |
|------|------|------|------|
| client_id | string | 是 | 代理ID |
| task_id | string | 是 | 任务ID |
| step | integer | 是 | 步骤 |
| file_content | string | 是 | 文件内容 |
| create_at | string | 是 | 创建时间（建议ISO8601格式） |

## 错误处理
- **400 Bad Request**: 请求参数错误，如缺少必要参数或参数格式不正确
- **500 Internal Server Error**: 服务器内部错误，如文件写入失败等

## 部署信息
- **运行命令**: `python pysvr_fastapi.py`
- **监听端口**: 8081
- **文件上传目录**: `./upload_files`
