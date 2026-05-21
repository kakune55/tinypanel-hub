# tinypanel-hub

`tinypanel-hub` 是一个给嵌入式桌面信息站使用的轻量 HTTP 服务端。

当前核心模型是：

```text
User -> Device -> Message
```

用户在 Hub 中绑定自己的设备，然后向设备发送消息；设备端通过自己的凭据拉取收件箱、确认消息、上报遥测。

## 设计目标

- 轻量：不依赖数据库，当前使用内存结构、JSON 状态文件和 JSONL 遥测日志。
- 清晰：HTTP 层、服务层、存储层分离，代码按职责组织。
- 可扩展：保留前端 UI 托管入口，后续可继续演进账号登录、设备管理和更强存储。
- 易部署：单个 Go 服务即可运行，配置文件为普通 JSON。

## 快速运行

```powershell
go run ./cmd/tinypanel-hub
```

默认值：

```text
监听地址：:8080
配置文件：etc/config.json
状态文件：data/tinypanel.json
遥测日志：data/telemetry.jsonl
```

创建本地配置：

```powershell
Copy-Item etc/config.example.json etc/config.json
go run ./cmd/tinypanel-hub
```

指定配置文件：

```powershell
go run ./cmd/tinypanel-hub -config .\etc\config.example.json
```

也可以使用环境变量：

```powershell
$env:TINYPANEL_CONFIG=".\etc\config.example.json"
go run ./cmd/tinypanel-hub
```

## 配置

配置示例：

```json
{
  "server": {
    "addr": ":8080",
    "api_token": "tk-123"
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

说明：

- `server.api_token` 只用于本地管理接口，例如创建开发用户。
- 用户 API 使用用户 token。
- 设备 API 使用 `X-Device-ID` 和 `X-Device-Secret`。
- `storage.data_file` 保存用户、设备、消息、TODO、天气等状态。
- `storage.telemetry_file` 使用 JSONL 追加保存遥测记录。
- 用户侧数据按当前用户隔离；设备侧数据按设备凭据隔离。
- 天气配置详见 [docs/weather.md](docs/weather.md)。

## 基本流程

### 创建开发用户

```powershell
$adminHeaders = @{ Authorization = "Bearer tk-123" }
Invoke-RestMethod http://localhost:8080/api/v1/admin/users `
  -Method Post `
  -Headers $adminHeaders `
  -ContentType "application/json" `
  -Body '{"name":"Alice","api_token":"alice-token"}'
```

### 设备首次注册

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/device/hello `
  -Method Post `
  -Headers @{ "X-Device-ID" = "tinypanel-001" }
```

首次响应会返回 `device_secret` 和 `bind_code`。设备应保存 `device_secret`，后续请求都要携带。

### 用户绑定设备

```powershell
$userHeaders = @{ Authorization = "Bearer alice-token" }
Invoke-RestMethod http://localhost:8080/api/v1/devices/bind `
  -Method Post `
  -Headers $userHeaders `
  -ContentType "application/json" `
  -Body '{"bind_code":"483921","name":"书桌屏幕"}'
```

### 用户发送消息

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/devices/tinypanel-001/messages `
  -Method Post `
  -Headers $userHeaders `
  -ContentType "application/json" `
  -Body '{"body":"hello panel","priority":"normal"}'
```

### 设备拉取并确认消息

```powershell
$deviceHeaders = @{
  "X-Device-ID" = "tinypanel-001"
  "X-Device-Secret" = "device-secret"
}

Invoke-RestMethod http://localhost:8080/api/v1/device/messages -Headers $deviceHeaders

Invoke-RestMethod http://localhost:8080/api/v1/device/messages/ack `
  -Method Post `
  -Headers $deviceHeaders `
  -ContentType "application/json" `
  -Body '{"message_ids":[1]}'
```

## API 文档

- 认证模型：[docs/auth.md](docs/auth.md)
- 设备绑定和消息：[docs/message.md](docs/message.md)
- 遥测：[docs/telemetry.md](docs/telemetry.md)
- 天气：[docs/weather.md](docs/weather.md)
- TODO：[docs/todolist.md](docs/todolist.md)

## API 分组

管理接口：

```http
POST /api/v1/admin/users
```

用户接口：

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

设备接口：

```http
POST /api/v1/device/hello
GET  /api/v1/device/messages
POST /api/v1/device/messages/ack
GET  /api/v1/device/todos
POST /api/v1/device/telemetry
POST /api/v1/device/telemetry/batch
GET  /api/v1/device/snapshot
```

公开接口：

```http
GET /healthz
```

## 项目结构

```text
cmd/tinypanel-hub/main.go       服务进程入口
internal/app/app.go             应用依赖组装
internal/config/config.go       JSON 配置加载
internal/domain/models.go       领域数据结构
internal/service/*.go           业务能力边界
internal/store/*.go             文件存储实现
internal/httpapi/*.go           HTTP 路由、中间件和 handler
internal/weather/*.go           天气供应商和缓存
internal/webui/                 嵌入式前端静态资源
web/                            后续前端 UI 项目目录
docs/                           详细协议文档
```

## 验证

```powershell
go test ./...
go build -o .\bin\tinypanel-hub.exe .\cmd\tinypanel-hub
```
