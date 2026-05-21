# web

这里预留给后续前端 UI 项目。

建议开发方式：

- 前端源码放在 `web/`。
- 开发环境由前端 dev server 代理 `/api/v1` 到 Go 后端。
- 生产构建产物输出后，同步到 `internal/webui/dist/`，由 Go 服务嵌入并托管。
