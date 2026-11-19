# 游戏大厅（Tools / Lobby）

游戏大厅是 `工具` 菜单下的只读页面：从 Klei 官方大厅接口拉取各区域公开的在线房间，支持按房间名搜索、按区域筛选、分页浏览，点击房间后进入全屏详情页查看在线玩家、模组、连接代码等信息。

- 后端：`app/tools` + `service/tools`，两个接口
- 前端：`src/views/tools/lobby.vue` + `src/views/tools/components/LobbyDetail.vue`
- 无数据库变更，**无需数据迁移**；`lite` 精简版不包含该功能

## 数据来源

| 用途 | 接口 |
| --- | --- |
| 房间列表 | `GET https://lobby-v2-cdn.klei.com/{region}-Steam.json.gz` |
| 房间详情 | `POST https://lobby-v2-{region}.klei.com/lobby/read` |
| 模组信息 | 复用本平台已有 `POST /v3/mod/info` |

详情请求体：

```json
{
  "__token": "<constant.KleiLobbyToken>",
  "__gameId": "DontStarveTogether",
  "query": { "__rowId": "<列表中的 __rowId>" }
}
```

区域白名单（`constant/dst.go`，顺序即前端展示顺序）：

`ap-east-1`、`ap-southeast-1`、`us-east-1`、`eu-central-1`

> Klei 可能调整区域列表，改动时同步 `constant/dst.go` 与前端语言包 `tools.lobby.regionName.*`。

## 接口契约

### 1. `POST /v3/tools/lobby/search`

请求：

```json
{
  "keyword": "yyxk",
  "regions": ["ap-east-1", "ap-southeast-1"],
  "page": 1,
  "pageSize": 10,
  "refresh": false
}
```

- `keyword`：房间名模糊匹配，空串表示不筛选；最多 64 个字符
- `regions`：空数组表示全部区域；非法区域返回 `7001`
- `page`：从 1 开始，越界时回落到最后一页
- `pageSize`：前端默认 10，可选 10/20/50/100；服务端缺省 10，夹紧到 1~100
- `refresh`：`true` 时跳过 60 秒缓存；同一区域 10 秒内重复刷新会复用缓存

响应 `data`：

```json
{
  "total": 3440,
  "page": 1,
  "pageSize": 10,
  "regions": ["ap-east-1", "ap-southeast-1", "us-east-1", "eu-central-1"],
  "failedRegions": [],
  "updatedAt": 1795081375,
  "rows": [
    {
      "rowID": "5907f7c1b89864235167fbb2ffb6181e",
      "region": "ap-east-1",
      "name": "yyxk",
      "addr": "117.176.119.34",
      "port": 33002,
      "connected": 20,
      "maxConnections": 20,
      "dedicated": true,
      "clientHosted": false,
      "mods": true,
      "pvp": false,
      "password": false,
      "clanOnly": false,
      "allowNewPlayers": true,
      "serverPaused": false,
      "season": "autumn",
      "intent": "endless",
      "version": 762172,
      "secondaries": [
        { "id": "2684075441", "addr": "118.36.65.173", "port": 10998, "connectionCode": "c_connect(\"118.36.65.173\", 10998)" }
      ],
      "connectionCode": "c_connect(\"117.176.119.34\", 33002)"
    }
  ]
}
```

- `regions` 始终返回全部支持区域，供前端渲染区域选择
- `failedRegions` 中的区域本次刷新失败；若该区域有旧缓存，`rows` 仍会包含旧数据
- `updatedAt` 为参与本次查询的区域缓存最早抓取时间（Unix 秒）
- 排序：在线人数降序 → 房间名升序 → `rowID` 升序，保证翻页稳定

### 2. `POST /v3/tools/lobby/detail`

请求：

```json
{ "rowID": "5907f7c1b89864235167fbb2ffb6181e", "region": "ap-east-1" }
```

响应 `data`：在列表行字段基础上追加：

```json
{
  "desc": "",
  "tick": 30,
  "days": 10,
  "dayElapsedInSeason": 9,
  "daysLeftInSeason": 11,
  "players": [
    { "name": "飞起来", "prefab": "wilson", "colour": "FCE6C9" }
  ],
  "mods": [
    { "id": "2626800998", "name": "夜雨心空3.3.98", "version1": "3.3.98", "version2": "3.3.98", "enabled": true }
  ],
  "modIDs": [2626800998],
  "playersParsed": true
}
```

- `players` / `days` 来自 Klei 返回的 Lua 脚本（`players`、`data` 字段）
- `playersParsed=false` 表示玩家脚本解析失败，前端只提示、不影响基础信息
- `mods` 来自 `mods_info`（扁平五元组：`workshop-<id>`、名称、版本1、版本2、是否启用）
- `modIDs` 供前端调用 `POST /v3/mod/info` 补充预览图与体积

