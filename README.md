# tinypanel-hub

`tinypanel-hub` 是一个给嵌入式桌面信息站使用的轻量 HTTP 服务端。

它的设计目标是保持简单、清楚、容易手写维护：

- 使用 Go 标准库 HTTP 服务。
- 不引入重型数据库，当前使用内存结构加 JSON 文件持久化。
- 提供天气、消息通知、遥测上报、面板快照等 API。
- 代码按职责拆分，避免把所有逻辑塞进一个包或一个文件。

## 运行

```powershell
go run ./cmd/tinypanel-hub
```

默认监听地址：`:8080`

默认数据文件：`data/tinypanel.json`

## 配置

服务使用 JSON 配置文件。默认配置路径是 `config.json`。

创建本地配置：

```powershell
Copy-Item config.example.json config.json
go run ./cmd/tinypanel-hub
```

也可以指定其他配置文件：

```powershell
go run ./cmd/tinypanel-hub -config .\config.example.json
```

或者通过环境变量指定配置路径：

```powershell
$env:TINYPANEL_CONFIG=".\config.example.json"
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
    "data_file": "data/tinypanel.json"
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

`config.json` 已加入 `.gitignore`，可以放本地真实 token。仓库里保留 `config.example.json` 作为模板。

天气配置里 `provider` 设为 `qweather` 时，服务会在 `GET /api/v1/weather` 或 `GET /api/v1/snapshot` 被请求时懒加载和风天气实时天气接口，并按 `cache_ttl` 缓存结果，默认 10 分钟。`api_host` 使用和风天气控制台中的 API Host；认证可二选一配置 `api_key` 或 `bearer_token`。`location` 支持 LocationID 或经纬度，例如 `101020100` 或 `121.47,31.23`。

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
cmd/tinypanel-hub/main.go   服务进程入口
internal/config/config.go   JSON 配置加载
internal/domain/models.go   领域数据结构
internal/store/jsonfile.go  内存结构 + JSON 文件存储
internal/httpapi/server.go  HTTP Server 类型和入口
internal/httpapi/router.go  路由注册
internal/httpapi/auth.go    Token 鉴权
internal/httpapi/*.go       各资源 handler 和 HTTP 工具
```

## 验证

```powershell
go test ./...
go build -o .\bin\tinypanel-hub.exe .\cmd\tinypanel-hub
```
