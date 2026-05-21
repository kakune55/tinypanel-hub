# TODO 列表接口

本文档描述 `tinypanel-hub` 的 TODO 列表接口。

这套接口面向用户侧 UI，支持 TODO 的创建、查询、修改和删除。TODO 归属当前用户，不会跨用户共享。修改与删除使用 CAS，避免多个客户端同时操作时覆盖彼此的更新。

## 核心思路

每个 TODO 都有一个服务端生成的 `id` 和一个递增的 `version`。

客户端读取 TODO 后，修改或删除时必须带上当前 `version`：

```http
PATCH /api/v1/todos/{id}
DELETE /api/v1/todos/{id}
```

如果服务端保存的版本仍然等于请求中的 `version`，操作成功；否则返回 `409 Conflict`，客户端应重新读取最新 TODO 后再决定是否重试。

## 鉴权

TODO API 属于用户侧 API，需要用户 Token。认证模型见 [auth.md](auth.md)。

推荐使用：

```http
Authorization: Bearer alice-token
```

后续示例默认使用：

```powershell
$headers = @{ Authorization = "Bearer alice-token" }
```

设备端不需要保存用户 Token。已绑定设备可以使用设备凭据只读同步 TODO：

```http
GET /api/v1/device/todos
X-Device-ID: tinypanel-001
X-Device-Secret: generated-secret
```

设备快照 `GET /api/v1/device/snapshot` 也会包含 `todos` 字段。设备侧不能创建、修改或删除 TODO。

## 数据结构

### Todo

```json
{
  "id": 1,
  "owner_id": "usr_8f2c1a7b",
  "text": "整理桌面",
  "status": 0,
  "version": 1,
  "created_at": "2026-05-11T10:00:00Z",
  "updated_at": "2026-05-11T10:00:00Z"
}
```

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | integer | 服务端生成的 TODO ID |
| `owner_id` | string | TODO 所属用户 ID |
| `text` | string | TODO 内容，去除首尾空白后不能为空，最多 50 个字符 |
| `status` | integer | TODO 状态，取值见下表 |
| `version` | integer | CAS 版本号，创建时为 `1`，每次修改成功后递增 |
| `created_at` | string | 服务端创建时间，RFC3339 格式 |
| `updated_at` | string | 服务端最后更新时间，RFC3339 格式 |

状态值：

| 值 | 说明 |
| --- | --- |
| `0` | 未完成 |
| `1` | 正在完成 |
| `2` | 已完成 |

## 接口

### 创建 TODO

```http
POST /api/v1/todos
```

请求体：

```json
{
  "text": "整理桌面",
  "status": 0
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `text` | string | 是 | TODO 内容，最多 50 个字符 |
| `status` | integer | 是 | 状态，只能是 `0`、`1`、`2` |

成功响应：`201 Created`

```json
{
  "id": 1,
  "owner_id": "usr_8f2c1a7b",
  "text": "整理桌面",
  "status": 0,
  "version": 1,
  "created_at": "2026-05-11T10:00:00Z",
  "updated_at": "2026-05-11T10:00:00Z"
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos -Method Post -Headers $headers -ContentType "application/json" -Body '{"text":"整理桌面","status":0}'
```

### 查询 TODO 列表

```http
GET /api/v1/todos
```

成功响应：`200 OK`

```json
[
  {
    "id": 1,
    "owner_id": "usr_8f2c1a7b",
    "text": "整理桌面",
    "status": 0,
    "version": 1,
    "created_at": "2026-05-11T10:00:00Z",
    "updated_at": "2026-05-11T10:00:00Z"
  }
]
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos -Headers $headers
```

### 查询单个 TODO

```http
GET /api/v1/todos/{id}
```

路径参数：

| 参数 | 说明 |
| --- | --- |
| `id` | TODO ID，必须是正整数 |

成功响应：`200 OK`

```json
{
  "id": 1,
  "owner_id": "usr_8f2c1a7b",
  "text": "整理桌面",
  "status": 0,
  "version": 1,
  "created_at": "2026-05-11T10:00:00Z",
  "updated_at": "2026-05-11T10:00:00Z"
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos/1 -Headers $headers
```

### 修改 TODO

```http
PATCH /api/v1/todos/{id}
```

请求体：

```json
{
  "version": 1,
  "text": "整理书桌",
  "status": 1
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `version` | integer | 是 | 客户端当前看到的版本号 |
| `text` | string | 否 | 新 TODO 内容，最多 50 个字符 |
| `status` | integer | 否 | 新状态，只能是 `0`、`1`、`2` |

`text` 和 `status` 至少要提供一个。修改成功后 `version` 会递增。

成功响应：`200 OK`

```json
{
  "id": 1,
  "owner_id": "usr_8f2c1a7b",
  "text": "整理书桌",
  "status": 1,
  "version": 2,
  "created_at": "2026-05-11T10:00:00Z",
  "updated_at": "2026-05-11T10:05:00Z"
}
```

版本冲突响应：`409 Conflict`

```json
{
  "current_version": 2,
  "error": "todo version conflict"
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos/1 -Method Patch -Headers $headers -ContentType "application/json" -Body '{"version":1,"status":1}'
```

### 删除 TODO

```http
DELETE /api/v1/todos/{id}
```

请求体：

```json
{
  "version": 2
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `version` | integer | 是 | 客户端当前看到的版本号 |

成功响应：`204 No Content`

版本冲突响应：`409 Conflict`

```json
{
  "error": "todo version conflict"
}
```

PowerShell 示例：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos/1 -Method Delete -Headers $headers -ContentType "application/json" -Body '{"version":2}'
```

## 客户端推荐流程

1. 调用 `GET /api/v1/todos` 获取列表。
2. 展示 `text` 和 `status`，并在本地保存每条 TODO 的 `id` 和 `version`。
3. 用户修改 TODO 时，调用 `PATCH /api/v1/todos/{id}`，请求体带上本地保存的 `version`。
4. 如果返回 `200`，用响应里的新 TODO 覆盖本地缓存。
5. 如果返回 `409`，重新调用 `GET /api/v1/todos/{id}` 或 `GET /api/v1/todos` 获取最新数据。
6. 用户删除 TODO 时，调用 `DELETE /api/v1/todos/{id}`，请求体带上本地保存的 `version`。

## 错误响应

错误响应通常为：

```json
{
  "error": "message"
}
```

常见状态码：

| 状态码 | 场景 |
| --- | --- |
| `400 Bad Request` | 请求体 JSON 错误、缺少 `text`、`text` 超过 50 个字符、`status` 非法、`version` 非法、ID 不是正整数 |
| `401 Unauthorized` | 请求没有携带正确用户 Token 或设备凭据 |
| `404 Not Found` | 指定的 TODO 不存在 |
| `409 Conflict` | 请求中的 `version` 不是当前版本 |
| `500 Internal Server Error` | 服务端存储失败 |

## 当前存储行为

当前实现使用 JSON 状态文件持久化：

- TODO 保存在 `todos` 中。
- 下一个 TODO ID 保存在 `next_todo_id` 中。
- TODO 会出现在用户侧 `GET /api/v1/snapshot` 和设备侧 `GET /api/v1/device/snapshot` 的 `todos` 字段中。

这套实现适合当前轻量服务端。后续如果 TODO 需要分页、排序或软删除，可以在不改变外部接口的前提下继续扩展存储层。
