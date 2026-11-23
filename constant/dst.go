package constant

// GameMainPath 游戏主目录
const GameMainPath = "dst"

// GameUgcPath 游戏ugc模组目录
const GameUgcPath = GameMainPath + "/ugc_mods"

// GameNotUgcPath 游戏非ugc模组目录
const GameNotUgcPath = GameMainPath + "/mods"

// GameModSettingPath 自动下载mod配置文件
const GameModSettingPath = GameNotUgcPath + "/dedicated_server_mods_setup.lua"

// BannedModsFileName 全局模组屏蔽名单文件名。
// 该文件位于存档根目录（与集群目录同级），内容为每行一个模组 ID，对本机所有集群生效。
const BannedModsFileName = "banned_mods.txt"

// DSTLocalVersionPath 饥荒版本文件
const DSTLocalVersionPath = "dst/version.txt"

// DSTServerVersionKlei 饥荒版本查询页面
const DSTServerVersionKlei = "https://kleiforums.com/game-updates/dst/"

// DSTServerVersionApi1 DSTServerVersionApi2 饥荒版本api
const DSTServerVersionApi1 = "http://ver.tugos.cn/getLocalVersion"
const DSTServerVersionApi2 = "https://dstv.3moredays.com/version.txt"

// SteamApiModDetail 模组详情接口
const SteamApiModDetail = "http://api.steampowered.com/IPublishedFileService/GetDetails/v1/"

// SteamApiModSearch 模组查询接口
const SteamApiModSearch = "http://api.steampowered.com/IPublishedFileService/QueryFiles/v1/"

// ========== Klei 游戏大厅 ==========

// KleiLobbyListURL 大厅房间列表接口（按区域；文件名为 .json.gz，响应可能是 gzip 或已解压的 JSON）
const KleiLobbyListURL = "https://lobby-v2-cdn.klei.com/%s-Steam.json.gz"

// KleiLobbyDetailURL 大厅房间详情接口（按区域）
const KleiLobbyDetailURL = "https://lobby-v2-%s.klei.com/lobby/read"

// KleiLobbyToken 大厅详情接口所需令牌，与 Klei 官方大厅网页请求一致
const KleiLobbyToken = "pds-g^KU_JpEFW0P3^GLDPpmMYhgm/Bd254Xwi9MqRouHutahdIfGzv4Nv+zE="

// KleiLobbyGameID 大厅详情请求体中的 __gameId
const KleiLobbyGameID = "DontStarveTogether"

// KleiLobbyTimeout Klei 大厅请求超时时间（秒）
const KleiLobbyTimeout = 15

// Klei 大厅区域，Klei 可能随时调整，改动时同步前端区域展示
const (
	KleiLobbyRegionAPEast1      = "ap-east-1"
	KleiLobbyRegionAPSoutheast1 = "ap-southeast-1"
	KleiLobbyRegionUSEast1      = "us-east-1"
	KleiLobbyRegionEUCentral1   = "eu-central-1"
)

const DstBlockList = "https://dst-block.miraclesses.top/api/blacklist/public"

// DmpFiles 平台文件根目录
const DmpFiles = "dmp_files"

// UserModsPath 用户模组缓存目录，按用户名分目录保存，不参与临时模组清理
const UserModsPath = DmpFiles + "/mods/users"

const PluginPath = DmpFiles + "/plugins"

const PluginTmiPath = PluginPath + "/tmi"

const TmirID = 3638290455

// PluginAiChatPath ai聊天插件根目录
const PluginAiChatPath = PluginPath + "/ai_chat"

// PluginAiChatWikiPath ai聊天wiki数据
const PluginAiChatWikiPath = PluginAiChatPath + "/wiki"

// PluginAiChatSearchDataPath ai聊天搜索用数据
const PluginAiChatSearchDataPath = PluginAiChatPath + "/search_data"

// ========== Darwin ==========

// DarwinDstContentsPath 游戏主目录
const DarwinDstContentsPath = "dst/dontstarve_dedicated_server_nullrenderer.app/Contents"

// DarwinDstBinDir macos下游戏二进制文件目录
const DarwinDstBinDir = DarwinDstContentsPath + "/MacOS"

// DarwinGameModSettingPath dedicated_server_mods_setup.lua文件
const DarwinGameModSettingPath = DarwinGameNotUgcPath + "/dedicated_server_mods_setup.lua"

// DarwinGameUgcPath macos ugc模组目录
const DarwinGameUgcPath = DarwinDstContentsPath + "/ugc_mods"

// DarwinGameNotUgcPath macos 非ugc模组目录
const DarwinGameNotUgcPath = DarwinDstContentsPath + "/mods"
