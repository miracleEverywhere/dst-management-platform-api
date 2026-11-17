# AI 诊断与一键修复（前端对接文档）

> 状态：后端 v0.1 已实现，前端按本文对接
> 更新：2026-11-17
> 适用范围：房间级诊断、平台级诊断、SSE 流式报告、用户授权修复
> 后端入口：`app/diagnosis`、`service/diagnosis`

## 1. 功能概述

页面入口：平台工具 → AI 诊断（`/tools/ai-diagnosis`，位于「AI 对话」上方）。

用户在页面发起「AI 诊断」后：

```text
POST /v3/diagnosis
  → 后端创建诊断会话，异步运行 Agent
  → Agent 调用只读工具获取 CPU/内存/日志/世界状态/模组/备份等
  → 调用 finish_diagnosis 得到结构化结论与候选动作
  → SSE 推送诊断进度、结构化结论、逐 token 报告
  → 用户勾选动作并二次确认
  → POST /v3/diagnosis/:id/apply
  → SSE 推送每个动作的执行与复检结果
```

角色：

- 普通用户：只能诊断自己有权限的房间，不能诊断平台，不能读取系统日志。
- 管理员：可以诊断任意房间，也可以诊断整个平台，可以读取系统日志和执行管理员动作。

## 2. 通用约定

- API 前缀：`/v3`
- 认证头：`X-DMP-TOKEN`
- 语言头：`X-I18n-Lang`，取值 `zh` 或 `en`
- JSON 响应：`{"code":200,"message":"success","data":...}`
- 业务成功码：`200`
- 登录失效：`420`，前端清除登录态并跳登录页
- 诊断错误码：`10001-10999`，详见第 12 节
- 时间：RFC3339，例如 `2026-11-17T16:10:00+08:00`
- 诊断 ID：`diag_` + 随机串；执行 ID：`apply_` + 随机串

SSE 使用 `fetch`，不要用 `EventSource`，因为需要携带 `X-DMP-TOKEN`。

## 3. 接口总览

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/v3/diagnosis` | 创建诊断 |
| GET | `/v3/diagnosis/:id/events` | 诊断 SSE |
| GET | `/v3/diagnosis/:id` | 诊断快照，用于刷新/兜底 |
| DELETE | `/v3/diagnosis/:id` | 取消诊断 |
| POST | `/v3/diagnosis/:id/apply` | 授权执行修复 |
| GET | `/v3/diagnosis/:id/apply/events` | 修复执行 SSE |
| GET | `/v3/diagnosis/history` | 诊断历史分页 |
| GET | `/v3/diagnosis/history/:id` | 历史详情 |
| GET | `/v3/diagnosis/setting` | 获取诊断 LLM 配置（管理员） |
| PUT | `/v3/diagnosis/setting` | 保存诊断 LLM 配置（管理员） |

### 3.1 诊断 LLM 配置

诊断优先使用自己的 LLM 配置；诊断未填写 `chatBaseURL`/`chatModel`/`chatApiKey` 时，自动回退 AI 对话的全局基础配置。两处都不可用时，创建诊断返回 `10061`。

`GET /v3/diagnosis/setting`（管理员）：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "source": "aichat",
    "configured": false,
    "setting": {
      "chatBaseURL": "",
      "chatApiKey": "",
      "chatModel": ""
    },
    "effective": {
      "chatBaseURL": "https://api.example.com/v1",
      "chatApiKey": "sk-***",
      "chatModel": "gpt-4o-mini"
    }
  }
}
```

- `source`：`diagnosis`（使用诊断专用配置）、`aichat`（回退 AI 对话配置）、`none`（两处都不可用）。
- `configured`：是否填写了诊断专用配置。
- `setting`：诊断专用配置原文，用于表单回显。
- `effective`：本次诊断实际使用的模型配置。
- 诊断固定使用 `temperature=0.2`、`max_tokens=4096`、单次请求 90 秒超时，不随配置变化。

`PUT /v3/diagnosis/setting`（管理员）请求体与 `setting` 相同：

