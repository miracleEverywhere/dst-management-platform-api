# 游戏安装/更新：SSE 改造（后端已实现 + 前端改造方案）

> 适用范围：DMP 主程序（`embedFS/dist` 前端）的「游戏安装 / 游戏更新」页面。
> 精简版（`lite`）没有安装页，也不受本次改动影响。

## 0. 一句话结论

游戏安装/更新不再复用 `/v3/platform/webssh` 的 WebSocket 全双工通道，改为**后台任务 + 一条 SSE 流**：

- 前端一次 `GET` 请求既是「启动」也是「订阅」；
- 输出以纯文本行推送，前端用普通滚动日志面板渲染，**不再需要 xterm**；
- 任务与请求解耦，**刷新页面、断网重连都能接回同一个任务**；
- 成败由后端给出的 `done` 事件（退出码）判定，**前端不再靠中文标记正则**。

---

## 1. 后端接口契约

### 1.1 状态查询：`GET /v3/platform/game/install/state`

请求头：`X-DMP-TOKEN`（管理员）。响应为普通 JSON：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "running": true,
    "action": "install",
    "streamed": true,
    "startedAt": 1763551200,
    "finishedAt": 0,
    "success": false,
    "exitCode": 0,
    "offset": 20480
  }
}
```

| 字段 | 说明 |
| --- | --- |
| `running` | 是否有任务在跑（包含定时任务、控制台更新、诊断修复触发的外部任务） |
| `action` | `install` / `update` |
| `streamed` | `false` 表示任务不是网页发起的，没有输出流；此时 `success`/`exitCode` 无意义，不要弹结果弹窗 |
| `startedAt` / `finishedAt` | Unix 秒；`finishedAt` 为 0 表示从未结束过 |
| `offset` | 已产生的输出总字节数，可作为重新订阅的起点 |

前端进入页面时**先查这个接口**，据此决定：显示准备页（空闲）／直接进入日志视图（运行中）。

### 1.2 启动 + 订阅：`GET /v3/platform/game/install/stream`

| 查询参数 | 必填 | 说明 |
| --- | --- | --- |
| `action` | 否 | `install` / `update`。**传了就启动新任务**；为空表示只订阅已有任务（刷新/重连） |
| `after` | 否 | 客户端已收到的输出偏移，默认 0。断线续传时传上次的 `id` |

成功：`200 text/event-stream`。失败：**普通 JSON 响应**（前端用 `content-type` 区分）。

| 错误码 | 含义 | 前端处理 |
| --- | --- | --- |
| 5068 | 已有安装/更新任务在跑 | 提示后改用「只订阅」方式接上现有任务 |
| 5069 | 脚本缺失或启动失败 | 提示错误，回到准备页 |
| 5070 | 没有可订阅的任务（进程启动后从未执行过） | 回到准备页 |
| 420 | 登录态失效 | 清 store，跳登录页（沿用全局逻辑） |

### 1.3 SSE 帧格式

```
retry: 3000

id: 4096
event: chunk
data: {"c":"正在更新 Steam 控制台客户端...\n"}

: ping

id: 8192
event: chunk
data: {"c":"Update state (0x61) downloading, progress: 12.34\n"}

