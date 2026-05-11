# 消息通知接口

本文档描述 `tinypanel-hub` 的消息通知机制。

这套接口面向嵌入式信息站设备，目标是让设备用很小的轮询开销发现消息，再按需拉取完整内容。

## 核心思路

服务端把消息发布到某个 `channel`。

设备端定时轮询订阅端点：

```http
GET /api/v1/subscriptions/{channel}?device_id={device_id}
```

订阅端点只返回：

- 当前设备还有多少条未读消息。
- 未读消息的 ID 列表。

设备拿到 ID 后，再逐条调用：

```http
GET /api/v1/messages/{id}
```

设备成功拉取并处理消息后，调用：

```http
POST /api/v1/messages/{id}/ack
```

服务端记录该设备已经确认过这条消息。下次这个设备轮询同一频道时，这条消息就不会再出现在未读列表里。

设备端拉取到消息内容后，统一以闪烁模态框展示。模态框内文字建议使用普通正文的 2 倍字号。

## 术语

`channel`

消息频道。可以用来区分不同用途的消息，例如 `desk`、`system`、`weather`、`alert`。如果发布消息时不传 `channel`，服务端会使用 `default`。

`device_id`

设备 ID。服务端用它区分不同设备的已读状态。不同设备对同一条消息的 ack 互不影响。

`message_id`

服务端生成的自增消息 ID。订阅端点只返回 ID，设备再通过 ID 拉取完整消息。

`ack`

确认消息已被某个设备处理。ack 是幂等的：同一设备重复 ack 同一条消息，不会产生额外效果。

## 鉴权

如果配置了 `server.api_token`，所有 `/api/v1/` 接口都需要 Token。

推荐使用：

```http
Authorization: Bearer change-me
```

也可以使用：

```http
X-API-Token: change-me
```

