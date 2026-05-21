# 认证模型

`tinypanel-hub` 当前有三类身份：管理端、用户端、设备端。

## 管理端

管理端使用配置中的 `server.api_token`。

用途：

- 创建开发用户。
- 后续可扩展为本地维护和紧急管理接口。

请求头：

```http
Authorization: Bearer change-me
```

也支持：

```http
X-API-Token: change-me
```

当前管理接口：

```http
POST /api/v1/admin/users
```

## 用户端

用户端使用用户 token。

请求头：

```http
Authorization: Bearer alice-token
```

用户 token 的 hash 保存在 `state.json` 中，接口响应不会返回 hash。

当前用户接口：

```http
GET    /api/v1/me
GET    /api/v1/snapshot
GET    /api/v1/weather
GET    /api/v1/devices
POST   /api/v1/devices/bind
GET    /api/v1/devices/{device_id}
PATCH  /api/v1/devices/{device_id}
DELETE /api/v1/devices/{device_id}
GET    /api/v1/devices/{device_id}/messages
POST   /api/v1/devices/{device_id}/messages
GET    /api/v1/devices/{device_id}/telemetry
GET    /api/v1/todos
POST   /api/v1/todos
GET    /api/v1/todos/{id}
PATCH  /api/v1/todos/{id}
DELETE /api/v1/todos/{id}
```

用户只能访问自己名下的设备。访问其他用户设备时，接口按未找到处理。

## 设备端

设备端使用设备 ID 和设备 secret。

请求头：

```http
X-Device-ID: tinypanel-001
X-Device-Secret: device-secret
```

首次 `POST /api/v1/device/hello` 可以只带 `X-Device-ID`。服务端会创建设备记录，并返回 `device_secret` 和 `bind_code`。设备应保存 `device_secret`，后续所有设备接口都必须携带。

设备 secret 的 hash 保存在 `state.json` 中，普通接口响应不会返回 hash。

当前设备接口：

```http
POST /api/v1/device/hello
GET  /api/v1/device/messages
POST /api/v1/device/messages/ack
POST /api/v1/device/telemetry
POST /api/v1/device/telemetry/batch
GET  /api/v1/device/snapshot
```

## 错误响应

认证失败返回 `401`：

```json
{
  "error": "missing or invalid user token",
  "error_detail": {
    "code": "unauthorized",
    "message": "missing or invalid user token"
  }
}
```

常见错误：

| 状态码 | 场景 |
| --- | --- |
| `400` | 请求体错误、参数缺失或字段非法 |
| `401` | token、设备 ID 或设备 secret 错误 |
| `404` | 资源不存在，或当前用户无权访问该资源 |
| `409` | 绑定码过期、已使用，或 TODO 版本冲突 |
| `500` | 存储写入失败 |
| `502` | 上游天气接口失败 |
