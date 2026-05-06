# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

DMP (Don't Starve Together Management Platform) is a Go web server that manages DST game servers. It provides a web UI for multi-room, multi-user game server administration — room creation, mod management, backup/restore, player tracking, and scheduled tasks (restart, keepalive, announcements).

## Build and run

```bash
# Build (static binary, no CGO)
CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# Run
./dmp -bind 80 -dbpath ./data -level info

# CLI flags
#   -bind    HTTP port (default: 80)
#   -dbpath  SQLite database directory (default: ./data)
#   -level   Log level: debug, info, warn, error (default: info)
#   -v       Print version and exit
```

There are no tests in this repository.

## Architecture

### Startup flow (`server/server.go`)

1. Parse CLI flags
2. Initialize zap logger (writes to `logs/access.log` and `logs/runtime.log`)
3. Extract embedded files (LuaJIT libs, shell scripts) from `embedFS/` to disk
4. Open SQLite database via GORM + glebarez/sqlite driver, auto-migrate all models
5. Start the gocron scheduler with global + per-room jobs
6. Register Gin routes for all `app/` modules, serve embedded SPA frontend as static files
7. `gin.Run()`

### App layer (`app/`)

Each subdirectory is a feature module following the same pattern:

- `handler.go` — HTTP handler struct and methods. The `Handler` struct holds the DAOs it needs.
- `router.go` — `RegisterRoutes(r *gin.Engine)` attaches endpoints under `/v3/<module>/...`
- `utils.go` — Request/response types and helper logic specific to the module
- `i18n.go` — Module-specific i18n messages merged into the global `utils.I18n`

Routes use `middleware.TokenCheck()` for authenticated endpoints and `middleware.AdminOnly()` for admin-only endpoints. Token is passed via `X-DMP-TOKEN` header.

### DST game controller (`dst/`)

The `Game` struct wraps a room + worlds + settings and operates on the DST server via:
- **File I/O**: reads/writes cluster.ini, server.ini, modoverrides.lua, adminlist.txt, etc. under `~/.klei/DoNotStarveTogether/Cluster_<id>/`
- **Screen commands**: starts/stops worlds using `screen` sessions, sends Lua console commands via `screen -X stuff`
- **Lua parsing**: uses `yuin/gopher-lua` to parse `modinfo.lua` (mod configuration forms) and `modoverrides.lua` (mod enabled state and options)
- **Session reading**: reads DST binary session files through Lua VM to extract game stats (cycles, season, phase)

`NewGameController(room, worlds, setting, lang)` creates a ready-to-use controller. It initializes paths, world configs, player lists, and mod directories.

### Database

- **ORM**: GORM with `glebarez/sqlite` driver, SQLite in WAL mode, single-connection (SetMaxOpenConns=1)
- **Models** (`database/models/`): User, Room, World, RoomSetting, GlobalSetting, System, UidMap
- **DAO** (`database/dao/`): Generic `BaseDAO[T]` provides CRUD + paginated query; typed DAOs (UserDAO, RoomDAO, etc.) embed BaseDAO and add domain-specific queries
- **In-memory cache** (`database/db/cache.go`): JWT secret, players statistics (per-room player snapshots), players online time, system metrics (CPU/memory/disk/network), internet IP, mod download state

### Scheduler (`scheduler/`)

Uses `go-co-op/gocron`. Jobs are defined in `initJobs()` and managed dynamically:

- **Global jobs**: online player polling, system metrics collection, game update check, internet IP refresh, temp mod cleanup
- **Per-room jobs**: backup (multiple times/day), backup cleanup, scheduled restart, scheduled start/stop, keepalive (world crash detection via log timestamps), announcements

`UpdateJob()` and `DeleteJob()` allow dynamic job management at runtime when room settings change.

### Middleware (`middleware/`)

- `TokenCheck()` — validates `X-DMP-TOKEN` JWT, sets username/nickname/role in Gin context, auto-refreshes token when >50% expired
- `AdminOnly()` — rejects non-admin users (role != "admin")
- `CacheControl()` — sets 48-hour cache headers on static asset extensions

### EmbedFS (`embedFS/`)

Embeds the frontend SPA (`dist/`), LuaJIT shared libraries (`luajit/`), and shell scripts (`shell/`). The SPA is served via `gin-static` as the catch-all route; LuaJIT libs and scripts are extracted to disk at startup.

### Utils (`utils/`)

- `constants.go` — version, API prefix, paths, external API URLs
- `jwt.go` — JWT generation/validation with HS256
- `i18n.go` — request-scoped i18n via `X-I18n-Lang` header (zh/en), each app module registers its own messages
- `security.go` — `IsSafeString` (prevents command injection in world/screen names), `IsSafePath` (prevents path traversal)
- `getter.go` — obfuscated Steam API key and DST token retrieval
- Various helpers for file I/O, zip/unzip, bash command execution, system metrics (CPU/memory/disk/network)

## CI/CD

GitHub Actions in `.github/workflows/go.yml` triggers on `v*` tags:
- Builds static binary with `CGO_ENABLED=0`, creates `.tgz` and draft GitHub release
- Builds and pushes Docker image to `ghcr.io/miracleeverywhere/dst-management-platform-api`

Docker setup: multi-stage build (Go build → Ubuntu 24.04 runtime with screen + wget), entry point in `docker/entry-point.sh`.

## Key conventions

- Indentation: tabs (Go), spaces for YAML/markdown — see `.editorconfig`
- API prefix: `/v3/` (defined in `utils.ApiVersion`)
- All API responses use `{"code": 200, "message": "...", "data": ...}`; HTTP status is always 200, errors signaled by `code` field
- Logger is the global `logger.Logger` (zap SugaredLogger); use structured logging: `logger.Logger.Error("msg", "key", val)`
- DST game state is controlled through the `dst.Game` controller — never manipulate DST files or screen sessions directly from handlers
