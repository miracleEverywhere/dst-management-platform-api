# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

DMP (Don't Starve Together Management Platform) is a Go web server that manages DST game servers. It provides a web UI for multi-room, multi-user game server administration — room creation, mod management, backup/restore, player tracking, and scheduled tasks (restart, keepalive, announcements).

The frontend SPA lives in a separate repo: `miracleEverywhere/dst-management-platform-web` (Vue/Vuetify). Its production build is embedded in `embedFS/dist/`.

## Build and run

```bash
# Full build: frontend + backend (frontend repo expected at ~/WebstormProjects/dst-management-platform-web)
make

# Backend only (static binary, no CGO)
make backend-only

# Or manually:
CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# Run
./dmp -bind 80 -dbpath ./data -level info

# CLI flags
#   -bind     HTTP port (default: 80)
#   -dbpath   SQLite database directory (default: ./data)
#   -level    Log level: debug, info, warn, error (default: info)
#   -cert     TLS certificate path (enables HTTPS when set with -key)
#   -key      TLS private key path (enables HTTPS when set with -cert)
#   -console  Run a console command instead of starting the server
#   -v        Print version and exit

# Console commands (run with -console flag):
#   reset_password     Reset a user password interactively
#   list_user          List all users
#   db_stats           Show database file size and table row counts
./dmp -console reset_password -dbpath ./data
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

When `-level debug` is set, pprof endpoints are registered at `/debug/pprof/`. Production mode (`gin.ReleaseMode`) is used otherwise.

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
- **`dao.FetchGameInfo(roomID)`** (`database/dao/composite.go`): convenience function that fetches Room + Worlds + RoomSetting in one call — used widely across handlers and scheduler jobs
- **Password hashing**: bcrypt (via `golang.org/x/crypto/bcrypt`, default cost). User model has `PasswordVersion` field (`"bcrypt"` / `"sha512"` / `""`) for backward compatibility — SHA512 passwords are upgraded to bcrypt on next successful login. `utils.GenerateBcryptPassword` / `utils.ValidatePassword` in `utils/crypto.go`.
- **In-memory cache** (`database/db/cache.go`): JWT secret, token version cache (username → version for revocation), players statistics (per-room player snapshots), players online time, system metrics (CPU/memory/disk/network), internet IP, mod download state
- **SystemDAO** (`database/dao/system.go`): key-value config store backed by the `system` table. `Get(key)` queries by key, `Set(key, value)` does an atomic upsert via `clause.OnConflict`. JWT secret is stored here (key `jwt_secret`), generated once on first startup.

### Scheduler (`scheduler/`)

Uses `go-co-op/gocron`. Jobs are defined in `initJobs()` and managed dynamically:

- **Global jobs**: online player polling, system metrics collection, game update check, internet IP refresh, temp mod cleanup
- **Per-room jobs**: backup (multiple times/day), backup cleanup, scheduled restart, scheduled start/stop, keepalive (world crash detection via log timestamps), announcements

`UpdateJob()` and `DeleteJob()` allow dynamic job management at runtime when room settings change.

### Middleware (`middleware/`)

- `TokenCheck()` — validates `X-DMP-TOKEN` JWT, checks token version against in-memory cache for revocation, sets username/nickname/role in Gin context, auto-refreshes token (returns new token in `X-DMP-NEW-TOKEN` header) when >50% expired. Returns code 420 on failure.
- `AdminOnly()` — rejects non-admin users (role != "admin"), returns code 201
- `LoginRateLimit()` — rate-limits login endpoint to 1 request/second per IP, returns code 429
- `CacheControl()` — sets cache headers on static asset extensions (duration: `utils.StaticCacheHours`, default 7 days)

### Token revocation

JWT tokens can be revoked without restarting the server via a **token version** mechanism:

- **User model** has a `TokenVersion` column (default 0, persisted in DB). Each JWT claim carries `TokenVersion`.
- **In-memory cache** (`db.TokenVersionCache` in `database/db/cache.go`) maps username → current valid version. Populated on login, checked on every authenticated request.
- **`db.ValidateTokenVersion()`** (`database/db/token_version.go`) — called by TokenCheck middleware. If cache miss → first request this session, auto-caches and allows. If version mismatch → token revoked (code 420).
- **`db.RevokeTokenVersion()`** — increments version in cache, returns new version. Caller must persist to DB.

Revocation triggers:
| Scenario | Location |
|---|---|
| User changes own password | `myselfPut` |
| Admin disables a user | `basePut` |
| Admin changes a user's role | `basePut` |
| Admin deletes a user | `baseDelete` |
| Admin explicitly revokes | `POST /v3/user/revoke` (new endpoint) |

JWT secret is persisted in the `system` DB table (key `jwt_secret`), generated once on first startup. Restarts do NOT invalidate tokens — only explicit revocation does.

### EmbedFS (`embedFS/`)

Embeds the frontend SPA (`dist/`), LuaJIT shared libraries (`luajit/`), and shell scripts (`shell/`). The SPA is served via `gin-static` as the catch-all route; LuaJIT libs (`liblua.so`, `libluajit.so`, `libpreload.so`) and shell scripts (`manual_install.sh`, `manual_update.sh`) are extracted to disk at startup under `dmp_files/`.

### WebSSH (`app/platform/`)

The platform module provides a WebSocket-based terminal (WebSSH) using the `olahol/melody` library. The terminal runs via `creack/pty` and streams I/O over a WebSocket connection. Messages are capped at 1MB.

### Utils (`utils/`)

- `constants.go` — version, API prefix, JWT expiration (72h), static cache duration (7d), paths, external API URLs
- `jwt.go` — JWT generation/validation with HS256, Claims carries Username/Nickname/Role/TokenVersion
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
- Logger is the global `logger.Logger` (zap SugaredLogger); use `*f` formatted methods: `logger.Logger.Errorf("msg, err: %v", err)`
- DST game state is controlled through the `dst.Game` controller — never manipulate DST files or screen sessions directly from handlers