event: done
data: {"action":"install","success":true,"exitCode":0}
```

- `event: chunk` → `{"c": string}`，**每段都以换行结尾**，前端按 `\n` 拆行即可。
  输出已在后端清洗：ANSI 颜色/光标控制序列被去掉，进度条的单独 `\r` 按「覆盖当前行」处理，
  不会出现半截碎片。
- `event: done` → 任务结束，**这是最后一条事件**，收到后服务端主动关闭响应。
  `success` 由脚本退出码决定（`exitCode === 0`）。
- `id:` 是累计**字节偏移**，不是行号。续传时把它作为 `after` 传回。
- `: ping` 是每 15 秒一次的注释式心跳，SSE 解析时忽略，用于穿透反向代理的空闲超时。

---

## 2. 前端改造方案

### 2.1 改动清单

| 文件 | 操作 |
| --- | --- |
| `package.json` | 删除 `xterm`、`xterm-addon-fit`（全仓只有安装页在用） |
| `src/utils/sse.js` | **新增**：基于 `fetch` 的 SSE 客户端（不要用 `EventSource`，也**不能**用 `axios`） |
| `src/api/platform.js` | 新增 `gameInstall.stream` / `gameInstall.state` |
| `src/views/install/index.vue` | 删掉 WebSocket + xterm，改成滚动日志面板 + SSE |
| `src/languages/lang-zh.js` / `lang-en.js` | 改写 `install.install.alert`，新增 4 条状态词条 |

> 注意：**不要引入 `EventSource`**。它不能带 `X-DMP-TOKEN` 头，只能把 JWT 放到 URL 里；而且无法读取
> 服务端返回的 `X-DMP-NEW-TOKEN` 续期头。

### 2.2 新增 `src/utils/sse.js`

```js
import { getToken } from '@/utils/tools'
import useUserStore from '@store/user'
import useGlobalStore from '@store/global.js'
import { showSnackbar } from '@/utils/snackbar'

/**
 * 基于 fetch 的 SSE 客户端。
 * 之所以不用 EventSource / axios：
 *   - EventSource 不能自定义请求头（拿不到 X-DMP-TOKEN），也无法读取续期响应头；
 *   - axios 会缓冲整个响应体，且实例 timeout 会掐断长时间静默的流。
 *
 * @param {string} url 相对或绝对地址（含查询参数）
 * @param {object} opts
 * @param {AbortSignal} opts.signal 取消信号
 * @param {(e:{event:string,data:any,id:string})=>void} opts.onEvent 每条事件回调
 * @param {(id:string)=>void} [opts.onOpen] 连接建立（含续传起点 id）
 * @param {(err:Error)=>void} [opts.onError] 传输层错误（断线）
 */
