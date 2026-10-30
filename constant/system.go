package constant

import "time"

// Version 平台版本号
const Version = "v3.2.2"

// ApiVersion 接口版本号
const ApiVersion = "v3"

// HttpTimeout HTTP请求超时时间
const HttpTimeout = 30

// JwtExpirationHours 登录令牌过期时间
const JwtExpirationHours = 24 * 3

// JwtSecretBytes JWT 密钥随机字节数
const JwtSecretBytes = 32

// StaticCacheHours 静态资源缓存时间
const StaticCacheHours = 24 * 7

// SteamLogPath steam日志
const SteamLogPath = "Steam/logs/bootstrap_log.txt"

// DarwinSteamLogPath macos steam日志
const DarwinSteamLogPath = "steamcmd/logs/console_log.txt"

// InternetIPApi1 公网IP查询接口
const InternetIPApi1 = "http://ip-api.com/json/?lang=zh-CN"

// InternetIPApi2 公网IP查询接口
const InternetIPApi2 = "http://cip.cc"

// Darwin 系统名
const Darwin = "darwin"

// LoginRateLimitInterval 单个用户登录限流间隔
const LoginRateLimitInterval = time.Second

// LoginRateLimitCleanupInterval 登录限流记录清理间隔
const LoginRateLimitCleanupInterval = 5 * time.Minute

// LoginRateLimitMaxEntries 登录限流记录的最大条目数
const LoginRateLimitMaxEntries = 2 << 15

// LoginBodyMaxBytes 登录请求体读取上限，仅用于提取用户名
const LoginBodyMaxBytes = 8 << 10