```json
{
  "chatBaseURL": "https://api.example.com/v1",
  "chatApiKey": "sk-xxx",
  "chatModel": "gpt-4o-mini"
}
```

将 `chatBaseURL`、`chatApiKey`、`chatModel` 全部留空即清除诊断专用配置并回退 AI 对话配置。配置不合法返回 `10063`。

## 4. 创建诊断

### 4.1 请求

```http
POST /v3/diagnosis
Content-Type: application/json
X-DMP-TOKEN: ...
X-I18n-Lang: zh

{"scope":"room","roomID":1}
```

平台级诊断：

```json
{"scope":"platform"}
```

字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `scope` | string | `room` 或 `platform`，默认 `room` |
| `roomID` | int | `scope=room` 时必填；`scope=platform` 时忽略 |

### 4.2 响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "diagnosisID": "diag_xxx",
    "scope": "room",
    "status": "running",
    "eventsURL": "/v3/diagnosis/diag_xxx/events"
  }
}
```

`eventsURL` 是完整路径，可直接用于 `fetch`。

### 4.3 常见失败

- `10001` 参数错误
- `10021` 无房间权限，或非管理员尝试平台诊断
- `10061` 未配置 AI 模型
- `10062` 同一账号/房间已有诊断在运行，或 15 秒内重复发起
- `2041` 房间不存在

## 5. 诊断 SSE

### 5.1 连接

```js
const response = await fetch(eventsURL, {
  headers: {
    'X-DMP-TOKEN': token,
    'X-I18n-Lang': lang,
    'Accept': 'text/event-stream',
    'Last-Event-ID': lastEventID || '',
  },
  signal: abortController.signal,
})
if (!response.ok || !response.headers.get('content-type')?.includes('text/event-stream')) {
  // 服务端在开始流之前会用普通 JSON 返回错误，按 response.json() 处理
}
```

服务端响应头：

```text
Content-Type: text/event-stream; charset=utf-8
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

### 5.2 帧格式

```text
id: 1
event: session
data: {"diagnosisID":"diag_xxx","scope":"room","status":"running"}

: ping

id: 2
event: step
data: {"index":1,"tool":"get_system_metrics","status":"running","startedAt":"..."}
```

- `id` 从 1 递增，可用于 `Last-Event-ID` 续传。
- 每 15 秒发送一次 `: ping` 心跳。
- 事件 JSON 中的 `data` 是单行 JSON，换行会被转义。
- 诊断事件流在 `done` 后关闭。

### 5.3 事件类型

| 事件 | 含义 | data |
|---|---|---|
| `session` | 会话开始 | `{diagnosisID, scope, roomID, status, startedAt}` |
| `step` | 工具调用进度 | `Step` |
| `notice` | 警告/提示 | `{level, message}` |
| `report_meta` | 结构化结论（在正文流之前） | `Report` 去掉 `markdown/usage` |
| `report_start` | 报告正文开始 | `{model}` |
| `report_delta` | 报告正文增量 | `{delta}` |
| `report_end` | 报告正文结束 | `{report, usage}` |
| `done` | 诊断结束 | `{status, finishedAt, usage}` |
| `error` | 错误 | `{code, message, retryable}` |

### 5.4 Step

```json
{
  "index": 1,
  "tool": "get_system_metrics",
  "status": "running",
  "startedAt": "2026-11-17T16:10:00+08:00",
  "durationMs": 0,
  "error": ""
}
```

`status`：`running`、`done`、`error`。

前端建议按 `tool` 映射文案：

| tool | 建议文案 |
|---|---|
| `get_system_metrics` | 正在检查系统资源 |
| `get_room_summary` | 正在读取房间信息 |
| `get_world_status` | 正在检查世界状态 |
| `get_logs` | 正在读取日志 |
| `get_mods` | 正在检查模组 |
| `get_game_version` | 正在检查游戏版本 |
| `get_scheduler` | 正在检查定时任务 |
| `get_backups` | 正在读取备份 |
| `search_kb` | 正在检索知识库 |
| `propose_action` | 正在整理修复动作 |
| `finish_diagnosis` | 正在生成诊断结论 |

### 5.5 完整示例

