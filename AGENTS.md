# AGENTS.md

本文档适用于整个仓库。若子目录中存在更具体的 `AGENTS.md`，以离目标文件最近的说明为准。

## 项目概览

DMP（Don't Starve Together Management Platform）是一个饥荒联机版服务器管理平台。本仓库包含 Go 后端，以及由独立前端仓库构建后嵌入二进制的静态资源；同一份代码用 `lite` 构建标签还会产出精简版后端，其前端以磁盘目录分发。

- Go 工具链：`go.mod` 指定 Go 1.27.3；模块路径为 `dst-management-platform-api`。升级 Go 版本时同步 `go.mod`、`.github/workflows/go.yml` 与 `docker/Dockerfile` 中的 `golang` 版本。
- 主程序技术栈：HTTP 用 Gin，数据访问用 GORM + SQLite，任务调度用 gocron，日志用 Zap。
- 仓库用构建标签产出两个可执行文件：
  - 主程序 `main.go`（`//go:build !lite`）调用 `server.Run()`，功能完整。
  - 精简版 `lite.go`（`//go:build lite`）调用 `lite.Run()`，只用标准库 `net/http` 和 JSON 单文件数据库，不引入 Gin/GORM/Zap，目标内存 8-10MB。
- `main.go` 安装 panic 捕获与日志刷盘钩子（`logger.HandlePanic`、`logger.Sync`）后调用 `server.Run()`；启动参数在 `server/flags.go`，控制台命令在 `server/console.go`，初始化、依赖装配和路由注册位于 `server/`。`lite/` 有自己独立的启动参数、路由和工具包，不复用这些包。
- API 版本取自 `constant.ApiVersion`，平台版本取自 `constant/system.go` 中的 `constant.Version`；发版时更新该常量，推送 `v*` tag 会触发 `.github/workflows/go.yml` 构建二进制与镜像。`lite/api` 使用自己的 `apiPrefix`（当前 `/v3`），改接口版本时两处都要同步。
- 运行环境主要面向 Linux，也支持 macOS；依赖 `screen`、SteamCMD、DST 文件与进程等外部资源。
- 前端由独立仓库构建后复制到 `embedFS/dist/`，再通过 `go:embed` 嵌入后端二进制；精简版前端则从磁盘目录 `dist-lite/` 提供。

## 目录职责

- `app/<module>/`：HTTP 业务模块，通常包含 `handler.go`、`router.go`、`i18n.go` 和 `utils.go`。
- `lite/`：精简版独立实现；`lite/api/` 是 net/http 路由与 handler，`lite/utils/` 是 JSON 单文件数据库、DST 文件与进程控制、JWT、密码和日志。它不 import 主程序的 `app/`、`server/`、`database/`、`dst/`、`scheduler/`、`i18n/`、`logger/`、`middleware/`、`webhook/`。`lite/dist-lite/` 是外部前端产物，运行时从可执行文件同级目录读取，不通过 `go:embed` 打进二进制。
- `aichat/`：游戏内 AI 对话监听、关键词/向量索引、知识库（wiki）解析与向量模型配置。
- `cache/`：进程级缓存初始化与访问。
- `constant/`：编译期常量；版本号、超时、JWT 与静态缓存等平台级常量在 `constant/system.go`，DST 路径、外部接口与插件目录在 `constant/dst.go`。
- `i18n/`：全局基础文案与 `BaseI18n`/`ExtendedI18n` 实现；各业务模块的 `i18n.go` 基于它合并本模块词条。
- `database/models/`：GORM 模型；`database/dao/`：数据访问；`database/db/`：SQLite 初始化、迁移和缓存。
- `dst/`：DST 房间、世界、模组、玩家、地图、日志和进程控制。
- `scheduler/`：全局及房间级定时任务。
- `middleware/`：JWT、权限、缓存和限流中间件。
- `webhook/`：Webhook 事件、签名和异步发送。
- `utils/`：共享工具函数（路径、命令、安全校验、进程内存 `GetRES` 等）；Klei TEX（KTEX）图集解包在 `utils/tex.go`，供 TMI 插件安装和地图资源提取使用，只包含解包链路，不含打包/压缩。
- `logger/`：Zap 日志、访问日志、panic 捕获与运行时崩溃输出。
- `embedFS/`：通过 `go:embed` 打包的前端、LuaJIT 库和安装脚本。
- `docs/`：项目文档及图片资源。
- `.agents/skills/`：本仓库的 agent 技能说明（`frontend-project`、`no-test`、`sandbox`、`zh-cn`），按任务类型加载。
- `.github/`：Issue 模板与发布流水线；`workflows/go.yml` 在 `v*` tag 上构建 Linux/macOS 主程序与 `dmp-lite.tgz`（含 `dmp-lite`、`lite/dist-lite`、手动安装/更新脚本），并推送 Docker 镜像（镜像只包含主程序）。
- `docker/`、`run.sh`：发布和最终用户部署，不是日常开发入口。

