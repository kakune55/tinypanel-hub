# tinypanel-hub

`tinypanel-hub` 是一个给嵌入式桌面信息站使用的轻量 HTTP 服务端。

它的设计目标是保持简单、清楚、容易手写维护：

- 使用 Go 标准库 HTTP 服务。
- 不引入重型数据库，当前使用内存结构加 JSON 文件持久化。
- 提供天气、消息通知、遥测上报、面板快照等 API。
- 提供 TODO 列表 API，支持客户端增删改查和版本 CAS。
- 代码按职责拆分，避免把所有逻辑塞进一个包或一个文件。

## 运行

```powershell
go run ./cmd/tinypanel-hub
```

默认监听地址：`:8080`

默认状态文件：`data/tinypanel.json`

默认遥测日志：`data/telemetry.jsonl`

## 配置

服务使用 JSON 配置文件。默认配置路径是 `etc/config.json`。

创建本地配置：

```powershell
Copy-Item etc/config.example.json etc/config.json
go run ./cmd/tinypanel-hub
```

也可以指定其他配置文件：

```powershell
go run ./cmd/tinypanel-hub -config .\etc\config.example.json
```

或者通过环境变量指定配置路径：

```powershell
$env:TINYPANEL_CONFIG=".\etc\config.example.json"
go run ./cmd/tinypanel-hub
```

配置示例：

```json
{
  "server": {
    "addr": ":8080",
    "api_token": "change-me"
  },
  "storage": {
    "data_file": "data/tinypanel.json",
    "telemetry_file": "data/telemetry.jsonl"
  },
  "weather": {
    "provider": "qweather",
    "api_host": "your-api-host.qweatherapi.com",
    "api_key": "your-api-key",
    "bearer_token": "",
    "location": "101020100",
    "lang": "zh",
    "unit": "m",
    "hours": "24h",
    "days": "3d",
    "cache_ttl": "10m",
    "timeout": "5s"
  }
}
```

`etc/config.json` 已加入 `.gitignore`，可以放本地真实 token，也方便 Docker 挂载配置目录。仓库里保留 `etc/config.example.json` 作为模板。

天气配置里 `provider` 设为 `qweather` 时，服务会在 `GET /api/v1/weather` 或 `GET /api/v1/snapshot` 被请求时懒加载和风天气实时天气接口，并按 `cache_ttl` 缓存结果，默认 10 分钟。`api_host` 使用和风天气控制台中的 API Host；认证可二选一配置 `api_key` 或 `bearer_token`。`location` 支持 LocationID 或经纬度，例如 `101020100` 或 `121.47,31.23`。

存储配置里 `data_file` 保存当前服务状态，包括天气缓存、消息和 ack 状态；`telemetry_file` 使用 JSONL，每行一条遥测记录，适合持续追加写入和后续流式处理。

## 鉴权

当 `server.api_token` 不为空时，所有 `/api/v1/` 接口都需要 Token。

`/healthz` 不需要鉴权，方便探活。

Token 可以通过任意一种 header 传递：

```http
Authorization: Bearer change-me
X-API-Token: change-me
```

## API

### 健康检查

```http
GET /healthz
```

### 面板快照

```http
GET /api/v1/snapshot
```

用于客户端一次性获取当前天气、最近消息和最近遥测数据。

### 天气

```http
GET /api/v1/weather
```

天气配置、缓存行为和响应样例见 [docs/weather.md](docs/weather.md)。

### 消息通知

服务端可以把消息推送到某个频道。设备端轮询订阅端点，只拿未读数量和消息 ID；再按 ID 拉取消息内容；处理完成后 ack，之后该消息不会再出现在该设备的未读列表里。

详细协议见 [docs/message.md](docs/message.md)。

接口：

```http
GET /api/v1/messages?limit=20
POST /api/v1/messages
GET /api/v1/messages/{id}
POST /api/v1/messages/{id}/ack
GET /api/v1/subscriptions/{channel}?device_id=tinypanel-001
```

推送消息到频道：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
Invoke-RestMethod http://localhost:8080/api/v1/messages -Method Post -Headers $headers -ContentType "application/json" -Body '{"channel":"desk","author":"hub","body":"hello panel"}'
```

设备轮询未读消息 ID：

```powershell
Invoke-RestMethod "http://localhost:8080/api/v1/subscriptions/desk?device_id=tinypanel-001" -Headers $headers
```

返回示例：

```json
{
  "device_id": "tinypanel-001",
  "channel": "desk",
  "unread_count": 1,
  "message_ids": [1]
}
```

拉取并确认消息：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/messages/1 -Headers $headers
Invoke-RestMethod http://localhost:8080/api/v1/messages/1/ack -Method Post -Headers $headers -ContentType "application/json" -Body '{"device_id":"tinypanel-001"}'
```

### TODO 列表

详细协议见 [docs/todo.md](docs/todo.md)。

```http
GET /api/v1/todos
POST /api/v1/todos
GET /api/v1/todos/{id}
PATCH /api/v1/todos/{id}
DELETE /api/v1/todos/{id}
```

TODO 状态使用数字：`0` 未完成，`1` 正在完成，`2` 已完成。`text` 最多 50 个字符。

修改和删除使用 CAS：客户端必须提交当前 `version`，版本不匹配时返回 `409`。

创建 TODO：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos -Method Post -Headers $headers -ContentType "application/json" -Body '{"text":"整理桌面","status":0}'
```

修改状态：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos/1 -Method Patch -Headers $headers -ContentType "application/json" -Body '{"version":1,"status":1}'
```

删除 TODO：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/todos/1 -Method Delete -Headers $headers -ContentType "application/json" -Body '{"version":2}'
```

### 遥测

```http
GET /api/v1/telemetry?limit=50
POST /api/v1/telemetry
```

当前遥测数据格式参考 `docs/遥测示例.json`。

上传示例：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
Invoke-RestMethod http://localhost:8080/api/v1/telemetry -Method Post -Headers $headers -ContentType "application/json" -Body (Get-Content docs/遥测示例.json -Raw)
```

## 项目结构

```text
cmd/tinypanel-hub/main.go       服务进程入口
internal/config/config.go       JSON 配置加载
internal/domain/models.go       领域数据结构
internal/store/state.go         JSON 状态文件
internal/store/telemetry_log.go JSONL 遥测日志
internal/store/messages.go      消息和订阅存储逻辑
internal/store/todos.go         TODO 存储逻辑
internal/store/telemetry.go     遥测存储逻辑
internal/httpapi/server.go      HTTP Server 类型和入口
internal/httpapi/router.go      路由注册
internal/httpapi/auth.go        Token 鉴权
internal/httpapi/*.go           各资源 handler 和 HTTP 工具
```

## 验证

```powershell
go test ./...
go build -o .\bin\tinypanel-hub.exe .\cmd\tinypanel-hub
```