```text
id: 1
event: session
data: {"diagnosisID":"diag_xxx","scope":"room","roomID":1,"status":"running"}

id: 2
event: step
data: {"index":1,"tool":"get_system_metrics","status":"running","startedAt":"2026-11-17T16:10:00+08:00"}

id: 3
event: step
data: {"index":1,"tool":"get_system_metrics","status":"done","startedAt":"2026-11-17T16:10:00+08:00","durationMs":120}

id: 4
event: report_meta
data: {"summary":"世界 B 可能因模组加载失败未启动","severity":"high","findings":[...],"actions":[...],"references":[],"model":"gpt-4o-mini"}

id: 5
event: report_start
data: {"model":"gpt-4o-mini"}

id: 6
event: report_delta
data: {"delta":"## 总体结论\n"}

id: 30
event: report_end
data: {"report":"## 总体结论\n...","usage":{"promptTokens":1234,"completionTokens":567,"totalTokens":1801}}

id: 31
event: done
data: {"status":"succeeded","finishedAt":"2026-11-17T16:11:00+08:00","usage":{...}}
```

### 5.6 重连

- 记录最后一个 `id`，重连时放到 `Last-Event-ID` 请求头。
- 后端会从该 ID 之后回放内存事件。
- 会话 15 分钟过期；过期后返回 `10041`，需要重新诊断。
- 断线不代表诊断取消，诊断会继续跑。

## 6. 诊断快照与取消

### 6.1 快照

```http
GET /v3/diagnosis/:id
```

