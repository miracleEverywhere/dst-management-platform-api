# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

DMP (Don't Starve Together Management Platform) 是一个饥荒联机版服务器管理平台，提供 Web UI 进行多房间、多世界、模组、玩家、备份等管理。Go 后端 + 嵌入式前端单二进制部署。

## 构建和运行

```bash
# 构建后端（单二进制，含嵌入式前端）
CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# 全量构建（先构建前端，复制产物到 embedFS/dist，再构建后端）
# 前端项目需位于 $HOME/WebstormProjects/dst-management-platform-web
make all

# 仅复制前端产物到 embedFS（不重新构建前端）
make copy-frontend

# 运行
./dmp -bind 80 -dbpath ./data -level info

# TLS 运行
./dmp -bind 443 -cert /path/to/fullchain.pem -key /path/to/privkey.pem

# 查看版本
./dmp -v

# 控制台命令
./dmp -console reset_password -dbpath ./data   # 重置用户密码
./dmp -console list_user -dbpath ./data        # 列出所有用户
./dmp -console db_stats -dbpath ./data         # 查看数据库统计
```

启动参数：`-bind`（端口，默认 80）、`-dbpath`（数据库目录，默认 `./data`）、`-level`（日志等级：debug/info/warn/error，默认 info）、`-cert`/`-key`（TLS 证书/私钥，不填则 HTTP）。

`run.sh` 是一键部署脚本，提供下载、启动、更新、虚拟内存设置、开机自启等功能，面向最终用户而非开发。

## 技术栈

- **Go 1.25** + **Gin** HTTP 框架
- **GORM** + **SQLite** (github.com/glebarez/sqlite, WAL 模式)
- **gocron** 定时任务调度
- **Zap** (go.uber.org/zap) 结构化日志
- **JWT** (golang-jwt/jwt/v5) 认证，HS256 签名
- **gopher-lua** 解析 DST 存档元数据
- **melody** WebSocket 支持
- 前端为独立的 Vue/Vuetify 项目，构建产物复制到 `embedFS/dist/` 由 Go embed 嵌入

## 项目结构

```
main.go              → 入口，调用 server.Run()
server/
  server.go          → 核心启动流程：解析参数 → 初始化日志/数据库/DAO → 启动调度器 → 注册路由 → 启动 HTTP(S)
  flags.go           → CLI 参数定义（-bind, -dbpath, -level, -cert, -key, -v, -console）
  console.go         → 控制台命令（reset_password, list_user, db_stats）
app/                 → 业务模块，每模块包含 handler.go + router.go + i18n.go + utils.go (Handler 结构体模式)
  user/              → 用户管理
  room/              → 房间/WORLD管理（创建、修改、启停、删除、上传存档）
  dashboard/         → 控制面板（房间状态、系统监控）
  platform/          → 平台管理（全局设置、系统信息）
  mod/               → 模组管理（下载、启用、配置）
  player/            → 玩家管理（名单、在线统计、UidMap）
  tools/             → 工具（WebSocket 终端等）
  logs/              → 日志查看
database/
  models/            → GORM 模型：User, Room, World, RoomSetting, GlobalSetting, System, UidMap
  dao/               → 泛型 BaseDAO[T] + 各模型专用 DAO（含复合查询、关联检索）
  db/
    database.go      → SQLite 初始化、AutoMigrate、WAL 配置
    cache.go         → 全局内存缓存：JWT 密钥、token 版本号、玩家统计、系统指标等
    token_version.go → Token 版本号缓存管理（用于 token 撤销）
dst/                 → 游戏控制器（Game struct），管理 DST 进程、配置文件、模组、世界、备份等
  dst.go             → 公开方法（SaveAll, StartWorld, Backup, ConsoleCmd 等）
  room.go            → 房间级配置读写（cluster.ini）、存档备份/恢复
  world.go           → 世界启停（screen 进程管理）
  mod.go             → 模组下载/启用/禁用/配置
  player.go          → 在线玩家列表（screen 命令交互）、名单管理
  map.go             → 地图生成
  logs.go            → 游戏日志读取
  utils.go           → 路径构建、screen 名称规范
scheduler/           → 基于 gocron 的定时任务系统
  init.go            → Start() 入口、UpdateJob/DeleteJob、按房间ID/类型查询任务
  jobs.go            → initJobs()：读取全局设置和房间设置，生成所有 JobConfig
  global.go           → 全局任务具体实现（在线玩家统计、系统指标、游戏更新检测、公网IP、模组清理）
  room.go            → 房间任务实现（备份/清理/重启/重置/定时启停/保活/通知）
  utils.go           → DST 版本检测、公网IP 获取
middleware/           → TokenCheck (JWT验证+自动刷新)、AdminOnly、CacheControl (静态资源缓存)、LoginRateLimit
webhook/              → 异步 webhook 通知：Events 常量、Sender.Send() fire-and-forget、HMAC-SHA256 签名
utils/                → 工具函数：JWT、bcrypt/SHA512 密码、i18n 基类、安全校验（webhook URL、路径穿越、XSS）
logger/               → Zap 日志初始化，输出到 logs/access.log 和 logs/runtime.log
embedFS/              → 嵌入前端静态资源（go:embed dist/）
```

## 核心架构设计

### 路由注册模式
每个 app 模块定义 `Handler` 结构体，持有所需 DAO 的引用。`RegisterRoutes(*gin.Engine)` 方法将路由挂载到 `/v3/<module>` 路径组下。路由组的中间件为 `TokenCheck()`，管理员专属接口额外使用 `AdminOnly()`。

### 认证体系
- JWT token 通过 `X-DMP-TOKEN` 请求头传递
- Token 包含 username、nickname、role、tokenVersion
- Token 版本号机制支持撤销：RevokeTokenVersion 递增版本号 → 持久化到 DB → 旧 token 在 `ValidateTokenVersion` 缓存比对时失效
- 登录成功返回 token 的同时，SetTokenVersion 写入内存缓存
- Token 剩余有效期 < 总有效期一半时自动刷新，通过 `X-DMP-NEW-TOKEN` 响应头返回

### 国际化 (i18n)
通过请求头 `X-I18n-Lang`（zh/en）选择语言。每个 app 模块有自己的 i18n 字典（message 变量，由 `utils.BaseI18n.Get(c, key)` 驱动），全局公共消息在 `utils/i18n.go` 的 `I18n` 变量中。

### 数据库层
- `database/db/cache.go` 持有全局状态（JWT 密钥、玩家统计、系统指标）。JWT 密钥在首次初始化时随机生成并持久化到 `system` 表
- DAO 使用泛型 `BaseDAO[T]` 提供通用 CRUD 和分页查询，各模型专用 DAO 通过嵌入 BaseDAO 扩展特定查询方法

### 游戏管理
`dst.Game` 结构体是核心游戏控制器，封装了对 DST 服务器进程的所有操作：
- 通过 Linux `screen` 命令管理进程生命周期（启动/停止/检测状态）
- 配置以 INI 格式写入 `~/.klei/DoNotStarveTogether/Cluster_{roomID}/` 目录
- 世界操作通过向 screen session 发送 Lua 命令实现
- 模组管理涉及 Steam Workshop API、文件下载、配置解析

### 定时任务
- 全局任务：在线玩家统计、系统监控、游戏更新检测、公网IP
- 房间级任务：备份、备份清理、重启、重置、定时启停、保活、公告
- 任务名格式：`{roomID}-{suffix}`，通过前缀匹配按房间批量管理
- 房间启停会动态添加/移除对应的定时任务

### Webhook 系统
- 支持房间级和全局级 webhook，可配置多个 URL + 事件订阅 + 密钥签名
- `sender.Send()` 异步 fire-and-forget，签名使用 HMAC-SHA256 → `X-DMP-Signature` 请求头
- 全局 webhook 支持按 roomIds 过滤

### 安全措施
- Webhook URL：仅允许 http/https，禁止 query/fragment，防 SSRF
- 游戏模式：正则校验字符集防 XSS（自定义模式支持 eval 值）
- 路径操作：禁止 `..` 和 `~` 防目录穿越
- 登录接口：IP 级别 1 秒限流
- 密码支持 bcrypt 和 SHA-512 双版本兼容

### 代码规范
- 缩进使用 Tab（见 `.editorconfig`），Go 源码文件字符集 UTF-8，换行 LF
- 所有 API 响应格式：`{"code": 200, "message": "...", "data": ...}`，HTTP 状态码统一返回 200，业务状态通过 `code` 字段区分

### API 版本
当前 `utils.ApiVersion = "v3"`，所有 API 路径前缀为 `/v3/`。`utils.Version = "v3.1.5"`。