## 常用命令

在仓库根目录执行：

```bash
# 格式化修改过的 Go 文件
gofmt -w path/to/changed.go

# 静态检查
go vet ./...

# Linux 发布构建（无 CGO）
CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# Make 构建后端；所有平台均为 CGO_ENABLED=0
# macOS 输出到 ~/dmp，Linux 输出到仓库根目录 ./dmp
make backend-only

# 构建精简版后端（-tags=lite，输出 ~/dmp-lite）
make lite

# 直接构建精简版
CGO_ENABLED=0 go build -tags=lite -ldflags '-s -w' -v -o dmp-lite

# 查看 Makefile 提供的目标
make help
```

`make all`、`make frontend-only`、`make frontend2backend`、`make copy-frontend` 以及 `make lite-frontend-only`、`make lite-copy-frontend`、`make lite-frontend2backend` 假定前端仓库位于 `$(HOME)/WebstormProjects/dst-management-platform-web`，并可能清空或替换前端 `dist/`、`dist-lite/` 及 `embedFS/dist/`、`lite/dist-lite/`。只有任务明确涉及前端构建产物、且该外部仓库存在时才运行；不要手工编辑 `embedFS/dist/`、`lite/dist-lite/` 中带哈希的资源。

沙箱环境下若构建因无法写入系统 Go 缓存而失败，把缓存指向 `/tmp`，例如 `GOCACHE=/tmp/gocache go vet ./...`、`GOCACHE=/tmp/gocache go build ...`（见 `.agents/skills/sandbox/SKILL.md`）。Windows 下 `/tmp` 不可用，改为指向工作区内的临时目录（如 `.gocache`，验证完删除），因为沙箱只允许写工作区。

如确需本地启动，使用非特权端口和隔离数据目录，例如：

```bash
go run . -bind 8080 -dbpath ./data -level debug
```

启动服务会创建数据库、日志、`dmp_files/` 和手动安装脚本，并启动可能访问网络或操作 DST 进程的调度任务。不要把启动服务作为普通代码修改的默认验证步骤。

可用启动参数包括 `-bind`、`-dbpath`、`-level`、`-cert`、`-key`、`-no-http-compress`、`-v`、`-pprof` 和 `-console`。`-cert` 与 `-key` 同时提供时启动 HTTPS；`-level debug` 只调整日志等级与 Gin 模式；`-pprof` 单独开启 pprof 路由，会暴露运行数据，不应在生产环境启用；`-console` 执行一次控制台命令（`reset_password`、`list_user`、`db_stats`、`help`）后退出。

精简版只有 `-bind` 一个参数（默认 `9999`），启动时从可执行文件同级的 `dist-lite/` 读前端，数据库写到工作目录的 `dmp-lite.sdb`，集群目录固定为 `~/.klei/DoNotStarveTogether/Cluster_1`；启动同样会操作 `screen` 与 DST 进程，验证时按主程序的要求对待。

## 修改规则

### Go 代码

- 遵循 `.editorconfig`，Go 文件使用 UTF-8、LF 和 Tab；所有修改过的 Go 文件必须通过 `gofmt`。
- 沿用现有包边界和构造方式。业务依赖通过模块 `Handler` 持有，并由 `server/server.go` 统一创建和注入。
- 主程序的编译期常量统一放在 `constant/`：DST 路径、外部接口地址与插件目录放 `constant/dst.go`，版本号、超时、JWT 与静态资源缓存等平台级常量放 `constant/system.go`；不要在 `utils/` 中新增常量。精简版的常量留在 `lite/` 内，不回引主线常量。
- 主程序新增路由时使用 `constant.ApiVersion`，并按相邻路由应用 `middleware.TokenCheck()`、`middleware.AdminOnly()` 等中间件；精简版路由使用 `lite/api/router.go` 的 `apiPrefix`。
- JSON API 保持现有响应形状：`{"code": ..., "message": ..., "data": ...}`。多数业务接口以 HTTP 200 返回，业务结果由响应体中的 `code` 表示；文件下载、流式响应和 WebSocket 等特殊接口沿用各自现有行为。
- 主程序面向用户的消息统一走 `i18n` 包：通用文案维护在 `i18n/i18n.go` 的全局 `I18n` 中，模块词条在 `app/<module>/i18n.go` 用 `i18n.NewExtendedI18n(map[string]i18n.Text{...})` 声明，读取使用模块内的 `message.Get(c, ...)` / `message.GetF(c, ...)`；`zh`、`en` 文案必须同时维护。精简版只有中文常量文案，见下节。
- 主程序使用已有 `logger`、DAO 和工具函数，不引入平行的日志、数据库或配置体系；精简版按 `lite/` 内的独立实现，不要反向把它接回主程序栈。
- 错误必须被返回、转换为 API 响应或记录；不要静默吞掉对业务正确性有影响的错误。

