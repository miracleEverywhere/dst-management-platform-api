# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DST Management Platform API — a Go backend for managing Don't Starve Together (DST) dedicated game servers. It provides a REST API + embedded frontend for creating/configuring game rooms (clusters), managing worlds, mods, players, backups, scheduled tasks, and server monitoring.

## Build & Run Commands

**推荐在 Linux 服务器上执行 `go build`、`go run` 等编译命令。** 已使用纯 Go 实现的 SQLite 驱动（`github.com/glebarez/sqlite`），无需 CGO。本地可编译，但目标平台仍为 Linux 服务器，需交叉编译或部署后测试。

```bash
# Build (仅在 Linux 服务器上执行)
go build -ldflags '-s -w' -v -o dmp .

# Run (default: port 80, DB in ./data, log level info)
go run main.go
go run main.go -bind 8080 -dbpath ./data -level debug

# Show version
go run main.go -v

# Command-line flags
#   -bind    int    Port (default 80)
#   -dbpath  string Database directory (default "./data")
#   -level   string Log level: debug|info|warn|error (default "info")
#   -v       bool   Print version and exit
```

There are no tests or Makefile in this project.

## Architecture

```
main.go                    → Entry point, calls server.Run()
server/                    → Bootstrapping: flags, DB init, router setup, scheduler start
app/                       → HTTP handler modules (one sub-package per domain)
  ├── room/                → Room/cluster CRUD, activate/deactivate, backup/restore, upload
  ├── mod/                 → Mod management (download, enable/disable, configure)
  ├── player/              → Player lists (admin/whitelist/block), online players, statistics
  ├── dashboard/           → Dashboard data (system metrics, player stats)
  ├── platform/            → Platform-level ops (game install, update, system settings)
  ├── logs/                → Log viewing (realtime via WebSocket, history)
  ├── tools/               → Utilities (console commands, map generation)
  └── user/                → User auth (login, CRUD, JWT)
dst/                       → Core game server logic (Game struct, file I/O, screen processes)
database/
  ├── db/                  → SQLite connection, global cache variables (player stats, metrics)
  ├── dao/                 → Data access objects (generic BaseDAO[T] + specialized DAOs)
  └── models/              → GORM models: User, Room, World, RoomSetting, GlobalSetting, UidMap
middleware/                 → Gin middleware: TokenCheck (JWT), AdminOnly, CacheControl
scheduler/                 → Cron jobs: backup, restart, keepalive, player tracking, game updates
embedFS/                   → Embedded static assets (frontend dist/, luajit libs, shell scripts)
utils/                     → Shared utilities: JWT, crypto, i18n, system commands, file ops
logger/                    → Custom zap logger (runtime.log + access.log)
```

## Key Patterns

- **App module structure**: Each `app/*` package follows the same pattern: `handler.go` (request logic), `router.go` (route registration), `utils.go` (helpers, permission checks), `i18n.go` (Chinese/English messages). Routes are registered via `NewHandler(dependencies).RegisterRoutes(engine)`.

- **API structure**: All API routes are versioned under `/v3/` (see `utils.ApiVersion`). Authenticated routes use the `TokenCheck()` middleware. Admin-only routes additionally use `AdminOnly()`. Auth info (username, nickname, role) is passed via `gin.Context.Set()`.

- **DAO pattern**: `database/dao/base.go` provides a generic `BaseDAO[T]` with CRUD and pagination. Specialized DAOs (e.g., `RoomDAO`, `UserDAO`) wrap `BaseDAO` with domain-specific queries. DAOs are instantiated in `server.Run()` and injected into handlers.

- **Game controller**: `dst.Game` is the central struct for all game server operations. Created via `dst.NewGameController(room, worlds, setting, lang)`, it holds file paths, screen names, mutexes, and parsed data. It manages DST server processes via GNU `screen` sessions.

- **Lua parsing**: The `dst/` package uses `gopher-lua` to parse DST's Lua config files (`modinfo.lua`, `modoverrides.lua`, session `.meta` files). Key types: `ModInfoParser`, `ModORParser`, `AcfParser`.

- **Scheduled tasks**: `scheduler/` uses `gocron` for periodic jobs (backup, restart, keepalive, player statistics, game updates, announcements). Jobs are identified by `"{roomID}-{index}-{Type}"` naming. Jobs are initialized at startup and dynamically updated when room settings change.

- **i18n**: Responses are localized using `utils.I18n.Get(c, key)` and per-module i18n maps. Language is determined by the `X-I18n-Lang` request header (`zh` or `en`, default `zh`).

- **Authentication**: JWT tokens via `X-DMP-TOKEN` header. Tokens are auto-refreshed when remaining lifetime drops below half. Non-admin users have room-level permissions tracked via `user.Rooms` (comma-separated room IDs).

- **Embedded frontend**: Static files in `embedFS/dist/` are served via `gin-static`. The frontend is a separate build artifact committed to the repo.

- **In-memory cache**: `database/db/cache.go` holds global mutable state: `PlayersStatistic`, `PlayersOnlineTime`, `SystemMetrics`, `JwtSecret`, `InternetIP`. These are protected by mutexes.
