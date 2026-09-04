---
name: sandbox
description: 在运行编译时，需要把 Go 缓存切到 /tmp目录。
---

在进行编译时，你需要把 Go 缓存切到 /tmp目录，因为沙箱不允许写入系统 Go 缓存而失败