### Lite 精简版

- 精简版与主程序共享模块路径，但包完全独立。给 `lite/` 加功能时不要 import 主程序的 `app/`、`server/`、`database/`、`dst/`、`scheduler/`、`i18n/`、`logger/`、`middleware/`、`webhook/`；确需复用的逻辑复制到 `lite/utils/`，或抽到不引入 Gin/GORM/Zap 的公共位置。
- 同一份外部契约存在两套实现（主程序 `app/` 与 `lite/api/`）。修改登录/JWT、房间与世界配置、`cluster.ini`/`server.ini`/`modoverrides.lua` 生成、日志读取或前端字段时，先判断精简版是否也要同步，并在交付说明中写明。
- `lite/utils/db.go` 用 `DBMu` 保护整个数据库：文件是 `./dmp-lite.sdb`，内容为 Base64(JSON)，整体读写并通过临时文件原子替换（权限 0600）。写接口必须包在 `withWriteLock` 中，它依靠 `db` 结构可比较来判断是否需要落盘；新增 `map`/`slice` 字段时要改成显式脏标记或序列化比较。
- `sync.RWMutex` 不可重入：`withWriteLock`/`withReadLock` 的 handler 内不能再调用 `DB.Read()` 或自行加锁；handler 内也不要做长时间阻塞操作，否则会挡住其它写请求。
- 精简版只提供中文文案（`lite/api/response.go` 的常量），不接 `i18n`。新增接口复用 `writeJSON`，保持 `{"code","message","data"}` 结构和 HTTP 200 的约定。
- `lite/api/static.go` 从可执行文件同级的 `dist-lite/` 提供前端：路径先 `path.Clean` 再映射磁盘，`/v3` 不参与前端 `index.html` 回退，目录不提供列表。改动静态路由时保留这些防护。
- 世界名在精简版中同样是 screen 会话名（`DMP_LITE_` 前缀）和集群子目录名；日志接口只接受数据库中已配置的世界名，新增路径拼接继续走 `utils.WorldLogPath`/`WorldChatLogPath` 的单级名校验。
- 精简版刻意控制内存：日志用 `ReadLastLines` 尾读并限制单次读取量，静态资源走磁盘而非 `go:embed`，日志用标准输出而非 Zap。新代码不要引入整文件读取、全量索引或重依赖。
- `update` 命令调用工作目录的 `manual_update.sh`；调整安装/更新脚本或 lite 打包内容时，同步 `.github/workflows/go.yml` 的 `build-lite` 步骤。

### 数据库与调度器

- 模型字段变更需要检查 JSON 标签、GORM 标签、零值语义及现有数据兼容性。
- 新增模型时同步新增 DAO，并将模型加入 `database/db/database.go` 的 `AllTables`，使 `AutoMigrate` 能创建表。
- 不要在 Handler 中绕过 DAO 复制已有查询逻辑；跨实体操作参考 `database/dao/composite.go`。
- 修改房间状态或配置时，检查是否需要同步更新 `scheduler/` 中的任务。任务名称和房间级任务生命周期应保持现有约定。
- `RoomDAO.UpdateConfiguration` 与 `WorldDAO.UpdateWorlds` 都是先删除该房间全部世界再插入。客户端提交或存档导入的世界不带 ID，重建后世界 ID 会变化；`cache.WorldMetrics` 按世界 ID 键控，改动这类调用点时要同步迁移或清理监控数据（参考 `app/room/utils.go` 的 `migrateWorldMetrics` / `deleteWorldMetrics`）。
- 修改 AI 模型配置（`database/models/roomAISetting.go`、`aichat/config.go`）时注意零值语义，例如 `EmbeddingDimensions` 为 `0` 表示不发送 `dimensions` 参数而非非法值；校验规则需与前端契约保持一致。

### 安全与外部操作

