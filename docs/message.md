# 设备绑定和消息接口

本文档描述 `tinypanel-hub` 的设备绑定和设备收件箱机制。

核心模型：

```text
User -> Device -> Message
```

用户绑定设备后，可以向自己的设备发送消息。设备端只拉取自己的待处理消息，显示或处理成功后 ack。

## 数据结构

### User

```json
{
  "id": "usr_8f2c1a7b",
  "name": "Alice",
  "email": "alice@example.com",
  "created_at": "2026-05-21T00:00:00Z"
}
```

### Device

```json
{
  "id": "tinypanel-001",
  "owner_id": "usr_8f2c1a7b",
  "name": "书桌屏幕",
  "bound_at": "2026-05-21T00:02:00Z",
  "last_seen_at": "2026-05-21T00:03:00Z",
  "created_at": "2026-05-21T00:01:00Z"
}
```

### Message

```json
{
  "id": 1,
  "owner_id": "usr_8f2c1a7b",
  "device_id": "tinypanel-001",
  "author_id": "usr_8f2c1a7b",
  "body": "hello panel",
  "priority": "normal",
  "status": "pending",
  "created_at": "2026-05-21T00:04:00Z"
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `priority` | `normal` 或 `high` |
| `status` | `pending` 或 `acked` |
| `acked_at` | 设备确认消息后出现 |

## 创建开发用户

```http
POST /api/v1/admin/users
Authorization: Bearer change-me
```

请求体：

```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "api_token": "alice-token"
}
```

`api_token` 可省略。省略时服务端生成 token，并只在本次响应中返回。

成功响应：

```json
{
  "user": {
    "id": "usr_8f2c1a7b",
    "name": "Alice",
    "email": "alice@example.com",
    "created_at": "2026-05-21T00:00:00Z"
  },
  "api_token": "alice-token"
}
```

## 设备 hello

```http
POST /api/v1/device/hello
X-Device-ID: tinypanel-001
```

首次 hello 不需要 `X-Device-Secret`。服务端会创建设备记录并返回设备 secret。

首次响应：

```json
{
  "device_id": "tinypanel-001",
  "device_secret": "generated-secret",
  "bound": false,
  "bind_code": "483921",
  "bind_code_ttl": 600,
  "server_time": "2026-05-21T00:01:00Z"
}
```

设备必须保存 `device_secret`。后续 hello：

```http
POST /api/v1/device/hello
X-Device-ID: tinypanel-001
X-Device-Secret: generated-secret
```

未绑定设备会继续返回绑定码。绑定码默认 10 分钟有效，过期后再次 hello 会刷新。

已绑定设备响应：

```json
{
  "device_id": "tinypanel-001",
  "bound": true,
  "name": "书桌屏幕",
  "server_time": "2026-05-21T00:03:00Z",
  "bound_at": "2026-05-21T00:02:00Z"
}
```

## 绑定设备

```http
POST /api/v1/devices/bind
Authorization: Bearer alice-token
```

请求体：

```json
{
  "bind_code": "483921",
  "name": "书桌屏幕"
}
```

成功后，设备归属当前用户，绑定码立即失效。

失败情况：

| 状态码 | 场景 |
| --- | --- |
| `404` | 绑定码不存在 |
| `409` | 绑定码过期或已被使用 |

## 用户侧设备管理

```http
GET    /api/v1/devices
GET    /api/v1/devices/{device_id}
PATCH  /api/v1/devices/{device_id}
DELETE /api/v1/devices/{device_id}
```

重命名设备：

```http
PATCH /api/v1/devices/tinypanel-001
Authorization: Bearer alice-token
Content-Type: application/json
```

```json
{
  "name": "客厅屏幕"
}
```

用户只能访问自己名下的设备。其他用户的设备会按 `404` 处理。

## 用户发送消息

```http
POST /api/v1/devices/{device_id}/messages
Authorization: Bearer alice-token
```

请求体：

```json
{
  "body": "hello panel",
  "priority": "normal"
}
```

`body` 必填。`priority` 可省略，默认 `normal`。

查询设备消息历史：

```http
GET /api/v1/devices/{device_id}/messages?limit=50
Authorization: Bearer alice-token
```

返回最近消息，按新到旧排序。

## 设备拉取消息

```http
GET /api/v1/device/messages?limit=10
X-Device-ID: tinypanel-001
X-Device-Secret: generated-secret
```

响应：

```json
{
  "device_id": "tinypanel-001",
  "messages": [
    {
      "id": 1,
      "owner_id": "usr_8f2c1a7b",
      "device_id": "tinypanel-001",
      "author_id": "usr_8f2c1a7b",
      "body": "hello panel",
      "priority": "normal",
      "status": "pending",
      "created_at": "2026-05-21T00:04:00Z"
    }
  ]
}
```

只返回当前设备的 `pending` 消息。

## 设备确认消息

```http
POST /api/v1/device/messages/ack
X-Device-ID: tinypanel-001
X-Device-Secret: generated-secret
Content-Type: application/json
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

如果某些 ID 不属于当前设备或不存在，会出现在 `missing_ids` 中。

```json
{
  "device_id": "tinypanel-001",
  "acked_ids": [1],
  "missing_ids": [99]
}
```

## 设备快照

```http
GET /api/v1/device/snapshot
X-Device-ID: tinypanel-001
X-Device-Secret: generated-secret
```

当前返回天气和设备待处理消息。后续可继续加入设备端需要的轻量状态。

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