用于刷新页面、深链恢复。返回 `SessionSnapshot`：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "diagnosisID": "diag_xxx",
    "scope": "room",
    "roomID": 1,
    "roomName": "My Room",
    "status": "succeeded",
    "createdAt": "...",
    "finishedAt": "...",
    "steps": [],
    "report": {
      "summary": "...",
      "severity": "high",
      "markdown": "## 总体结论\n...",
      "findings": [],
      "actions": [],
      "references": [],
      "model": "gpt-4o-mini",
      "usage": {"promptTokens": 0, "completionTokens": 0, "totalTokens": 0}
    },
    "canApply": true,
    "apply": null
  }
}
```

- 会话在内存中时返回实时状态。
- 会话过期后回退数据库记录，`canApply=false`。
- `status`：`running`、`succeeded`、`failed`、`canceled`、`interrupted`。

### 6.2 取消

```http
DELETE /v3/diagnosis/:id
```

会话不存在返回 `10041`，无权限返回 `10021`。

## 7. 报告数据结构

### 7.1 Report

```json
{
  "summary": "一句话结论",
  "severity": "high",
  "markdown": "完整 Markdown 报告",
  "findings": [],
  "actions": [],
  "references": [],
  "model": "gpt-4o-mini",
  "usage": {
    "promptTokens": 1234,
    "completionTokens": 567,
    "totalTokens": 1801
  }
}
```

`severity`：`low`、`medium`、`high`、`critical`。

### 7.2 Finding

```json
{
  "findingID": "f_1",
  "title": "模组加载失败",
  "severity": "high",
  "confidence": 0.9,
  "kbID": "mod-load-failure",
  "cause": "模组 123 加载失败",
  "solutions": ["禁用模组 123", "重启世界"],
  "evidenceIDs": ["ev_3"],
  "evidence": [
    {
      "evidenceID": "ev_3",
      "tool": "get_logs",
      "source": "room:1/world:2/server_log.txt",
      "summary": "读取 game 日志 80 行",
      "detail": "日志内容（已脱敏、截断）",
      "truncated": false,
      "createdAt": "..."
    }
  ]
}
```

### 7.3 Action

```json
{
  "actionID": "disable_mod",
  "titleKey": "diagnosis action disable mod",
  "risk": "medium",
  "reversible": true,
  "adminOnly": false,
  "confirmationLevel": "normal",
  "confirmationHint": "请确认执行该动作",
  "params": {"modID": 123},
  "preview": "将禁用模组 workshop-123",
  "reason": "日志显示该模组加载失败",
  "evidenceIDs": ["ev_3"],
  "dependsOn": [],
  "order": 20,
  "available": true,
  "unavailableReason": "",
  "status": "proposed",
  "applyResult": null
}
```

`confirmationLevel`：

- `none`：直接执行
- `normal`：需要 `confirm=true`
- `typed`：需要 `confirm=true` 且 `confirmationText` 精确匹配提示文本

`status`：`proposed`、`running`、`succeeded`、`failed`、`skipped`。

## 8. 修复动作目录（v0.1）

| actionID | params | 风险 | 确认 | 权限 | 说明 |
|---|---|---|---|---|---|
| `start_world` | `worldID` | low | none | 房间/管理员 | 启动未运行的世界 |
| `restart_world` | `worldID` | low | none | 房间/管理员 | 停止后启动世界 |
| `restart_room` | `roomID`（平台级） | medium | normal | 房间/管理员 | 停止并启动全部世界 |
| `clean_logs` | `targets` | low | none | 房间/管理员 | 清理 game/chat；runtime/access/steam 仅管理员 |
| `disable_mod` | `modID` | medium | normal | 房间/管理员 | 从模组配置中禁用 |
| `enable_mod` | `modID` | medium | normal | 房间/管理员 | 重新启用模组 |
| `delete_snapshot` | `sessionID` | high | typed（房间名） | 房间/管理员 | 先备份再删除存档快照 |
| `restore_backup` | `backupName` | high | typed（房间名） | 房间/管理员 | 先备份当前存档再恢复 |
| `update_game` | 无 | medium | typed（`UPDATE`） | 仅管理员 | 通过 SteamCMD 更新，耗时较长 |

`clean_logs` 的 `targets` 示例：

```json
{"targets":{"game":true,"chat":false,"runtime":false,"access":false,"steam":false}}
```

平台级动作如果需要指定房间，在 `params` 中传 `roomID`；`update_game` 是全局动作，不需要 `roomID`。

### 8.1 证据要求

- `disable_mod`：证据中必须出现 `workshop-<modID>`；未传 `modID` 时后端会自动提取。
- `delete_snapshot`：证据中必须出现 `session/<sessionID>`；未传 `sessionID` 时后端会自动提取。
- `update_game`：必须有版本检查证据，且包含 `"gameVersionOutdated": true` 或 `"dstUpdating": true`。
- 其他动作至少需要一条证据。

### 8.2 尚未开放

以下动作不在 v0.1 的 AI 动作目录中，前端不要展示为可执行：

- `predownload_mods`：需要 Steam 模组文件地址解析，暂走模组页人工操作。
- `change_port`：涉及端口、NAT、配置文件与重启，暂走房间设置人工操作。
- `reset_world`、`delete_room`：不可逆破坏性操作，永远不交给 AI 执行。

## 9. 执行修复

### 9.1 请求

```http
POST /v3/diagnosis/:id/apply
Content-Type: application/json
```

```json
{
  "actionIDs": ["disable_mod"],
  "confirm": true,
  "confirmationText": "",
  "paramOverrides": {}
}
```

- `actionIDs`：必须是 `report.actions` 里的动作。
- `confirm`：`normal` / `typed` 动作必须为 `true`。
- `confirmationText`：`typed` 动作必填，见 `confirmationHint`。
- `paramOverrides`：可选，仅用于覆盖动作参数；服务端会重新校验。

### 9.2 响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "applyID": "apply_xxx",
    "status": "running",
    "eventsURL": "/v3/diagnosis/diag_xxx/apply/events"
  }
}
```

### 9.3 失败

- `10041` 会话不存在
- `10042` 会话过期
- `10043` 诊断未成功结束
- `10064` 动作不在候选列表
- `10065` 参数/状态不满足
- `10066` 缺少确认或确认文本不匹配
- `10067` 房间已有修复任务
- `10021` 无权限

## 10. 修复执行 SSE

### 10.1 连接

`GET /v3/diagnosis/:id/apply/events`，请求头与诊断 SSE 相同。

### 10.2 事件