- 所有文件路径、命令参数、Webhook URL、上传内容和游戏配置都视为不可信输入，复用 `utils/` 中已有的路径、URL、XSS 和命令安全校验。
- 拼接 shell 命令时，对每个来自数据库、存档或请求的字符串使用 `utils.ShellQuote`（或改用 `exec.Command` 的 argv 形式），不要把变量直接插进命令字符串；`utils.IsSafeString` 用于世界名这类会同时进入命令与文件路径的标识符。
- 不削弱 JWT token 版本校验、管理员权限、登录限流或 Webhook HMAC 签名。
- JWT 密钥必须用 `crypto/rand` 生成（`constant.JwtSecretBytes` 字节后 Base64 RawURL 编码），`GenerateJWTSecret` 的错误要向上返回；不要退回 `math/rand`，两套实现的密钥格式与 `JwtExpirationHours` 保持同步。
- 涉及 shell、`screen`、SteamCMD、DST 存档或备份的改动，必须考虑路径转义、幂等性、部分失败和清理行为。
- 测试不得依赖或修改真实的 `~/.klei/DoNotStarveTogether`、生产数据库、运行中的 `screen` 会话或真实 DST 安装；优先使用临时目录、临时 SQLite 数据库和可替换的边界。

### DST 交互（命令、日志、存档）

- 世界名（`World.WorldName`）同时是存档目录名、`-shard` 参数和 screen 会话名的一部分：同一房间内必须唯一，且只允许安全字符；改动存档导入或房间保存逻辑时要保持这一约束。
- 在线玩家信息的采集依赖两处严格对应：`dst/world.go` 中写入游戏日志的 Lua `print` 格式（7 个字段，`<-@dmp@->` 分隔）与 `playerListPattern` 正则；`readPlayerListFromEnd` 的回读窗口必须能覆盖一次完整玩家列表的输出。

### 生成物与依赖

- `embedFS/dist/` 与 `lite/dist-lite/` 都是独立前端仓库的编译产物（后者由 `pnpm run build:lite` 生成）。不要手工编辑带哈希的 JS/CSS/图片；仅在任务明确要求更新前端产物时整体替换并说明来源。
- `embedFS/luajit/*.so` 是二进制依赖，不进行格式化或手工修改。
- 修改 `embedFS/shell/` 时要注意这些脚本会在运行时释放到工作目录。
- 不要为了整理而改写 `go.mod`/`go.sum`。仅在依赖确有变化时执行 `go mod tidy`，并审查依赖差异。
- 不提交本地运行产物，如 `dmp`、`dmp-lite`、`dmp-lite.sdb`、`data/`、`logs/`、`dmp_files/`、`.DS_Store` 或运行时生成的脚本。

### 测试与验证

- 当前仓库没有 `_test.go` 测试文件；遵循 `.agents/skills/no-test/SKILL.md`，不要为普通修改新增测试文件。
- 按需加载 `.agents/skills/` 中的其他技能：涉及前端逻辑用 `frontend-project`（只读前端源码，不得修改），沙箱内构建失败用 `sandbox`，回复语言与文案用 `zh-cn`。
- 优先执行与改动范围匹配的 `gofmt`、`go vet ./...` 或构建命令；需要 Linux、DST、网络或外部服务的行为无法验证时，在交付说明中明确指出。

## 验证要求

按改动范围执行最小充分验证：

1. 对所有修改过的 Go 文件运行 `gofmt`，并确认没有意外格式变化。
2. 修改共享工具、中间件、数据库、调度器或 API 契约时，运行 `go vet ./...`；涉及 `lite.go` 的构建标签时用 `go vet -tags=lite ./...` 才能覆盖该入口。
3. 修改入口、依赖、嵌入资源或发布相关代码时，在 Linux 上执行 `CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp`；macOS 执行 `make backend-only`（产物输出到 `~/dmp`）。涉及 `lite/`、构建标签或发布流水线时，再执行 `CGO_ENABLED=0 go build -tags=lite -ldflags '-s -w' -v -o dmp-lite`（或 `make lite`），并确认 `build-lite` 打包仍包含 `dmp-lite`、`dist-lite` 与脚本。验证后不要提交生成的 `dmp`/`dmp-lite`。
4. 无法执行依赖 Linux/DST/网络的验证时，在交付说明中明确列出未验证项和原因。

## Agent 工作方式

- 修改前先读取相关模块、检查 `git status`，保留用户已有改动，不覆盖或回退无关文件。
- 只修改完成任务所需的文件；不要顺带重构、批量改名或更新生成资源。
- 优先使用仓库已有模式。只有在确实减少复杂度或重复时才增加抽象。
- 不执行破坏性 Git 命令，不删除用户数据，不启动、停止或重置真实 DST 服务，除非用户明确要求并确认目标。
- 不主动创建提交、标签、发布或推送远端；只有用户明确要求时才执行。
- 交付时简要说明修改内容、已运行的验证以及仍存在的环境限制。