export async function connectSSE(url, { signal, onEvent, onOpen, onError } = {}) {
  const globalStore = useGlobalStore()
  const headers = { Accept: 'text/event-stream' }

  const token = getToken()
  if (token) {
    headers['X-DMP-TOKEN'] = token
  }
  if (globalStore.language) {
    headers['X-I18n-Lang'] = globalStore.language
  }

  let response

  try {
    response = await fetch(url, { headers, signal, cache: 'no-store' })
  } catch (error) {
    onError?.(error)
    throw error
  }

  const contentType = response.headers.get('content-type') || ''

  // 启动失败、任务忙、登录失效等仍按普通 JSON 返回
  if (!response.ok || !contentType.includes('text/event-stream')) {
    const payload = await response.json().catch(() => ({}))
    const status = Number(payload.status ?? payload.code)

    if (status === 420 || response.status === 420) {
      const userStore = useUserStore()

      showSnackbar(payload.message || '登录已过期', 'error')
      await userStore.clearStore()
      if (window.location.hash !== '#/login') {
        window.location.href = '/#/login'
      }
    }

    const error = Object.assign(new Error(payload.message || `SSE HTTP ${response.status}`), payload)

    throw error
  }

  // token 临期时后端会回写新 token，这里补上 axios 拦截器做的那一步
  const newToken = response.headers.get('x-dmp-new-token')
  if (newToken && newToken !== getToken()) {
    useUserStore().token = newToken
  }

  onOpen?.()

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let lastID = ''

  try {
    for (;;) {
      const { value, done } = await reader.read()
      if (done) {
        break
      }

      buffer += decoder.decode(value, { stream: true })

      let index
      while ((index = buffer.indexOf('\n\n')) >= 0) {
        const frame = buffer.slice(0, index)
        buffer = buffer.slice(index + 2)

        let event = 'message'
        let data = ''

        for (const line of frame.split('\n')) {
          if (line.startsWith(':')) {
            continue // 心跳
          }
          if (line.startsWith('id:')) {
            lastID = line.slice(3).trim()
          } else if (line.startsWith('event:')) {
            event = line.slice(6).trim()
          } else if (line.startsWith('data:')) {
            data += line.slice(5).trim()
          }
        }

        if (data) {
          try {
            onEvent?.({ event, data: JSON.parse(data), id: lastID })
          } catch (error) {
            console.error('SSE 事件处理失败', event, error)
          }
        }
      }
    }
  } catch (error) {
    if (error?.name !== 'AbortError') {
      onError?.(error)
      throw error
    }
  }
}
```

### 2.3 `src/api/platform.js` 增补

```js
gameInstall: {
  state: {
    url: `/platform/game/install/state`,
    get: async function () {
      return await http.get(this.url)
    },
  },
  // 流式接口不走 http 封装，交给 utils/sse.js 的 connectSSE 直接请求
  stream: {
    url: `/platform/game/install/stream`,
  },
},
```

### 2.4 改写 `src/views/install/index.vue`

**删除**（全部与传输层有关）：

- 依赖 `xterm` / `xterm-addon-fit` / `xterm/css/xterm.css`
- `ws`、`connected`、`customCommand`、`term`、`fitAddon`、`terminalInitialized`、`fitAddonLoaded`
- `connectWebSocket`、`reconnect`、`sendCommand`、`sendResizeMessage`、`initTerminal`、`cleanupTerminal`
- 4 条中文标记正则（`regexSuccess` / `regexFail` / `regexUpdateSuccess` / `regexUpdateFail`）
- `handleInstall` / `handleUpdate` 里的 `await sleep(2000)`

**保留**：左边的机器信息卡片（`osInfo`、`rating`）、右边准备页的三颗按钮、四个 `confirm-box`。

**新增**：日志行数组 + 自动滚动 + 状态机 + 重连。

模板中把原来的 `<div ref="terminalEl" class="terminal" />` 换成：

```vue
<template>
  <!-- …… 准备页保持不变 …… -->

  <div v-else class="install-log-wrapper">
    <v-alert color="warning" density="compact" class="mt-2 mb-2">
      {{ t('install.install.alert') }}
    </v-alert>
    <div ref="logEl" class="install-log" :style="{ height: `${calculateContainerSize() - 78}px` }">
      <div v-if="lines.length === 0" class="install-log-empty">
        {{ t('install.install.waiting') }}
      </div>
      <div v-for="(line, index) in lines" :key="index" class="install-log-line">{{ line }}</div>
    </div>
    <div v-if="disconnected" class="install-log-retry">
      <v-btn color="primary" variant="flat" size="small" @click="restore">
        {{ t('install.install.retry') }}
      </v-btn>
    </div>
  </div>
</template>
```

`<script setup>` 完整替换为：

```vue
<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDisplay } from 'vuetify/framework'
import { formatBytes } from '@/utils/tools.js'
import useGlobalStore from '@store/global.js'
import platformApi from '@/api/platform.js'
import { connectSSE } from '@/utils/sse.js'
import { showSnackbar } from '@/utils/snackbar.js'
import { ApiVersion } from '@/config'

const MAX_LINES = 2000          // 面板最多保留的行数，超出丢弃最旧的
const MAX_RETRY = 3             // 断线自动重连次数

const globalStore = useGlobalStore()
const { mobile } = useDisplay()
const { t } = useI18n()

const installing = ref(false)   // 是否处于日志视图
const lines = ref([])
const logEl = ref(null)
const disconnected = ref(false)
const runningAction = ref('')   // install / update，用于选择结果弹窗文案

let abort = null
let offset = 0
let retry = 0
let manualClose = false

const installSuccessDialog = ref(false)
const installErrorDialog = ref(false)
const updateSuccessDialog = ref(false)
const updateErrorDialog = ref(false)

// …… osInfo / rating / calculateContainerSize / handleResize 保持原样 ……

const handleInstall = () => start('install')
const handleUpdate = () => start('update')

const start = async action => {
  showSnackbar(t('install.prepare.right.tip'))
  resetLog(action)
  await attach({ action })
}