| 事件 | data |
|---|---|
| `apply_start` | `{applyID, actionIDs, startedAt}` |
| `action_start` | `{actionID, index, startedAt}` |
| `action_done` | `ActionResult` |
| `apply_end` | `{status, results, finishedAt}` |
| `error` | `{code, message, retryable}` |

`ActionResult`：

```json
{
  "actionID": "disable_mod",
  "status": "succeeded",
  "code": 0,
  "message": "",
  "verified": true,
  "verifyMessage": "模组已禁用",
  "durationMs": 1234
}
```

整体 `status`：`succeeded`、`failed`、`partial`。

失败后后端会停止后续动作，并给剩余动作发 `skipped` 结果。

## 11. 历史记录

### 11.1 列表

```http
GET /v3/diagnosis/history?page=1&pageSize=10&scope=room&status=succeeded
```

返回：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "rows": [
      {
        "diagnosisID": "diag_xxx",
        "scope": "room",
        "roomID": 1,
        "roomName": "My Room",
        "username": "admin",
        "status": "succeeded",
        "severity": "high",
        "summary": "...",
        "model": "gpt-4o-mini",
        "actionCount": 2,
        "appliedCount": 1,
        "createdAt": "...",
        "finishedAt": "...",
        "canApply": false
      }
    ],
    "page": 1,
    "pageSize": 10,
    "total": 1
  }
}
```

### 11.2 详情

`GET /v3/diagnosis/history/:id` 返回 `SessionSnapshot`，结构与诊断快照一致，`CanApply=false`。

### 11.3 权限

- 管理员：全部记录。
- 普通用户：只返回自己有权限的房间记录。
- 平台级记录仅管理员可见。

## 12. 错误码

| 码 | 中文 | English | 前端处理 |
|---|---|---|---|
| 10001 | 请求参数错误 | Bad Request | 提示检查参数 |
| 10021 | 权限不足 | Insufficient Permissions | 隐藏/禁用操作 |
| 10041 | 诊断记录不存在或已被清理 | Diagnosis not found | 刷新诊断页，提示重新诊断 |
| 10042 | 诊断已过期 | Diagnosis expired | 提示重新诊断 |
| 10043 | 诊断尚未完成 | Diagnosis is not ready | 等待完成后重试 |
| 10061 | 尚未配置可用的大模型 | No AI model is configured | 引导管理员配置诊断或 AI 对话设置 |
| 10062 | 诊断任务正在进行中 | Diagnosis is already running | 禁用按钮，稍后重试 |
| 10063 | 诊断大模型配置不完整或不合法 | Diagnosis AI configuration invalid | 检查 Base URL、模型名与采样参数 |
| 10064 | 动作不在本次诊断的候选列表中 | Action is not part of this diagnosis | 刷新诊断结果 |
| 10065 | 动作参数或当前状态不满足 | Action precondition failed | 重新诊断/刷新状态 |
| 10066 | 高风险动作需要二次确认 | Confirmation required | 弹出确认框 |
| 10067 | 该房间已有修复任务 | Repair task is already running | 禁用执行按钮 |
| 10091 | AI 调用失败 | AI request failed | 显示失败提示/重试 |
| 10092 | 修复执行失败 | Repair failed | 展示动作结果 |
| 10093 | 修复后复检未通过 | Verification failed | 提示人工检查 |
| 10094 | 数据库连接失败 | Database error | 提示查看服务端日志 |

`420` 按现有全局逻辑处理：清除登录态并跳转登录页。

## 13. 前端实现建议

### 13.1 状态机

```text
idle
  → starting（POST /diagnosis）
  → connecting（打开诊断 SSE）
  → running（收到 session/step）
  → streaming（收到 report_delta）
  → report_ready（report_end + done）
  → applying（POST apply + 打开执行 SSE）
  → applied / apply_failed
