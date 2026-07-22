# AGENTS.md

本文档适用于整个仓库。若子目录中存在更具体的 `AGENTS.md`，以离目标文件最近的说明为准。

## 项目概览

DMP（Don't Starve Together Management Platform）是一个饥荒联机版服务器管理平台。本仓库包含 Go 后端，以及由独立前端仓库构建后嵌入二进制的静态资源。

- Go 工具链：`go.mod` 指定 Go 1.25.4。
- HTTP：Gin；数据访问：GORM + SQLite；任务调度：gocron；日志：Zap。
- `main.go` 仅调用 `server.Run()`；初始化、依赖装配和路由注册位于 `server/`。
- API 版本取自 `utils.ApiVersion`，平台版本取自 `utils.Version`。
- 运行环境主要面向 Linux，并依赖 `screen`、SteamCMD、DST 文件与进程等外部资源。

## 目录职责

- `app/<module>/`：HTTP 业务模块，通常包含 `handler.go`、`router.go`、`i18n.go` 和 `utils.go`。
- `database/models/`：GORM 模型；`database/dao/`：数据访问；`database/db/`：SQLite 初始化、迁移和缓存。
- `dst/`：DST 房间、世界、模组、玩家、地图、日志和进程控制。
- `scheduler/`：全局及房间级定时任务。
- `middleware/`：JWT、权限、缓存和限流中间件。
- `webhook/`：Webhook 事件、签名和异步发送。
- `utils/`、`logger/`：共享工具与日志设施。
- `embedFS/`：通过 `go:embed` 打包的前端、LuaJIT 库和安装脚本。
- `docker/`、`run.sh`：发布和最终用户部署，不是日常开发入口。

## 常用命令

在仓库根目录执行：

```bash
# 格式化修改过的 Go 文件
gofmt -w path/to/changed.go

# 静态检查
go vet ./...

# 与发布工作流一致的后端构建
CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# 等价的 Make 目标
make backend
```

`make all` 和 `make copy-frontend` 假定前端仓库位于 `$(HOME)/WebstormProjects/dst-management-platform-web`，并会清空再替换 `embedFS/dist/`。只有任务明确涉及前端构建产物、且该外部仓库存在时才运行。

如确需本地启动，使用非特权端口和隔离数据目录，例如：

```bash
go run . -bind 8080 -dbpath ./data -level debug
```

启动服务会创建数据库、日志、`dmp_files/` 和手动安装脚本，并启动可能访问网络或操作 DST 进程的调度任务。不要把启动服务作为普通代码修改的默认验证步骤。

## 修改规则

### Go 代码

- 遵循 `.editorconfig`，Go 文件使用 UTF-8、LF 和 Tab；所有修改过的 Go 文件必须通过 `gofmt`。
- 沿用现有包边界和构造方式。业务依赖通过模块 `Handler` 持有，并由 `server/server.go` 统一创建和注入。
- 新增路由时使用 `utils.ApiVersion`，并按相邻路由应用 `middleware.TokenCheck()`、`middleware.AdminOnly()` 等中间件。
- JSON API 保持现有响应形状：`{"code": ..., "message": ..., "data": ...}`。多数业务接口以 HTTP 200 返回，业务结果由响应体中的 `code` 表示；文件下载、流式响应和 WebSocket 等特殊接口沿用各自现有行为。
- 面向用户的消息应接入对应模块的 `i18n.go`，同时维护 `zh`、`en` 文案，并通过现有 `BaseI18n`/`message.Get` 模式读取。
- 使用已有 `logger`、DAO 和工具函数，不引入平行的日志、数据库或配置体系。
- 错误必须被返回、转换为 API 响应或记录；不要静默吞掉对业务正确性有影响的错误。

### 数据库与调度器

- 模型字段变更需要检查 JSON 标签、GORM 标签、零值语义及现有数据兼容性。
- 新增模型时同步新增 DAO，并将模型加入 `database/db/database.go` 的 `AllTables`，使 `AutoMigrate` 能创建表。
- 不要在 Handler 中绕过 DAO 复制已有查询逻辑；跨实体操作参考 `database/dao/composite.go`。
- 修改房间状态或配置时，检查是否需要同步更新 `scheduler/` 中的任务。任务名称和房间级任务生命周期应保持现有约定。

### 安全与外部操作

- 所有文件路径、命令参数、Webhook URL、上传内容和游戏配置都视为不可信输入，复用 `utils/` 中已有的路径、URL、XSS 和命令安全校验。
- 不削弱 JWT token 版本校验、管理员权限、登录限流或 Webhook HMAC 签名。
- 涉及 shell、`screen`、SteamCMD、DST 存档或备份的改动，必须考虑路径转义、幂等性、部分失败和清理行为。
- 测试不得依赖或修改真实的 `~/.klei/DoNotStarveTogether`、生产数据库、运行中的 `screen` 会话或真实 DST 安装；优先使用临时目录、临时 SQLite 数据库和可替换的边界。

### 生成物与依赖

- `embedFS/dist/` 是独立前端仓库的编译产物。不要手工编辑带哈希的 JS/CSS/图片；仅在任务明确要求更新前端产物时整体替换并说明来源。
- `embedFS/luajit/*.so` 是二进制依赖，不进行格式化或手工修改。
- 修改 `embedFS/shell/` 时要注意这些脚本会在运行时释放到工作目录。
- 不要为了整理而改写 `go.mod`/`go.sum`。仅在依赖确有变化时执行 `go mod tidy`，并审查依赖差异。
- 不提交本地运行产物，如 `dmp`、`data/`、`logs/`、`dmp_files/` 或运行时生成的脚本。

## 验证要求

按改动范围执行最小充分验证：

1. 对所有修改过的 Go 文件运行 `gofmt`，并确认没有意外格式变化。
2. 修改共享工具、中间件、数据库、调度器或 API 契约时，运行 `go vet ./...`。
3. 修改入口、依赖、嵌入资源或发布相关代码时，执行 `CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp`；验证后不要提交生成的 `dmp`。
4. 无法执行依赖 Linux/DST/网络的验证时，在交付说明中明确列出未验证项和原因。

## Agent 工作方式

- 修改前先读取相关模块、检查 `git status`，保留用户已有改动，不覆盖或回退无关文件。
- 只修改完成任务所需的文件；不要顺带重构、批量改名或更新生成资源。
- 优先使用仓库已有模式。只有在确实减少复杂度或重复时才增加抽象。
- 不执行破坏性 Git 命令，不删除用户数据，不启动、停止或重置真实 DST 服务，除非用户明确要求并确认目标。
- 不主动创建提交、标签、发布或推送远端；只有用户明确要求时才执行。
- 交付时简要说明修改内容、已运行的验证以及仍存在的环境限制。
