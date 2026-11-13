package cache

// CloudKey 云凭据 AES-GCM 主密钥，启动时由 SystemDAO 从 system 表载入。
var CloudKey []byte
