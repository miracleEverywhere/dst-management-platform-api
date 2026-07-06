package utils

// Version 平台版本号
const Version = "v3.1.6"

// ApiVersion 接口版本号
const ApiVersion = "v3" //

// HttpTimeout HTTP请求超时时间
const HttpTimeout = 30 //

// JwtExpirationHours 登录令牌过期时间
const JwtExpirationHours = 24 * 3

// StaticCacheHours 静态资源缓存时间
const StaticCacheHours = 24 * 7

// GameModSettingPath 自动下载mod配置文件
const GameModSettingPath = "dst/mods/dedicated_server_mods_setup.lua"

// DSTLocalVersionPath 饥荒版本文件
const DSTLocalVersionPath = "dst/version.txt"

// DSTServerVersionApi 饥荒版本查询页面
const DSTServerVersionApi = "https://forums.kleientertainment.com/game-updates/dst"

// InternetIPApi1 公网IP查询接口
const InternetIPApi1 = "http://ip-api.com/json/?lang=zh-CN"

// InternetIPApi2 公网IP查询接口
const InternetIPApi2 = "http://cip.cc"

// SteamApiModDetail 模组详情接口
const SteamApiModDetail = "http://api.steampowered.com/IPublishedFileService/GetDetails/v1/"

// SteamApiModSearch 模组查询接口
const SteamApiModSearch = "http://api.steampowered.com/IPublishedFileService/QueryFiles/v1/"

// ClusterPath 饥荒存档根目录
const ClusterPath = ".klei/DoNotStarveTogether"

// DmpFiles 平台文件根目录
const DmpFiles = "dmp_files"

const PluginPath = DmpFiles + "/plugins"

const PluginTmiPath = PluginPath + "/tmi"

const TmirID = 3638290455