// 页面刷新后接回正在跑的任务
const restore = async () => {
  const state = await fetchState()
  if (!state?.running) {
    installing.value = false
    return
  }

  resetLog(state.action, false)
  await attach({})
}

const fetchState = async () => {
  try {
    const response = await platformApi.gameInstall.state.get()
    return response?.data || null
  } catch (error) {
    return null
  }
}

const resetLog = (action, scroll = true) => {
  runningAction.value = action || ''
  lines.value = []
  offset = 0
  retry = 0
  disconnected.value = false
  installing.value = true
  if (scroll) {
    scrollToBottom()
  }
}

const attach = async ({ action } = {}) => {
  abort?.abort()
  abort = new AbortController()
  manualClose = false

  const params = new URLSearchParams()

  if (action) {
    params.set('action', action)
  }
  if (offset > 0) {
    params.set('after', String(offset))
  }

  const query = params.toString()
  const url = `/${ApiVersion}${platformApi.gameInstall.stream.url}${query ? `?${query}` : ''}`

  let finished = false

  try {
    await connectSSE(url, {
      signal: abort.signal,
      onOpen: () => {
        retry = 0
        disconnected.value = false
      },
      onEvent: ({ event, data, id }) => {
        if (event === 'chunk') {
          offset = Number(id) || offset
          appendText(data.c)
          return
        }
        if (event === 'done') {
          offset = Number(id) || offset
          finished = true
          finish(data)
        }
      },
      onError: () => {
        // 交给下面的 catch 统一处理
      },
    })

    // 流没收到 done 就结束了：说明服务端异常关闭，按断线处理
    if (!finished && !manualClose) {
      await retryLater()
    }
  } catch (error) {
    if (manualClose) {
      return
    }
    // 5068：已有任务，改为只订阅
    if (Number(error?.code) === 5068) {
      showSnackbar(error.message, 'warning')
      const state = await fetchState()
      if (state?.running) {
        resetLog(state.action)
        await attach({})
      }
      return
    }
    // 5070 / 5069 / 420 等：直接提示并回到准备页
    if (Number(error?.code) === 5070 || Number(error?.code) === 5069) {
      showSnackbar(error.message, 'error')
      installing.value = false
      return
    }
    if (!error?.message) {
      return
    }
    await retryLater()
  }
}

const retryLater = async () => {
  if (retry >= MAX_RETRY) {
    disconnected.value = true
    return
  }
  retry += 1
  showSnackbar(t('install.install.reconnecting'), 'warning')
  const delay = 1000 * 2 ** (retry - 1)
  await new Promise(resolve => setTimeout(resolve, delay))
  await attach({})
}

// 后端保证每段输出以换行结尾，按行拆开追加
const appendText = text => {
  if (!text) {
    return
  }
  const next = lines.value.concat(text.split('\n').filter(line => line !== ''))
  lines.value = next.length > MAX_LINES ? next.slice(next.length - MAX_LINES) : next
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  const el = logEl.value
  if (el) {
    el.scrollTop = el.scrollHeight
  }
}

const finish = data => {
  if (data.action === 'update') {
    if (data.success) {
      updateSuccessDialog.value = true
    } else {
      updateErrorDialog.value = true
    }
  } else if (data.success) {
    installSuccessDialog.value = true
  } else {
    installErrorDialog.value = true
  }
}

const reloadPage = () => {
  window.location.reload()
}

const cleanup = () => {
  manualClose = true
  abort?.abort()
  abort = null
  window.removeEventListener('resize', handleResize)
}

onMounted(async () => {
  window.addEventListener('resize', handleResize)
  getOSInfo()

  // 刷新后若任务还在跑，直接接回日志视图
  const state = await fetchState()
  if (state?.running && state.streamed) {
    resetLog(state.action)
    await attach({})
  }
})

onUnmounted(cleanup)
</script>
```

样式（替换掉原来的 `.terminal` / `:deep(.xterm)`）：

```css
.install-log {
  width: 100%;
  overflow-y: auto;
  padding: 12px;
  border-radius: 4px;
  background: rgba(var(--v-theme-surface-variant), 0.4);
  font-family: Menlo, Consolas, 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}