```

异常：

- 开始失败：回到 idle，显示错误。
- SSE 断线：按 `Last-Event-ID` 重连，最多 3 次。
- 诊断 `error`：显示错误；如果已经有 `report_meta`，仍展示结构化结论。
- 刷新页面：用 `GET /diagnosis/:id` 恢复；如果 10041/10042，跳历史或提示重新诊断。

### 13.2 SSE 客户端

建议新增 `src/utils/sse.js`：

```js
export async function connectSSE(url, { token, lang, lastEventID, signal, onEvent }) {
  const response = await fetch(url, {
    headers: {
      'X-DMP-TOKEN': token,
      'X-I18n-Lang': lang,
      Accept: 'text/event-stream',
      ...(lastEventID ? { 'Last-Event-ID': String(lastEventID) } : {}),
    },
    signal,
  })
  if (!response.ok) throw new Error(`SSE HTTP ${response.status}`)
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('text/event-stream')) {
    const payload = await response.json().catch(() => null)
    if (payload?.code === 420) throw Object.assign(new Error('token expired'), { code: 420 })
    throw Object.assign(new Error(payload?.message || 'SSE failed'), { code: payload?.code })
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let eventID = lastEventID || 0
  let eventType = 'message'
  let dataLines = []

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const blocks = buffer.split('\n\n')
    buffer = blocks.pop()
    for (const block of blocks) {
      eventType = 'message'
      dataLines = []
      for (const line of block.split('\n')) {
        if (line.startsWith(':')) continue
        if (line.startsWith('id:')) eventID = Number(line.slice(3).trim()) || eventID
        else if (line.startsWith('event:')) eventType = line.slice(6).trim()
        else if (line.startsWith('data:')) dataLines.push(line.slice(5).trim())
      }
      if (!dataLines.length) continue
      const payload = JSON.parse(dataLines.join('\n'))
      onEvent({ id: eventID, type: eventType, data: payload })
    }
  }
}
```

注意：

- 不要用 `axios` 读 SSE，`axios` 默认会缓冲整个响应。
- `report_delta` 可能很频繁，建议 50-100ms 节流后再更新 Markdown 渲染。
- 组件卸载时 `abortController.abort()`。

### 13.3 报告渲染

- 使用项目已有的 `MdPreview`（`md-editor-v3`）渲染 `report.markdown`。
- 证据 `detail` 用 `<pre>` 纯文本展示，不要再用 Markdown 渲染，避免日志内容注入。
- `report_meta` 先到，可以先渲染发现列表和动作卡片；但“执行修复”按钮建议在 `done` 后再启用。
- 严重程度用 chip：`low` 绿、`medium` 橙、`high` 红、`critical` 深红。

### 13.4 动作卡片

每张卡片展示：

- 动作标题（可用 `titleKey` 走前端 i18n）
- 风险、是否可逆、是否仅管理员
- `preview`
- `reason`
- 证据折叠面板
- 勾选框
- 不可用时展示 `unavailableReason`

高风险动作：

- `delete_snapshot`、`restore_backup`：输入房间名。
- `update_game`：输入 `UPDATE`。
- 弹窗中明确写“执行前会自动备份”。

### 13.5 执行进度

- `POST apply` 成功后打开 `apply/events`。
- 按 `action_start` / `action_done` 更新卡片状态。
- `apply_end` 后展示整体结果和复检信息。
- 失败后不要自动重试；提供“重新诊断”。

### 13.6 文案

后端 `i18n/diagnosis.go` 提供动作标题 key：

- `diagnosis action start world`
- `diagnosis action restart world`
- `diagnosis action restart room`
- `diagnosis action clean logs`
- `diagnosis action disable mod`
- `diagnosis action enable mod`
- `diagnosis action delete snapshot`
- `diagnosis action restore backup`
- `diagnosis action update game`

前端也可以直接用 `Action.titleKey` 查自己的语言表。

## 13.7 建议改动文件

- `src/api/diagnosis.js`：新增接口封装。
- `src/utils/sse.js`：新增基于 fetch 的 SSE 客户端。
- `src/views/tools/ai-diagnosis.vue`：独立诊断页面，包含诊断、历史与管理员 LLM 配置。
- 后端菜单项：`tools/ai-diagnosis`，路由 `/tools/ai-diagnosis`，位于 `/tools/aichat` 上方。
- `src/languages/lang-zh.js`、`src/languages/lang-en.js`：新增 `diagnosis.*` 与动作标题文案。
- 如果动作卡片要复用现有组件，优先使用项目已有的 `v-card`、`v-chip`、`v-dialog`、`v-expansion-panels` 和 `MdPreview`。

## 14. 安全与隐私

- 诊断只读取数据，工具全部只读；唯一的写操作是用户授权后的动作。
- AI 不能生成 shell 命令，只能返回动作 ID。
- 动作必须在服务端白名单中，并重新校验权限、参数、证据和当前状态。
- 不发送：房间 Token、房间密码、ClusterKey、`cluster_token.txt`、AI API Key、Webhook 地址、玩家聊天日志。
- 系统日志仅在管理员诊断时读取；普通用户只能读 game 日志。
- 日志中可能包含敏感信息，服务端会脱敏并用 `truncated` 标记截断。
- 高风险动作先备份，确认文本不匹配不执行。
- 所有执行动作写入 `diagnosis_action_records` 审计表。

## 15. 知识库

后端内置 `service/diagnosis/kb.json`，启动时通过 `//go:embed` 加载并校验。