后续示例默认使用：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
```

## 数据结构

### Message

```json
{
  "id": 1,
  "channel": "desk",
  "author": "hub",
  "body": "hello panel",
  "created_at": "2026-05-10T13:30:00Z"
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | integer | 服务端生成的消息 ID |
| `channel` | string | 消息频道 |
| `author` | string | 消息发送者；为空时默认为 `anonymous` |
| `body` | string | 消息内容；必填 |
| `created_at` | string | 服务端创建时间，RFC3339 格式 |

### MessageSubscription

```json
{
  "device_id": "tinypanel-001",
  "channel": "desk",
  "unread_count": 1,
  "message_ids": [1]
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `device_id` | string | 本次查询的设备 ID |
| `channel` | string | 本次查询的频道 |
| `unread_count` | integer | 该设备在该频道下的未读总数 |
| `message_ids` | integer array | 未读消息 ID 列表，受 `limit` 限制 |

`unread_count` 表示未读总数，`message_ids` 表示本次返回的 ID 列表。如果未读很多，`unread_count` 可能大于 `message_ids.length`。

## 接口

### 发布消息

```http
POST /api/v1/messages
```

请求体：

```json
{
  "channel": "desk",
  "author": "hub",
  "body": "hello panel"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `channel` | string | 否 | 消息频道，默认 `default` |
| `author` | string | 否 | 发送者，默认 `anonymous` |
| `body` | string | 是 | 消息内容 |

成功响应：`201 Created`

```json
{
  "id": 1,
  "channel": "desk",
  "author": "hub",
  "body": "hello panel",
  "created_at": "2026-05-10T13:30:00Z"
}
```

PowerShell 示例：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
Invoke-RestMethod http://localhost:8080/api/v1/messages -Method Post -Headers $headers -ContentType "application/json" -Body '{"channel":"desk","author":"hub","body":"hello panel"}'
```

### 查询最近消息

```http
GET /api/v1/messages?limit=20
```

用于调试或管理端查看最近消息。

查询参数：

| 参数 | 必填 | 默认 | 范围 | 说明 |
| --- | --- | --- | --- | --- |
| `limit` | 否 | `20` | `1` 到 `100` | 返回最近多少条消息 |

成功响应：`200 OK`

```json
[
  {
    "id": 1,
    "channel": "desk",
    "author": "hub",
    "body": "hello panel",
    "created_at": "2026-05-10T13:30:00Z"
  }
]
```

### 轮询频道订阅

```http
GET /api/v1/subscriptions/{channel}?device_id={device_id}&limit=20
```

这是设备端最常调用的接口。它不会返回消息正文，只返回未读数量和消息 ID。

路径参数：

| 参数 | 说明 |
| --- | --- |
| `channel` | 要订阅的频道 |

查询参数：

| 参数 | 必填 | 默认 | 范围 | 说明 |
| --- | --- | --- | --- | --- |
| `device_id` | 是 | 无 | - | 设备 ID |
| `limit` | 否 | `20` | `1` 到 `100` | 本次最多返回多少个消息 ID |

成功响应：`200 OK`

```json
{
  "device_id": "tinypanel-001",
  "channel": "desk",
  "unread_count": 2,
  "message_ids": [1, 2]
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod "http://localhost:8080/api/v1/subscriptions/desk?device_id=tinypanel-001" -Headers $headers
```

### 拉取单条消息

```http
GET /api/v1/messages/{id}
```

路径参数：

| 参数 | 说明 |
| --- | --- |
| `id` | 消息 ID，必须是正整数 |

成功响应：`200 OK`

```json
{
  "id": 1,
  "channel": "desk",
  "author": "hub",
  "body": "hello panel",
  "created_at": "2026-05-10T13:30:00Z"
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/messages/1 -Headers $headers
```

### 确认消息

```http
POST /api/v1/messages/{id}/ack
```

请求体：

```json
{
  "device_id": "tinypanel-001"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `device_id` | string | 是 | 确认消息的设备 ID |

成功响应：`200 OK`

```json
{
  "device_id": "tinypanel-001",
  "message_id": 1,
  "acked": true
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/messages/1/ack -Method Post -Headers $headers -ContentType "application/json" -Body '{"device_id":"tinypanel-001"}'
```

## 设备端推荐流程

1. 设备启动后确定自己的 `device_id`。
2. 设备选择要订阅的频道，例如 `desk`。
3. 定时调用订阅端点：

```http
GET /api/v1/subscriptions/desk?device_id=tinypanel-001&limit=10
```

4. 如果 `unread_count` 为 `0`，本轮结束。
5. 如果 `message_ids` 不为空，逐个调用 `GET /api/v1/messages/{id}` 拉取内容。
6. 设备显示或处理消息。
7. 处理成功后调用 `POST /api/v1/messages/{id}/ack`。
8. 下一轮轮询时，已 ack 的消息不会再返回。

消息展示不需要额外字段控制。只要设备端通过 `GET /api/v1/messages/{id}` 拉取到消息，就按闪烁模态框处理。

建议设备端只在内容真正写入本地状态或显示成功后 ack。如果拉取成功但处理失败，可以不 ack，下一轮还能再次看到该消息。

## 错误响应

错误响应统一为：

```json
{
  "error": "message"
}
```

常见状态码：

| 状态码 | 场景 |
| --- | --- |
| `400 Bad Request` | 请求体 JSON 错误、缺少 `body`、缺少 `device_id`、消息 ID 不是正整数 |
| `401 Unauthorized` | 配置了 Token，但请求没有携带正确 Token |
| `404 Not Found` | 指定的消息不存在 |
| `500 Internal Server Error` | 服务端存储失败 |

## 当前存储行为

当前实现使用 JSON 文件持久化：

- 消息保存在 `messages` 中。
- 每个设备的 ack 状态保存在 `message_acks` 中。
- 默认最多保留最近 100 条消息。
- 旧版本没有 `channel` 字段的消息会在加载时补成 `default`。

这套实现适合当前轻量服务端。后续如果消息量变大，可以在不改变外部接口的前提下，把存储层替换成更适合查询和清理的实现。
