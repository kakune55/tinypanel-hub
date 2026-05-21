# 设备消息接口

本文档描述 `tinypanel-hub` 的设备绑定和设备收件箱机制。

核心模型：

```text
User -> Device -> Message
```

用户登录 Hub 后绑定自己的设备，再向设备发送消息。设备端只关心自己的收件箱，不再使用频道或订阅概念。

## 鉴权

用户侧 API 使用用户 Token：

```http
Authorization: Bearer user-token
```

设备侧 API 使用设备凭据：

```http
X-Device-ID: tinypanel-001
X-Device-Secret: device-secret
```

本地管理接口使用配置里的 `server.api_token`，用于创建开发账号：

```http
Authorization: Bearer change-me
```

## 创建用户

```http
POST /api/v1/admin/users
```

请求体：

```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "api_token": "alice-token"
}
```

`api_token` 可省略，服务端会生成一个并只在本次响应中返回。

## 设备 hello

```http
POST /api/v1/device/hello
X-Device-ID: tinypanel-001
```

首次请求不需要 `X-Device-Secret`。服务端会创建设备、返回设备 secret 和绑定码。

```json
{
  "device_id": "tinypanel-001",
  "device_secret": "generated-secret",
  "bound": false,
  "bind_code": "483921",
  "bind_code_ttl": 600,
  "server_time": "2026-05-21T00:00:00Z"
}
```

后续请求必须携带 `X-Device-Secret`。未绑定设备的绑定码默认有效 10 分钟，过期后再次 hello 会刷新。

## 绑定设备

```http
POST /api/v1/devices/bind
Authorization: Bearer alice-token
```

```json
{
  "bind_code": "483921",
  "name": "书桌屏幕"
}
```

成功后设备归属当前用户，绑定码失效。

## 用户侧设备接口

```http
GET    /api/v1/me
GET    /api/v1/devices
GET    /api/v1/devices/{device_id}
PATCH  /api/v1/devices/{device_id}
DELETE /api/v1/devices/{device_id}
```

用户只能访问自己名下的设备。

## 用户发送消息

```http
POST /api/v1/devices/{device_id}/messages
Authorization: Bearer alice-token
```

```json
{
  "body": "hello panel",
  "priority": "normal"
}
```

`priority` 支持 `normal` 和 `high`，省略时默认为 `normal`。

查询设备消息：

```http
GET /api/v1/devices/{device_id}/messages?limit=50
```

## 设备拉取和确认消息

```http
GET /api/v1/device/messages?limit=10
X-Device-ID: tinypanel-001
X-Device-Secret: device-secret
```

响应：

```json
{
  "device_id": "tinypanel-001",
  "messages": [
    {
      "id": 1,
      "owner_id": "usr_x",
      "device_id": "tinypanel-001",
      "author_id": "usr_x",
      "body": "hello panel",
      "priority": "normal",
      "status": "pending",
      "created_at": "2026-05-21T00:00:00Z"
    }
  ]
}
```

确认消息：

```http
POST /api/v1/device/messages/ack
```

```json
{
  "message_ids": [1, 2]
}
```

成功响应：

```json
{
  "device_id": "tinypanel-001",
  "acked_ids": [1, 2]
}
```

## 设备遥测和快照

设备遥测使用设备凭据，不再由请求体决定 `device_id`：

```http
POST /api/v1/device/telemetry
POST /api/v1/device/telemetry/batch
GET  /api/v1/device/snapshot
```

用户可查询自己设备的遥测：

```http
GET /api/v1/devices/{device_id}/telemetry?limit=50
```

## 错误响应

```json
{
  "error": "message",
  "error_detail": {
    "code": "invalid_request",
    "message": "message"
  }
}
```