- 结构定义：`service/diagnosis/kb.schema.json`
- 内容由维护者手动维护；它是独立快照，不与任何外部文档项目同步。
- 知识库只用于模型主动调用 `search_kb` 做关键词检索，不参与启动时的自动规则匹配，也不参与 AI 失败后的降级。
- 修改知识库后需要重新编译后端。
- 知识库加载失败时只记录日志，不影响创建诊断；`search_kb` 会返回空结果。

最小条目骨架：

```json
{
  "id": "example-rule",
  "enabled": true,
  "category": "startup",
  "priority": 50,
  "severity": "high",
  "title": {"zh": "示例问题", "en": "Example issue"},
  "symptoms": {"zh": ["现象"], "en": ["Symptom"]},
  "checks": {"zh": ["检查项"], "en": ["Check"]},
  "causes": {"zh": ["原因"], "en": ["Cause"]},
  "solutions": {"zh": ["处理步骤"], "en": ["Step"]},
  "manualOnly": false,
  "actionHints": [
    {
      "actionID": "restart_room",
      "risk": "medium",
      "requiresConfirmation": true,
      "reason": {"zh": "日志显示房间异常", "en": "Logs show a room issue"}
    }
  ],
  "risk": {"zh": "风险说明", "en": "Risk"},
  "tags": ["example"],
  "updatedAt": "2026-11-17"
}
```

- `enabled=false` 的条目不会出现在检索结果中。
- `priority` 只影响检索结果排序。
- `manualOnly=true` 时不能配置 `actionHints`。

## 16. v0.1 已实现 / 未实现

已实现：

- 房间级、平台级诊断
- 只读工具：系统指标、房间摘要、世界状态、日志、模组、版本、定时任务、备份、知识库检索
- 工具调用失败时返回 `10091`，不再降级为本地知识库报告
- 取证预算用尽后进入收尾模式：只向模型暴露 `finish_diagnosis`，不再用 `tool_choice` 指定函数（思考模型会拒绝），模型只回正文时按正文生成报告
- 诊断专用 LLM 配置，未配置时回退 AI 对话配置
- SSE 进度、结构化结论、报告逐 token 流式
- 动作授权、执行、复检、SSE 进度
- 诊断历史与动作审计落库
- 9 个修复动作（见第 8 节）

暂未实现：

- `predownload_mods`、`change_port` 动作
- 平台级系统日志清理动作
- SSE 事件超过内存上限后的 `reset` 重放
- 诊断报告导出、分享

## 17. 后端验证命令

```bash
gofmt -w $(git diff --name-only -- '*.go')
GOCACHE=/tmp/gocache go vet ./...
GOCACHE=/tmp/gocache go run ./cmd/codedoc
GOCACHE=/tmp/gocache go run ./cmd/codedoc -verify
GOCACHE=/tmp/gocache CGO_ENABLED=0 go build -o /tmp/dmp-check .
```

真实 AI 调用、DST 进程操作、SteamCMD 更新需要在目标环境联调；沙箱只能验证编译、静态检查、错误码与知识库加载。