### 错误码

| 码 | 场景 |
| --- | --- |
| `7001` | 参数非法（page/pageSize/region/rowID/keyword） |
| `7072` | 全部查询区域都拉取失败且无缓存 |
| `7073` | 详情获取失败（房间下线、令牌失效、Klei 返回空） |

## 后端实现

| 文件 | 职责 |
| --- | --- |
| `constant/dst.go` | 大厅 URL、`__token`、区域、超时 |
| `constant/codes.go` / `code_registry.go` | `7072`、`7073` |
| `i18n/tools.go` | 失败文案（zh/en） |
| `service/tools/lobby.go` | 参数校验、区域/分页/排序、搜索与详情组装 |
| `service/tools/lobby_client.go` | Klei 请求、gzip 兜底、响应转换、连接代码 |
| `service/tools/lobby_cache.go` | 区域列表缓存、单飞、刷新节流 |
| `service/tools/lobby_parse.go` | Lua 沙箱解析天数/玩家、解析 `mods_info` |
| `app/tools/handler.go` / `router.go` | 两个 POST 路由与参数绑定 |

要点：

- **缓存**：区域列表缓存 60 秒；同一区域并发请求只有一次真正下载；拉取失败时若有旧缓存则返回旧数据并标记 `failedRegions`
- **Lua 沙箱**：复用 `utils.NewSafeLuaState()`，不加载 `os`/`io`/`debug`，脚本长度上限 512KB、执行超时 2 秒
- **gzip**：按 `Content-Encoding` 透明解压，响应头缺失时按 gzip 魔数兜底
- **安全**：区域必须命中白名单（会拼进 URL）；`rowID` 只放进请求体，不拼 URL；关键字与 `rowID` 均有长度上限
- **内存**：缓存只保留列表所需字段，不保存 `guid`/`steamid`/`tags` 等字段

## 前端实现

| 文件 | 改动 |
| --- | --- |
| `src/views/tools/lobby.vue` | 游戏大厅列表页 |
| `src/views/tools/components/LobbyDetail.vue` | 全屏详情对话框 |
| `src/views/tools/lobbyLabels.js` | 区域/模式/季节/颜色映射 |
| `src/api/tools.js` | 新增 `lobby.search`、`lobby.detail` |
| `src/languages/lang-zh.js`、`lang-en.js` | `menu.toolsLobby` 与 `tools.lobby.*` |

页面行为：

- 打开即按默认 `pageSize=10` 加载，默认勾选东亚（`ap-east-1`）与东南亚（`ap-southeast-1`），其余区域可手动勾选
- 关键字输入 400ms 防抖；区域多选；刷新按钮；`pageSize` 可选 10/20/50/100
- 列表列：房间名（含地址）、区域、在线/上限、模式、季节、专服、模组、密码、状态、详情
- 季节只识别春/夏/秋/冬，其余值统一展示“未识别”
- 行点击或“详情”按钮打开全屏对话框
- 详情页展示：描述、连接代码（一键复制）、天数/季节/模式/人数/服务器类型/PVP/密码/允许新玩家加入/版本/Tick/主世界地址、附属世界地址与连接代码、在线玩家（方形角色头像 + 中文名）、模组（方形预览图/名称/ID/大小/启用状态/创意工坊链接）
- 移动端只保留房间名、在线人数、操作三列

模组信息复用 `POST /v3/mod/info`：先用大厅 `mods` 立即渲染名称，再用 `modIDs` 拉取预览图与体积；接口失败时保留大厅信息并提示。

## 验收清单

- [ ] 打开页面自动加载并展示 10 条，默认选中东亚与东南亚
- [ ] 切换每页条数、翻页、关键字搜索、区域多选均正常
- [ ] 刷新按钮生效；10 秒内重复刷新不重复请求
- [ ] 某区域网络失败时展示缓存数据并出现 warning
- [ ] 点击行/详情按钮打开详情，玩家与模组正确展示（图片为正方形）
- [ ] 未知季节显示“未识别”
- [ ] 复制连接代码成功
- [ ] 管理员与普通用户均能看到“游戏大厅”菜单
- [ ] 移动端布局可用

## 已知限制

- 区域列表是编译期常量，Klei 新增区域需要改代码
- 详情依赖 `constant.KleiLobbyToken`，Klei 更换令牌后需要更新
- 详情不缓存，重复打开会再次请求 Klei（列表缓存不受影响）
- 部分服务器不上报 `prefab`，玩家角色会显示为占位图
- 大厅数据为公开信息，页面只读展示，不写库