.install-log-line {
  min-height: 1.5em;
}

.install-log-empty {
  opacity: 0.6;
}

.install-log-retry {
  display: flex;
  justify-content: center;
  margin-top: 12px;
}
```

### 2.5 i18n

`lang-zh.js` / `lang-en.js` 的 `install.install`：

```js
// zh
install: {
  alert: '安装/更新在后台继续执行，刷新或关闭本页面不会中断；重新打开本页即可接回实时日志',
  waiting: '正在等待输出…',
  reconnecting: '日志连接中断，正在重试…',
  retry: '重新连接日志',
  success: '安装完成',
  fail: '安装失败',
  updateSuccess: '更新完成',
  updateFail: '更新失败',
  confirm: '确认',
}

// en
install: {
  alert: 'The installation runs in the background; refreshing or closing this page will not interrupt it. Reopen the page to reattach the live log.',
  waiting: 'Waiting for output…',
  reconnecting: 'Log connection lost, retrying…',
  retry: 'Reconnect log',
  success: 'Installation Completed',
  fail: 'Installation Failed',
  updateSuccess: 'Update Completed',
  updateFail: 'Update Failed',
  confirm: 'Confirm',
}
```

---

## 3. 交互时序

### 3.1 首次安装

```
页面加载 → GET state（running=false）→ 显示准备页
点击「安装」 → GET stream?action=install
                ├─ 5068 → 提示 + GET state → 接上已有任务
                └─ 200 SSE → chunk…（追加日志）→ done{success} → 结果弹窗
```

### 3.2 安装中刷新页面

```
页面加载 → GET state（running=true, streamed=true）→ 直接进入日志视图
        → GET stream?after=0 → 服务端补发保留的历史 + 后续新输出
```

### 3.3 网络抖动断线

```
流异常结束 → 1s 后 GET stream?after=<最后收到的 id>（不带 action，不会重启任务）
          → 最多重试 3 次 → 仍失败则显示「重新连接日志」按钮
```

### 3.4 定时任务正在更新时用户点安装

```
GET stream?action=install → 5068 → snackbar 提示 → GET state(running=true, streamed=false)
                          → 说明「自动更新进行中，暂无日志」，停留在准备页
```

---

## 4. 注意事项

1. **绝不能用 `axios` 读 SSE**：实例 `timeout: 600000` 会掐断长时间静默的流，响应拦截器还会等待完整 body。
2. **不要用 `EventSource`**：无法携带 `X-DMP-TOKEN`，也读不到 `X-DMP-NEW-TOKEN` 续期头。
3. **`after` 是字节偏移，不是行号**；直接回传收到的最后一个 `id` 即可，不要自己累加。
4. **`done` 之后服务端关流**，属于正常结束；前端不要把它当断线去重连（`manualClose` 之外的正常路径由弹窗接管）。
5. 反向代理：后端已返回 `X-Accel-Buffering: no`，nginx 上一般无需额外配置；如果用了 `proxy_buffering on` 且自定义了 `proxy_read_timeout`，保持默认 60s 即可（有 15s 心跳）。
6. 面板行数上限 `MAX_LINES` 建议保留，安装输出可能有上千行；服务端 ring buffer 上限 512KB。

---

## 5. 验收清单

- [ ] 首次点击「安装」：日志逐行出现，不是等到结束才一次性刷出
- [ ] 安装过程中刷新页面：自动回到日志视图并接上进度
- [ ] 安装过程中关掉浏览器再打开：同上
- [ ] 安装过程中断网 10 秒：恢复后日志从断点续上，不重复不丢失
- [ ] 安装成功/失败：弹窗与脚本真实结果一致（改脚本文案不影响判定）
- [ ] 定时任务正在更新时点「安装」：提示忙，不启动第二个 SteamCMD
- [ ] 非管理员直接请求接口：返回权限错误
- [ ] 安装结束后 `platform/game_version` 返回值已更新（本地版本号不再是 0）
