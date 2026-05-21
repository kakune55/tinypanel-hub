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

用于用户侧一次性获取当前天气、最近消息和最近遥测数据。设备侧使用：

```http
GET /api/v1/device/snapshot
```

可以用 `include` 裁剪响应字段：

```http
GET /api/v1/snapshot?include=weather,messages,todos
```

### 天气

```http
GET /api/v1/weather
```

天气配置、缓存行为和响应样例见 [docs/weather.md](docs/weather.md)。

### 消息通知

Hub 使用 `User -> Device -> Message` 模型。用户绑定设备后，可以在 Hub 上向自己的设备发送消息；设备端轮询自己的收件箱，处理完成后 ack。

详细协议见 [docs/message.md](docs/message.md)。

接口：

```http
POST /api/v1/admin/users
POST /api/v1/device/hello
GET /api/v1/devices
POST /api/v1/devices/bind
POST /api/v1/devices/{device_id}/messages
GET /api/v1/device/messages?limit=10
POST /api/v1/device/messages/ack
```

创建开发用户：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
Invoke-RestMethod http://localhost:8080/api/v1/admin/users -Method Post -Headers $headers -ContentType "application/json" -Body '{"name":"Alice","api_token":"alice-token"}'
```

设备首次 hello：

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/device/hello -Method Post -Headers @{ "X-Device-ID" = "tinypanel-001" }
```

用户绑定设备并发送消息：

```powershell
$userHeaders = @{ Authorization = "Bearer alice-token" }
Invoke-RestMethod http://localhost:8080/api/v1/devices/bind -Method Post -Headers $userHeaders -ContentType "application/json" -Body '{"bind_code":"483921","name":"书桌屏幕"}'
Invoke-RestMethod http://localhost:8080/api/v1/devices/tinypanel-001/messages -Method Post -Headers $userHeaders -ContentType "application/json" -Body '{"body":"hello panel"}'
```

设备拉取并确认消息：

```powershell
$deviceHeaders = @{ "X-Device-ID" = "tinypanel-001"; "X-Device-Secret" = "device-secret" }
Invoke-RestMethod http://localhost:8080/api/v1/device/messages -Headers $deviceHeaders
Invoke-RestMethod http://localhost:8080/api/v1/device/messages/ack -Method Post -Headers $deviceHeaders -ContentType "application/json" -Body '{"message_ids":[1]}'
```

### TODO 列表

详细协议见 [docs/todolist.md](docs/todolist.md)。

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
GET /api/v1/devices/{device_id}/telemetry?limit=50
POST /api/v1/device/telemetry
POST /api/v1/device/telemetry/batch
```

当前遥测数据格式参考 `docs/遥测示例.json`。

上传示例：

```powershell
$headers = @{ Authorization = "Bearer change-me" }
Invoke-RestMethod http://localhost:8080/api/v1/device/telemetry -Method Post -Headers $deviceHeaders -ContentType "application/json" -Body (Get-Content docs/遥测示例.json -Raw)
```

## 项目结构

```text
cmd/tinypanel-hub/main.go       服务进程入口
internal/app/app.go             应用依赖组装
internal/config/config.go       JSON 配置加载
internal/domain/models.go       领域数据结构
internal/store/state.go         JSON 状态文件
internal/store/telemetry_log.go JSONL 遥测日志
internal/store/messages.go      设备消息存储逻辑
internal/store/devices.go       设备绑定存储逻辑
internal/store/users.go         用户存储逻辑
internal/store/todos.go         TODO 存储逻辑
internal/store/telemetry.go     遥测存储逻辑
internal/httpapi/server.go      HTTP Server 类型和入口
internal/httpapi/router.go      路由注册
internal/httpapi/auth.go        Token 鉴权
internal/httpapi/*.go           各资源 handler 和 HTTP 工具
internal/service/*.go           薄服务层和业务能力边界
internal/webui/                 嵌入式前端静态资源
web/                            后续前端 UI 项目目录
```

## 验证

```powershell
go test ./...
go build -o .\bin\tinypanel-hub.exe .\cmd\tinypanel-hub
```
