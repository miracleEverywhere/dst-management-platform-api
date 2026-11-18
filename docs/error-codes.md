# 业务错误码

> 本文件由 `go run ./cmd/codedoc` 自动生成，请勿手工编辑。
> 新增错误码：在 `constant/code_registry.go` 登记条目，在 service 里引用 `constant.CodeXxx`，再重新生成。

响应结构固定为 `{"code": ..., "message": ..., "data": ...}`；`code` 唯一标识一个错误原因，`message` 由 `i18n` 按 `code` 翻译（`X-I18n-Lang` 决定中英文）。成功固定 `200`，认证失效固定 `420`（前端据此清登录态）。

## 号段

| 号段 | 模块 |
|---|---|
| 0-999 | 框架层（token/权限/限流/参数） |
| 1000-1999 | 用户 |
| 2000-2999 | 房间 |
| 3000-3999 | 模组 |
| 4000-4999 | 控制台 |
| 5000-5999 | 平台 |
| 6000-6999 | 日志 |
| 7000-7999 | 工具 |
| 8000-8999 | 玩家 |
| 9000-9999 | 云服务 |
| 10000-10999 | AI 诊断 |

## 错误码一览

| 码 | 模块 | 中文 | English | 级别 | 排查建议 | 出现位置 |
|---:|---|---|---|---|---|---|
| 1 | 框架层 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | middleware/middleware.go:LoginRateLimit |
| 2 | 框架层 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | middleware/utils.go:tokenNoPermission |
| 420 | 框架层 | Token认证失败 | Token Auth Fail | auth | 登录态缺失、过期或签名不匹配。重新登录；若反复出现，检查服务端 jwt-secret 是否被重置。 | middleware/utils.go:tokenMissing<br>middleware/utils.go:tokenRevoked |
| 429 | 框架层 | 请求过于频繁，请稍后再试 | Too Many Requests | warn | 同一账号短时间内请求过于频繁，触发了登录限流。等待约 1 秒后重试。 | middleware/utils.go:tooManyRequests |
| 1001 | 用户 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/user/handler.go:baseDelete<br>app/user/handler.go:basePost<br>app/user/handler.go:basePut<br>app/user/handler.go:loginPost<br>app/user/handler.go:myselfPut<br>app/user/handler.go:registerPost<br>app/user/handler.go:revokePost<br>app/user/handler.go:userListGet |
| 1021 | 用户 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/user/admin.go:UserList |
| 1041 | 用户 | 用户不存在 | User Not Exist | warn | 用户不存在或已被删除。刷新用户列表后重试。 | service/user/admin.go:DeleteUser<br>service/user/admin.go:Revoke<br>service/user/admin.go:UpdateUser<br>service/user/service.go:Login |
| 1061 | 用户 | 请勿重复注册 | User Existed | warn | 用户名已被占用，换一个用户名。 | service/user/admin.go:CreateUser<br>service/user/service.go:Register |
| 1062 | 用户 | 密码错误 | Wrong Password | warn | 密码错误。确认大小写与输入法；多次失败会被限流。 | service/user/service.go:Login |
| 1063 | 用户 | 用户已被禁用 | User is Disabled | warn | 账号已被禁用，需要管理员在用户管理里解除禁用。 | service/user/service.go:Login |
| 1064 | 用户 | 禁止删除所有用户 | Prohibit deletion of all users | warn | 这是最后一个管理员账号，系统拒绝删除以免把自己锁在门外。先创建另一个管理员再删除。 | service/user/admin.go:DeleteUser |
| 1065 | 用户 | 更新失败 | Update Fail | warn | 更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。 | service/user/admin.go:UpdateUser<br>service/user/profile.go:UpdateMyself |
| 1091 | 用户 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/user/admin.go:CreateUser<br>service/user/admin.go:DeleteUser<br>service/user/admin.go:Revoke<br>service/user/admin.go:UpdateUser<br>service/user/admin.go:UserList<br>service/user/profile.go:Base<br>service/user/profile.go:UpdateMyself<br>service/user/service.go:Login<br>service/user/service.go:Register |
| 1092 | 用户 | 创建失败 | Create Fail | warn | 创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。 | service/user/admin.go:CreateUser<br>service/user/service.go:Register |
| 1093 | 用户 | 登录失败 | Login Fail | warn | 登录流程失败，常见于签发 token 或写入登录信息时出错。查看 logs/runtime.log。 | service/user/service.go:Login |
| 1094 | 用户 | Token撤销失败 | Token Revocation Failed | warn | 撤销 token 失败，通常是数据库写入异常。查看 logs/runtime.log。 | service/user/admin.go:Revoke |
| 1095 | 用户 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/user/admin.go:DeleteUser |
| 2001 | 房间 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/room/handler.go:activatePost<br>app/room/handler.go:deactivatePost<br>app/room/handler.go:favoriteApplyPost<br>app/room/handler.go:favoriteDelete<br>app/room/handler.go:favoriteGet<br>app/room/handler.go:favoritePost<br>app/room/handler.go:favoritePut<br>app/room/handler.go:listGet<br>app/room/handler.go:roomDelete<br>app/room/handler.go:roomGet<br>app/room/handler.go:roomMetricsGet<br>app/room/handler.go:roomPost<br>app/room/handler.go:roomPut<br>app/room/handler.go:roomWorldsGet<br>app/room/handler.go:uploadPost<br>app/room/handler.go:webhookTestPost<br>app/room/handler.go:worldCustomStartupCmdPut<br>service/room/import.go:UploadTarget<br>service/room/lifecycle.go:Activate<br>service/room/lifecycle.go:Deactivate<br>service/room/lifecycle.go:Delete<br>service/room/query.go:Metrics<br>service/room/save.go:Create<br>service/room/save.go:CustomStartupCmdPut<br>service/room/save.go:Update |
| 2002 | 房间 | 非法URL | Invalid URL | warn | 提交的 URL 格式不合法（Webhook 等）。必须以 http:// 或 https:// 开头且可解析。 | service/room/favorite_snapshot.go:ValidateFavoriteData<br>service/room/save.go:Create<br>service/room/save.go:Update<br>service/room/service.go:WebhookTest |
| 2003 | 房间 | 世界名 [%s] 非法，只允许字母、数字、下划线、中划线和点，且不能以点或中划线开头 | World name [%s] is invalid, only letters, digits, underscore, hyphen and dot are allowed, and it cannot start with dot or hyphen | warn | 世界名会被拼进命令和文件路径，只允许字母、数字、下划线、中划线和点，且不能以点或中划线开头。修改世界名后重试。 | service/room/service.go:validateWorldNames |
| 2004 | 房间 | 世界名 [%s] 重复，同一房间内的世界名必须唯一 | World name [%s] is duplicated, world names in the same room must be unique | warn | 同一房间内的世界名必须唯一，否则多个世界会写入同一个存档目录。改成不同的名字。 | service/room/import_upload.go:handleUpload<br>service/room/service.go:validateWorldNames |
| 2005 | 房间 | 收藏中的游戏模式非法 | Invalid game mode in favorite | warn | 收藏里的游戏模式不是合法取值（可能来自旧版本数据）。删除该收藏后重新保存。 | service/room/favorite_snapshot.go:ValidateFavoriteData |
| 2006 | 房间 | 收藏中的 Webhook 配置非法 | Invalid webhook setting in favorite | warn | 收藏里的 Webhook 配置不是合法 JSON，或 URL 非法。修正后重新收藏。 | service/room/favorite_snapshot.go:ValidateFavoriteData |
| 2021 | 房间 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/room/basic.go:Worlds<br>service/room/favorite.go:FavoriteApply<br>service/room/favorite.go:FavoriteCreate<br>service/room/import.go:UploadTarget<br>service/room/lifecycle.go:Activate<br>service/room/lifecycle.go:Deactivate<br>service/room/query.go:Get<br>service/room/query.go:Metrics<br>service/room/save.go:Create<br>service/room/save.go:Update<br>service/room/service.go:CreateGuard |
| 2041 | 房间 | 房间不存在 | Room Not Found | warn | 房间不存在或已被删除。刷新房间列表后重试。 | service/diagnosis/service.go:Start<br>service/room/favorite.go:FavoriteCreate |
| 2042 | 房间 | 收藏不存在 | Favorite Not Found | warn | 收藏不存在、已被删除，或不属于当前账号。刷新收藏列表后重试。 | service/room/favorite.go:FavoriteApply<br>service/room/favorite.go:FavoriteDelete<br>service/room/favorite.go:FavoriteDetail<br>service/room/favorite.go:FavoriteUpdate |
| 2061 | 房间 | 端口 [%d] 冲突 | Port [%d] Conflict | warn | 端口与已有房间/世界冲突。换一个端口，或先停用占用该端口的房间。 | service/room/favorite.go:FavoriteApply<br>service/room/save.go:Create<br>service/room/save.go:Update |
| 2062 | 房间 | 写入文件失败 | write file fail | fatal | 配置文件写入磁盘失败。检查 DST 集群目录权限与磁盘空间，以及是否有异常的世界名。 | service/room/favorite.go:FavoriteApply<br>service/room/import.go:importRoomFromZip<br>service/room/save.go:Create<br>service/room/save.go:Update |
| 2063 | 房间 | 收藏中至少需要一个世界 | Favorite must contain at least one world | warn | 房间下没有任何世界，无法生成收藏快照。先给房间添加至少一个世界。 | service/room/favorite.go:FavoriteCreate<br>service/room/favorite_snapshot.go:ValidateFavoriteData |
| 2064 | 房间 | 收藏数据无效 | Invalid Favorite Data | warn | 收藏数据损坏或结构不兼容（旧版本数据）。删除该收藏重新保存。 | service/room/favorite.go:FavoriteApply<br>service/room/favorite.go:FavoriteDetail<br>service/room/favorite.go:FavoriteList<br>service/room/favorite.go:FavoriteUpdate |
| 2065 | 房间 | 房间名不能为空 | Room name is required | warn | 未提供房间名，且收藏里也没有可用名称。在弹窗里填写一个房间名。 | service/room/favorite.go:FavoriteApply |
| 2066 | 房间 | 激活失败 | Activate Fail | warn | 启用房间失败，常见于定时通知配置 JSON 解析失败。检查房间设置里的通知配置。 | service/room/lifecycle.go:Activate |
| 2067 | 房间 | 更新失败 | Update Fail | warn | 更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。 | service/room/save.go:CustomStartupCmdPut |
| 2068 | 房间 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/room/lifecycle.go:Delete |
| 2069 | 房间 | Webhook 测试失败: %s | Webhook Test Failed: %s | warn | Webhook 测试请求发送失败。检查目标地址可达性、证书与签名密钥。 | service/room/service.go:WebhookTest |
| 2070 | 房间 | 存档文件中没有发现可用的世界 | no available worlds found | warn | 上传的存档里没有解析出任何世界目录。确认 zip 内是完整的 Cluster 目录结构。 | service/room/import.go:importRoomFromZip |
| 2071 | 房间 | 上传存档世界个数与当前房间世界个数不相等 | the number of worlds does not match | warn | 上传存档的世界数量与当前房间不一致，无法覆盖导入。删除多余世界或改用新建房间导入。 | service/room/import.go:importRoomFromZip |
| 2072 | 房间 | 上传文件保存失败 | file save fail | warn | 上传文件落盘或解压失败。检查临时目录权限与磁盘空间，确认上传的是合法 zip。 | app/room/handler.go:uploadPost<br>service/room/import.go:UploadImport |
| 2073 | 房间 | 解压失败 | unzip file fail | warn | zip 解压失败。确认文件完整、未加密、未损坏。 | service/room/import_upload.go:handleUpload |
| 2074 | 房间 | 查询存档主目录失败 | find DST main path fail | warn | 解压后找不到包含 cluster.ini 的目录。确认 zip 的目录层级正确。 | service/room/import_upload.go:handleUpload |
| 2075 | 房间 | cluster.ini文件不存在 | cluster.ini file not found | warn | 存档根目录缺少 cluster.ini。确认这是 DST 的 Cluster 存档。 | service/room/import_upload.go:handleUpload |
| 2076 | 房间 | 读取cluster.ini文件失败 | read cluster.ini file fail | warn | cluster.ini 无法读取或编码异常。检查文件权限与内容格式。 | service/room/import_upload.go:handleUpload |
| 2077 | 房间 | cluster.ini中未发现[cluster_name]字段 | cluster_name not found in cluster.ini | warn | cluster.ini 里没有 cluster_name。补上房间名后重新导入。 | service/room/import_upload.go:handleUpload |
| 2078 | 房间 | cluster.ini中未发现[game_mode]字段 | game_mode not found in cluster.ini | warn | cluster.ini 里没有 game_mode。补上游戏模式后重新导入。 | service/room/import_upload.go:handleUpload |
| 2079 | 房间 | cluster.ini的[game_mode]取值非法 | invalid game_mode in cluster.ini | warn | cluster.ini 的 game_mode 不是受支持的取值（防 XSS 校验拦截）。改成合法模式。 | service/room/import_upload.go:handleUpload |
| 2080 | 房间 | server.ini文件不存在 | server.ini file not found | warn | 世界目录缺少 server.ini。确认每个世界目录结构完整。 | service/room/import_upload.go:handleUpload |
| 2081 | 房间 | 读取server.ini文件失败 | read server.ini file fail | warn | server.ini 无法读取。检查文件权限与内容格式。 | service/room/import_upload.go:handleUpload |
| 2082 | 房间 | server.ini中未发现[is_master]字段 | is_master not found in server.ini | warn | server.ini 里没有 is_master。补上主从标记后重新导入。 | service/room/import_upload.go:handleUpload |
| 2083 | 房间 | 读取server.ini[is_master]字段失败 | read server.ini[is_master] fail | warn | server.ini 的 is_master 不是合法布尔值。改成 true/false。 | service/room/import_upload.go:handleUpload |
| 2084 | 房间 | 未发现世界配置 | world level data not found | warn | 世界目录缺少 leveldataoverride.lua / worldgenoverride.lua。补上世界配置后重新导入。 | service/room/import_upload.go:handleUpload |
| 2085 | 房间 | 获取世界目录失败 | get worlds path fail | warn | 世界目录名非法或目录读取失败。目录名只允许字母数字和斜杠。 | service/room/import_upload.go:handleUpload |
| 2086 | 房间 | 存档中存在重复的世界名 | duplicated world name in the save archive | warn | 存档里存在重复的世界名，导入会互相覆盖存档。改名后重新导入。 | service/room/import_upload.go:handleUpload |
| 2087 | 房间 | 存档中存在非法世界名 | invalid world name in the save archive | warn | 存档里存在非法世界名（会被拼进命令和路径）。改名后重新导入。 | service/room/import_upload.go:handleUpload |
| 2091 | 房间 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/room/basic.go:AllBasic<br>service/room/basic.go:Factor<br>service/room/basic.go:PermittedBasic<br>service/room/basic.go:Worlds<br>service/room/favorite.go:FavoriteApply<br>service/room/favorite.go:FavoriteCreate<br>service/room/favorite.go:FavoriteDelete<br>service/room/favorite.go:FavoriteDetail<br>service/room/favorite.go:FavoriteList<br>service/room/favorite.go:FavoriteUpdate<br>service/room/import.go:importRoomFromZip<br>service/room/lifecycle.go:Activate<br>service/room/lifecycle.go:Deactivate<br>service/room/lifecycle.go:Delete<br>service/room/query.go:Get<br>service/room/query.go:List<br>service/room/query.go:Metrics<br>service/room/save.go:Create<br>service/room/save.go:CustomStartupCmdPut<br>service/room/save.go:Update<br>service/room/service.go:CreateGuard |
| 3001 | 模组 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/mod/handler.go:acfDelete<br>app/mod/handler.go:addDisablePost<br>app/mod/handler.go:addEnablePost<br>app/mod/handler.go:deletePost<br>app/mod/handler.go:deletePostMulti<br>app/mod/handler.go:downloadPost<br>app/mod/handler.go:downloadStatusGet<br>app/mod/handler.go:downloadedModIDsGet<br>app/mod/handler.go:downloadedModsGet<br>app/mod/handler.go:getEnabledModsGet<br>app/mod/handler.go:modInfoPost<br>app/mod/handler.go:modSearchGet<br>app/mod/handler.go:settingModConfigStructGet<br>app/mod/handler.go:settingModConfigValueGet<br>app/mod/handler.go:settingModConfigValuePut<br>service/mod/download.go:DownloadedModIDs<br>service/mod/download.go:ModDownload<br>service/mod/service.go:ModSearch |
| 3021 | 模组 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/mod/config.go:AddDisable<br>service/mod/config.go:AddEnable<br>service/mod/config.go:EnabledMods<br>service/mod/config.go:SettingModConfigStruct<br>service/mod/config.go:SettingModConfigValue<br>service/mod/config.go:SettingModConfigValuePut<br>service/mod/delete.go:Delete<br>service/mod/delete.go:DeleteAcf<br>service/mod/delete.go:DeleteMulti<br>service/mod/download.go:DownloadedModIDs<br>service/mod/download.go:DownloadedMods<br>service/mod/download.go:ModDownload |
| 3061 | 模组 | 获取模组信息失败 | Search Mod Fail | warn | 从 Steam 创意工坊查询模组失败。检查服务器到 Steam API 的网络与 Steam API Key。 | service/mod/config.go:ModInfo<br>service/mod/service.go:ModSearch |
| 3062 | 模组 | 获取模组配置信息失败 | Generate Mod Configuration Options Error | warn | 读取模组配置项失败。确认模组已下载完整（modoverrides.lua 可解析）。 | service/mod/config.go:SettingModConfigStruct |
| 3063 | 模组 | 获取模组配置失败 | Generate Mod Configurations Error | warn | 读取模组配置当前值失败。同上，检查模组文件是否完整。 | service/mod/config.go:SettingModConfigValue |
| 3064 | 模组 | 修改模组配置失败 | Modify Mod Configuration Error | warn | 写入模组配置失败。检查模组目录权限与磁盘空间。 | service/mod/config.go:SettingModConfigValuePut |
| 3065 | 模组 | 模组启用失败 | Mod Enable Fail | warn | 启用模组失败。确认模组已下载完成且 ID 正确。 | service/mod/config.go:AddEnable |
| 3066 | 模组 | 模组禁用失败 | Mod Disable Fail | warn | 禁用模组失败。查看 logs/runtime.log 中的具体错误。 | service/mod/config.go:AddDisable |
| 3067 | 模组 | 获取启用模组失败 | Get Enabled Mods Fail | warn | 读取已启用模组列表失败。确认 modoverrides.lua 存在且格式正确。 | service/mod/config.go:EnabledMods |
| 3068 | 模组 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/mod/delete.go:Delete<br>service/mod/delete.go:DeleteAcf<br>service/mod/delete.go:DeleteMulti |
| 3069 | 模组 | 下载失败 | Download Fail | warn | 模组下载失败或下载大小与预期不符。检查网络、下载地址与模组文件是否完整，查看 logs/runtime.log 中的具体错误。 | service/mod/download.go:ModDownload |
| 3091 | 模组 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/mod/config.go:AddDisable<br>service/mod/config.go:AddEnable<br>service/mod/config.go:EnabledMods<br>service/mod/config.go:SettingModConfigStruct<br>service/mod/config.go:SettingModConfigValue<br>service/mod/config.go:SettingModConfigValuePut<br>service/mod/delete.go:Delete<br>service/mod/delete.go:DeleteAcf<br>service/mod/delete.go:DeleteMulti<br>service/mod/download.go:DownloadedModIDs<br>service/mod/download.go:DownloadedMods<br>service/mod/download.go:ModDownload |
| 3092 | 模组 | 服务器内部错误 | Internal Server Error | fatal | 服务端内部异常，通常是被其他请求触发的空指针或未初始化状态。查看 logs/runtime.log。 | service/mod/download.go:ModDownload |
| 4001 | 控制台 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/dashboard/handler.go:checkLobbyPost<br>app/dashboard/handler.go:connectionCodeGet<br>app/dashboard/handler.go:connectionCodePut<br>app/dashboard/handler.go:execGamePost<br>app/dashboard/handler.go:infoBaseGet<br>service/dashboard/exec.go:ExecGame<br>service/dashboard/lobby.go:CheckLobby<br>service/dashboard/service.go:ConnectionCode<br>service/dashboard/service.go:InfoBase |
| 4021 | 控制台 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/dashboard/exec.go:ExecGame<br>service/dashboard/service.go:ConnectionCode<br>service/dashboard/service.go:ConnectionCodeUpdate<br>service/dashboard/service.go:InfoBase |
| 4061 | 控制台 | 执行失败 | Execute Fail | warn | 执行游戏操作失败。确认对应房间进程存在、screen 会话可用。 | service/dashboard/exec.go:ExecGame |
| 4062 | 控制台 | 启动失败 | Startup Fail | warn | 启动游戏失败。检查端口占用、DST 安装完整性、screen 是否可用。 | service/dashboard/exec.go:ExecGame |
| 4063 | 控制台 | 关闭失败 | Shutdown Fail | warn | 关闭游戏失败。进程可能已退出或 screen 会话丢失。 | service/dashboard/exec.go:ExecGame |
| 4064 | 控制台 | 重启失败 | Restart Fail | warn | 重启游戏失败。查看 logs/runtime.log 与 DST 自身日志。 | service/dashboard/exec.go:ExecGame |
| 4065 | 控制台 | 重置失败 | Reset Fail | warn | 重置存档失败。确认存档目录存在且可写。 | service/dashboard/exec.go:ExecGame |
| 4066 | 控制台 | 清空世界失败 | Delete Fail | warn | 删除游戏相关文件失败。确认没有进程占用这些文件。 | service/dashboard/exec.go:ExecGame |
| 4067 | 控制台 | 宣告失败 | Announce Fail | warn | 定时通知创建失败。检查通知内容的间隔与配置格式。 | service/dashboard/exec.go:ExecGame |
| 4068 | 控制台 | 通知失败 | System Message Send Fail | warn | 发送系统消息失败。确认世界进程正在运行。 | service/dashboard/exec.go:ExecGame |
| 4069 | 控制台 | 直连代码获取失败 | Get Connection Code Fail | warn | 更新直连信息（IP/端口）失败。检查端口是否被占用、IP 是否可达。 | service/dashboard/service.go:ConnectionCode |
| 4070 | 控制台 | 检查世界失败 | Check Lobby Fail | warn | 查询 Klei 大厅失败。检查服务器出境网络与 Klei 接口可用性。 | service/dashboard/lobby.go:CheckLobby |
| 4071 | 控制台 | 更新失败 | Update Fail | warn | 更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。 | service/dashboard/service.go:ConnectionCodeUpdate |
| 4091 | 控制台 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/dashboard/exec.go:ExecGame<br>service/dashboard/service.go:ConnectionCode<br>service/dashboard/service.go:ConnectionCodeUpdate<br>service/dashboard/service.go:InfoBase |
| 5001 | 平台 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/platform/handler.go:globalSettingsPost<br>app/platform/handler.go:metricsGet<br>app/platform/handler.go:pluginActionPost<br>app/platform/handler.go:pluginInstallPost<br>app/platform/handler.go:pluginListGet<br>app/platform/handler.go:pluginStatusGet<br>app/platform/handler.go:screenKillPost<br>app/platform/handler.go:screenRunningGet<br>app/platform/handler.go:webhookTestPost<br>service/platform/plugin.go:PluginInstall<br>service/platform/plugin.go:preparePluginAction<br>service/platform/plugin_chat.go:installAiChatPlugin<br>service/platform/plugin_tmi.go:installTmiPlugin<br>service/platform/service.go:KillScreen<br>service/platform/service.go:ScreenRunning<br>service/platform/settings.go:GlobalSettingsPost |
| 5002 | 平台 | 非法URL | Invalid URL | warn | 提交的 URL 格式不合法（Webhook 等）。必须以 http:// 或 https:// 开头且可解析。 | service/platform/service.go:WebhookTest<br>service/platform/settings.go:GlobalSettingsPost |
| 5061 | 平台 | 安装失败: %s | Install Failed: %s | warn | 插件安装失败。检查依赖（stex 等）是否安装、目标目录权限与网络。 | service/platform/plugin_chat.go:installAiChatPlugin<br>service/platform/plugin_tmi.go:installTmiPlugin |
| 5062 | 平台 | 卸载失败 | Uninstall Fail | warn | 插件卸载失败。确认插件目录存在且未被占用。 | service/platform/plugin.go:preparePluginAction |
| 5063 | 平台 | 更新失败 | Update Fail | warn | 更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。 | service/platform/plugin.go:applyPluginRuntimeAction<br>service/platform/plugin.go:updateTmiPlugin<br>service/platform/settings.go:GlobalSettingsPost |
| 5064 | 平台 | 获取系统信息失败 | Get OS Info Fail | warn | 读取系统信息失败。检查 /proc 或系统命令是否可用（容器环境常见）。 | service/platform/overview.go:OSInfoGet |
| 5065 | 平台 | 获取Screens失败 | Get Screens Fail | warn | 列出 screen 会话失败。确认已安装 screen 且当前用户可执行。 | service/platform/service.go:ScreenRunning |
| 5066 | 平台 | 关闭Screens失败 | Kill Screens Fail | warn | 结束 screen 会话失败。确认会话存在且属于当前用户。 | service/platform/service.go:KillScreen |
| 5067 | 平台 | Webhook 测试失败: %s | Webhook Test Failed: %s | warn | Webhook 测试请求发送失败。检查目标地址可达性、证书与签名密钥。 | service/platform/service.go:WebhookTest |
| 5091 | 平台 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/platform/plugin.go:PluginAction<br>service/platform/plugin.go:PluginList<br>service/platform/plugin.go:PluginStatus<br>service/platform/plugin.go:updatePlugin<br>service/platform/plugin.go:updateTmiPlugin<br>service/platform/plugin_chat.go:installAiChatPlugin<br>service/platform/plugin_tmi.go:installTmiPlugin<br>service/platform/service.go:ReadmeGet<br>service/platform/service.go:ReadmePost<br>service/platform/service.go:ScreenRunning<br>service/platform/settings.go:GlobalSettingsGet<br>service/platform/settings.go:GlobalSettingsPost |
| 6001 | 日志 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/logs/handler.go:cleanDelete<br>app/logs/handler.go:cleanInfoGet<br>app/logs/handler.go:contentGet<br>app/logs/handler.go:contentStreamGet<br>app/logs/handler.go:historyContentGet<br>app/logs/handler.go:historyListGet<br>service/logs/service.go:contentLogPath |
| 6021 | 日志 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/logs/service.go:Clean<br>service/logs/service.go:CleanInfo<br>service/logs/service.go:Download<br>service/logs/service.go:HistoryContent<br>service/logs/service.go:HistoryList<br>service/logs/service.go:contentLogPath |
| 6061 | 日志 | 下载失败 | Download Fail | warn | 日志打包或读取失败。检查临时目录 dmp_files/tmp 的权限与磁盘空间。 | service/logs/service.go:Download |
| 6062 | 日志 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/logs/service.go:Clean |
| 6091 | 日志 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/logs/service.go:Clean<br>service/logs/service.go:CleanInfo<br>service/logs/service.go:Download<br>service/logs/service.go:HistoryContent<br>service/logs/service.go:HistoryList<br>service/logs/service.go:contentLogPath |
| 7001 | 工具 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/tools/handler.go:aiBaseSettingPut<br>app/tools/handler.go:aiSettingGet<br>app/tools/handler.go:aiSettingPut<br>app/tools/handler.go:announceGet<br>app/tools/handler.go:announcePut<br>app/tools/handler.go:backupDelete<br>app/tools/handler.go:backupDownloadGet<br>app/tools/handler.go:backupGet<br>app/tools/handler.go:backupPost<br>app/tools/handler.go:backupRestorePost<br>app/tools/handler.go:categoryItemsGet<br>app/tools/handler.go:consolePost<br>app/tools/handler.go:mapGet<br>app/tools/handler.go:mapPlayersGet<br>app/tools/handler.go:snapshotDelete<br>app/tools/handler.go:snapshotGet<br>app/tools/handler.go:tokenPost<br>service/tools/announce.go:AnnounceGet<br>service/tools/announce.go:AnnouncePut<br>service/tools/backup.go:BackupCreate<br>service/tools/backup.go:BackupDelete<br>service/tools/backup.go:BackupDownload<br>service/tools/backup.go:BackupList<br>service/tools/backup.go:BackupRestore<br>service/tools/map.go:Map<br>service/tools/map.go:mapWorldContext<br>service/tools/snapshot.go:SnapshotDelete<br>service/tools/snapshot.go:SnapshotGet<br>service/tools/token.go:Token |
| 7002 | 工具 | 嵌入模型 Base URL 不合法 | Embedding model Base URL is invalid | warn | 向量服务的 Base URL 不合法。填写完整的 http(s) 地址。 | service/tools/aichat.go:AIEmbeddingIndexRebuild |
| 7003 | 工具 | 当前没有正在进行的嵌入索引构建 | No embedding index build is running | warn | 当前没有正在运行的向量索引构建任务，无需取消。 | service/tools/aichat.go:AIEmbeddingIndexCancel |
| 7004 | 工具 | 嵌入模型配置不完整 | Embedding model configuration is incomplete | warn | 向量模型配置不完整（缺少模型名、维度或 API Key）。补全后重试。 | service/tools/aichat.go:AIEmbeddingIndexRebuild |
| 7005 | 工具 | 向量模型配置不合法 | embedding config invalid | warn | 向量模型配置不合法（Base URL、模型名、维度或 API Key）。按提示补全后重试。 | service/tools/aichat.go:AIBaseSettingPut<br>service/tools/aichat.go:AISettingPut |
| 7021 | 工具 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/tools/aichat.go:AISettingGet<br>service/tools/aichat.go:AISettingPut<br>service/tools/announce.go:AnnounceGet<br>service/tools/announce.go:AnnouncePut<br>service/tools/backup.go:BackupCreate<br>service/tools/backup.go:BackupDelete<br>service/tools/backup.go:BackupDownload<br>service/tools/backup.go:BackupList<br>service/tools/backup.go:BackupRestore<br>service/tools/map.go:mapWorldContext<br>service/tools/snapshot.go:SnapshotDelete<br>service/tools/snapshot.go:SnapshotGet<br>service/tools/tmi.go:Console |
| 7041 | 工具 | 存档不存在，请先启动一次世界 | save file not found | warn | 地图存档文件不存在。确认该世界已生成过地图存档。 | service/tools/map.go:Map |
| 7061 | 工具 | 创建备份文件失败 | create backup fail | warn | 创建备份失败。检查备份目录权限与磁盘空间。 | service/tools/backup.go:BackupCreate |
| 7062 | 工具 | 获取备份文件失败 | get backup fail | warn | 读取备份列表失败。检查备份目录是否存在。 | service/tools/backup.go:BackupList |
| 7063 | 工具 | 恢复失败 | restore fail | warn | 恢复备份失败。确认备份文件完整、没有进程占用存档目录。 | service/tools/backup.go:BackupRestore |
| 7064 | 工具 | 创建失败 | Create Fail | warn | 创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。 | service/tools/token.go:Token |
| 7065 | 工具 | 更新失败 | Update Fail | warn | 更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。 | service/tools/aichat.go:AIBaseSettingPut<br>service/tools/aichat.go:AIEmbeddingIndexRebuild<br>service/tools/aichat.go:AIKeywordIndexRebuild<br>service/tools/aichat.go:AISettingPut<br>service/tools/announce.go:AnnouncePut |
| 7066 | 工具 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/tools/snapshot.go:SnapshotDelete |
| 7067 | 工具 | 执行失败 | Execute Fail | warn | 执行游戏操作失败。确认对应房间进程存在、screen 会话可用。 | service/tools/tmi.go:Console |
| 7068 | 工具 | 获取游戏存档失败 | Get Snapshot Fail | warn | 读取快照列表失败。检查存档目录与 DST 快照文件。 | service/tools/snapshot.go:SnapshotGet |
| 7069 | 工具 | 地图生成繁忙，请稍后重试 | map generation is busy | fatal | 地图正在生成中，请等当前任务结束后再试。 | service/tools/map.go:Map<br>service/tools/map.go:MapPlayers |
| 7070 | 工具 | 解析存档失败 | parse save fail | warn | 解析地图数据失败。存档可能损坏或版本不兼容。 | service/tools/map.go:Map |
| 7071 | 工具 | 快照已删除，但重新启动游戏失败 | Snapshot deleted, but restarting the game failed | warn | 快照已删除，但房间世界重新启动失败。检查 DST 安装、screen 会话与端口占用，确认世界进程是否恢复运行。 | service/tools/snapshot.go:SnapshotDelete |
| 7091 | 工具 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/tools/aichat.go:AIBaseSettingGet<br>service/tools/aichat.go:AIBaseSettingPut<br>service/tools/aichat.go:AIEmbeddingIndexRebuild<br>service/tools/aichat.go:AISettingGet<br>service/tools/aichat.go:AISettingPut<br>service/tools/announce.go:AnnounceGet<br>service/tools/announce.go:AnnouncePut<br>service/tools/backup.go:BackupCreate<br>service/tools/backup.go:BackupDelete<br>service/tools/backup.go:BackupList<br>service/tools/backup.go:BackupRestore<br>service/tools/map.go:mapWorldContext<br>service/tools/snapshot.go:SnapshotDelete<br>service/tools/snapshot.go:SnapshotGet<br>service/tools/tmi.go:Console<br>service/tools/tmi.go:TMICategoryItems<br>service/tools/tmi.go:TMICategoryList |
| 8001 | 玩家 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/player/handler.go:chatGet<br>app/player/handler.go:ganttGet<br>app/player/handler.go:listGet<br>app/player/handler.go:listPost<br>app/player/handler.go:onlineGet<br>app/player/handler.go:statisticsOnlineTimeGet<br>app/player/handler.go:statisticsPlayerCountGet<br>app/player/handler.go:statisticsPlayerCountV2Get<br>app/player/handler.go:uidMapGet<br>service/player/list.go:List<br>service/player/list.go:UpdateList<br>service/player/service.go:Chat<br>service/player/service.go:Online<br>service/player/service.go:UIDMap<br>service/player/statistics.go:Gantt<br>service/player/statistics.go:StatisticsOnlineTime<br>service/player/statistics.go:StatisticsPlayerCount<br>service/player/statistics.go:StatisticsPlayerCountV2 |
| 8021 | 玩家 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。 | service/player/list.go:List<br>service/player/list.go:UpdateList<br>service/player/service.go:Chat<br>service/player/service.go:Online<br>service/player/service.go:UIDMap<br>service/player/statistics.go:Gantt<br>service/player/statistics.go:StatisticsOnlineTime<br>service/player/statistics.go:StatisticsPlayerCount<br>service/player/statistics.go:StatisticsPlayerCountV2 |
| 8061 | 玩家 | 添加失败 | Add Fail | warn | 添加失败。确认目标（白名单/黑名单/管理员）内容格式正确。 | service/player/list.go:UpdateList |
| 8062 | 玩家 | 删除失败 | Delete Fail | warn | 删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。 | service/player/list.go:UpdateList |
| 8063 | 玩家 | 玩家聊天信息获取失败 | get chat message fail | warn | 读取聊天记录失败。确认聊天日志已生成且可读。 | service/player/service.go:Chat |
| 8091 | 玩家 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/player/list.go:List<br>service/player/list.go:UpdateList<br>service/player/service.go:Chat<br>service/player/service.go:UIDMap<br>service/player/statistics.go:Gantt<br>service/player/statistics.go:StatisticsPlayerCount<br>service/player/statistics.go:StatisticsPlayerCountV2 |
| 9001 | 云服务 | 请求参数错误 | Bad Request | warn | 请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。 | app/cloud/handler.go:archiveUploadPost<br>app/cloud/handler.go:linkPost<br>app/cloud/handler.go:loginPost<br>app/cloud/handler.go:syncPost<br>app/cloud/handler.go:unlinkPost<br>service/cloud/account.go:Link<br>service/cloud/login.go:Login |
| 9002 | 云服务 | 云存档 ID 非法 | Invalid cloud archive ID | warn | 云存档 ID 格式非法。刷新列表后重试。 | service/cloud/archive.go:ArchiveDownload<br>service/cloud/archive.go:ArchiveImport |
| 9021 | 云服务 | 没有该房间的操作权限 | No permission for this room | warn | 当前账号没有该房间权限，无法上传云存档。 | service/cloud/archive.go:ArchiveUpload |
| 9022 | 云服务 | 没有创建房间的权限 | No permission to create rooms | warn | 当前账号没有创建房间的权限，无法导入云存档。 | service/cloud/archive.go:ArchiveImport |
| 9061 | 云服务 | 该云账号尚未绑定本地账号，请先用本地账号登录并在设置中绑定 | This cloud account is not linked to a local account yet | warn | 本地账号还没有绑定云账号。先到「云服务」页面完成绑定。 | service/cloud/login.go:authenticateLinkedCloudUser<br>service/cloud/login.go:bootstrapCloudUser |
| 9062 | 云服务 | 该云账号已绑定到本机其他账号 | This cloud account is already linked to another local account | warn | 该云账号已经绑定了别的本地账号。先解绑或换一个云账号。 | service/cloud/account.go:Link |
| 9063 | 云服务 | 当前账号尚未绑定云账号 | No cloud account is linked | warn | 云账号绑定不存在或已被解绑。重新绑定后再试。 | service/cloud/account.go:Unlink<br>service/cloud/archive.go:ArchiveUpload<br>service/cloud/service.go:requireLink<br>service/cloud/sync.go:Sync |
| 9064 | 云服务 | 绑定的本地账号不存在，请重新绑定 | The linked local account no longer exists | warn | 云账号对应的本地用户不存在（可能已被删除）。重新绑定或重建本地用户。 | service/cloud/account.go:Link<br>service/cloud/login.go:authenticateLinkedCloudUser<br>service/cloud/login.go:fallbackLocalLogin |
| 9065 | 云服务 | 本地账号已被禁用 | Local account is disabled | warn | 云账号对应的本地用户已被禁用。让管理员解除禁用。 | service/cloud/login.go:authenticateLinkedCloudUser<br>service/cloud/login.go:fallbackLocalLogin |
| 9066 | 云服务 | 密码错误 | Wrong password | warn | 云账号或本地密码错误。确认输入；多次失败会被限流。 | service/cloud/account.go:Unlink<br>service/cloud/login.go:fallbackLocalLogin |
| 9067 | 云服务 | 云账号关联已失效，请重新绑定 | Cloud link expired, please link again | warn | 云账号的登录凭据已失效（token 被撤销或密码变更）。重新绑定云账号。 | service/cloud/service.go:failCloud |
| 9068 | 云服务 | 云服务暂时不可用，请稍后重试 | Cloud service is temporarily unavailable | fatal | 云服务不可达或返回异常。检查服务器到云服务的网络，稍后重试；本地登录仍可用。 | service/cloud/login.go:fallbackLocalLogin<br>service/cloud/service.go:failCloud |
| 9069 | 云服务 | 云同步失败 | Cloud sync failed | warn | 云同步失败。查看响应 message 与 logs/runtime.log 中的具体原因。 | service/cloud/sync.go:Sync |
| 9070 | 云服务 | 不支持的同步方向 | Unsupported sync direction | warn | 同步方向参数非法，只能是 both/upload/download。 | service/cloud/sync.go:Sync |
| 9071 | 云服务 | 存档压缩包超过 50MB，无法上传 | The archive exceeds 50MB | warn | 云存档超过大小上限。清理日志或快照后重新上传。 | service/cloud/archive.go:ArchiveImport<br>service/cloud/archive.go:ArchiveUpload |
| 9072 | 云服务 | 云存档是旗舰版专属功能，升级到旗舰版后即可使用 | Cloud archives require the ultimate plan | warn | 云存档是旗舰套餐功能。升级套餐后重试。 | service/cloud/archive.go:ArchiveUpload |
| 9073 | 云服务 | 打包存档失败 | Failed to package the save archive | warn | 打包云存档失败。检查临时目录权限与磁盘空间。 | service/cloud/archive.go:ArchiveUpload |
| 9074 | 云服务 | 云存档导入失败 | Cloud archive import failed | warn | 导入云存档失败。存档可能损坏或与当前版本不兼容。 | service/cloud/archive.go:ArchiveImport |
| 9075 | 云服务 | 下载云存档失败 | Failed to download cloud archive | warn | 从云服务下载存档失败。检查网络与云账号状态后重试。 | service/cloud/archive.go:ArchiveImport |
| 9076 | 云服务 | 云服务返回错误 | Cloud service error | warn | 云服务返回了业务错误，具体原因见 message（由云端给出，未经过本地翻译）。 | service/cloud/service.go:failCloud |
| 9091 | 云服务 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。 | service/cloud/account.go:Link<br>service/cloud/account.go:Status<br>service/cloud/account.go:Unlink<br>service/cloud/archive.go:ArchiveUpload<br>service/cloud/login.go:Login<br>service/cloud/login.go:authenticateLinkedCloudUser<br>service/cloud/login.go:bootstrapCloudUser<br>service/cloud/login.go:fallbackLocalLogin<br>service/cloud/service.go:requireLink |
| 9092 | 云服务 | 云账号登录失败 | Cloud login failed | warn | 云登录失败，常见于签发本地 token 或写入绑定信息时出错。查看 logs/runtime.log。 | service/cloud/login.go:bootstrapCloudUser<br>service/cloud/login.go:fallbackLocalLogin<br>service/cloud/login.go:loginLinkedCloudUser |
| 9093 | 云服务 | 创建失败 | Create Fail | warn | 创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。 | service/cloud/login.go:bootstrapCloudUser |
| 10001 | AI 诊断 | 请求参数错误 | Bad Request | warn | 诊断请求参数缺失或取值非法。核对 scope、roomID、actionIDs 等字段。 | app/diagnosis/handler.go:applyPost<br>app/diagnosis/handler.go:historyGet<br>app/diagnosis/handler.go:settingPut<br>app/diagnosis/handler.go:startPost<br>service/diagnosis/apply.go:Apply<br>service/diagnosis/service.go:Start |
| 10002 | AI 诊断 | 异常描述过长，最多 1000 个字符 | The issue description is too long; 1000 characters maximum | warn | 异常描述最长 1000 个字符。精简描述后重新发起诊断。 | service/diagnosis/service.go:Start |
| 10021 | AI 诊断 | 权限不足 | Insufficient Permissions | auth | 当前账号没有该房间的诊断权限，或尝试执行管理员专属动作。 | service/diagnosis/actions.go:applyAction<br>service/diagnosis/apply.go:Apply<br>service/diagnosis/service.go:AISettingGet<br>service/diagnosis/service.go:AISettingPut<br>service/diagnosis/service.go:ApplyStream<br>service/diagnosis/service.go:Cancel<br>service/diagnosis/service.go:Get<br>service/diagnosis/service.go:Start<br>service/diagnosis/service.go:Stream |
| 10041 | AI 诊断 | 诊断记录不存在或已被清理 | Diagnosis not found | warn | 诊断会话不存在或已被清理。重新发起一次诊断。 | service/diagnosis/apply.go:Apply<br>service/diagnosis/service.go:ApplyStream<br>service/diagnosis/service.go:Cancel<br>service/diagnosis/service.go:Get<br>service/diagnosis/service.go:HistoryDetail<br>service/diagnosis/service.go:Stream |
| 10042 | AI 诊断 | 诊断已过期，请重新诊断 | Diagnosis expired, please run it again | warn | 诊断会话已过期，修复动作不能继续执行。重新诊断后再操作。 | service/diagnosis/apply.go:Apply |
| 10043 | AI 诊断 | 诊断尚未完成，请稍后再试 | Diagnosis is not ready yet | warn | 诊断仍在进行或尚未成功结束。等待诊断完成后再执行修复。 | service/diagnosis/apply.go:Apply |
| 10061 | AI 诊断 | 尚未配置 AI 诊断专用大模型，请先在 AI 诊断的模型设置中配置 | No AI model is configured for AI diagnosis. Configure it in AI diagnosis model settings first | warn | 诊断未配置专用大模型。先在工具-AI 诊断-模型设置中填写 Chat Base URL、API Key 与模型名；诊断与 AI 对话配置相互独立，不会回退。 | service/diagnosis/service.go:Start<br>service/diagnosis/service.go:resolveChatConfig |
| 10062 | AI 诊断 | 诊断任务正在进行中，请稍后再试 | A diagnosis is already running, please try again later | warn | 同一账号或房间已有诊断在运行，或请求过于频繁。稍后重试。 | service/diagnosis/service.go:Start |
| 10063 | AI 诊断 | 诊断大模型配置不完整或不合法 | The diagnosis AI configuration is incomplete or invalid | warn | 诊断专用大模型配置不完整或不合法。检查 Base URL、API Key 与模型名。 | service/diagnosis/service.go:AISettingPut<br>service/diagnosis/service.go:resolveChatConfig |
| 10064 | AI 诊断 | 动作不在本次诊断的候选列表中 | The action is not part of this diagnosis | warn | 动作不在本次诊断的候选中，或动作 ID 不受支持。刷新诊断结果后重试。 | service/diagnosis/actions.go:applyAction<br>service/diagnosis/apply.go:Apply |
| 10065 | AI 诊断 | 动作参数或当前服务器状态不满足执行条件 | The action parameters or current server state do not meet the requirements | warn | 动作参数或当前服务器状态不满足执行条件。重新诊断并确认证据。 | service/diagnosis/actions.go:applyAction<br>service/diagnosis/apply.go:Apply |
| 10066 | AI 诊断 | 高风险动作需要二次确认，请输入正确的确认文本 | This action requires confirmation with the correct text | warn | 高风险动作需要二次确认，确认文本必须与房间名或动作短语一致。 | service/diagnosis/apply.go:Apply |
| 10067 | AI 诊断 | 该房间已有修复任务正在执行 | A repair task is already running for this room | warn | 该房间已有修复任务在执行。等待完成后再提交。 | service/diagnosis/apply.go:Apply<br>service/diagnosis/service.go:Cancel |
| 10091 | AI 诊断 | AI 调用失败，请检查模型配置或稍后重试 | AI request failed. Check the model settings or retry later | fatal | 调用大模型失败。检查 Base URL、API Key、模型是否支持 tools，或稍后重试。 | service/diagnosis/agent.go:completeDiagnosis<br>service/diagnosis/agent.go:runDiagnosis |
| 10092 | AI 诊断 | 修复执行失败 | Repair failed | warn | 修复动作执行失败。查看动作结果与 logs/runtime.log 中的具体错误。 | service/diagnosis/actions.go:applyAction |
| 10093 | AI 诊断 | 修复执行后复检未通过 | Verification failed after the repair | warn | 动作已执行但复检未通过。重新诊断，确认是否需要人工介入。 | service/diagnosis/actions.go:applyAction |
| 10094 | AI 诊断 | 数据库连接失败 | Database Connection Error | fatal | 数据库读写失败。检查数据目录磁盘空间与文件权限。 | service/diagnosis/agent.go:runDiagnosis<br>service/diagnosis/apply.go:Apply<br>service/diagnosis/service.go:AISettingGet<br>service/diagnosis/service.go:AISettingPut<br>service/diagnosis/service.go:Get<br>service/diagnosis/service.go:History<br>service/diagnosis/service.go:HistoryDetail<br>service/diagnosis/service.go:Start<br>service/diagnosis/service.go:resolveChatConfig |
| 10095 | AI 诊断 | 诊断执行异常中断，请重试 | The diagnosis stopped unexpectedly, please retry | fatal | 诊断协程内部异常（panic）被兜底终止。查看 logs/runtime.log 中的调用栈，按提示重新发起诊断。 | service/diagnosis/agent.go:runDiagnosis |
