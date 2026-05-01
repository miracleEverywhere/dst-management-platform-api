---
name: no-build
description: 修改代码后禁止执行 go build、go run 等编译/运行命令。本项目目标平台是 Linux 服务器，代码修改后由用户自行部署测试。
---

# 禁止本地编译/运行

修改代码后**禁止**执行以下命令：
- `go build`
- `go run`
- `go test`
- `go install`
- `go vet`（需要编译）
- 任何会触发 Go 编译的命令

**原因**：
1. 目标平台是 Linux x86_64，代码修改后由用户自行部署到 Linux 服务器上测试
2. 已使用纯 Go 实现 SQLite 驱动，无需 CGO

**允许的操作**：
- `go fmt`（格式化，不编译）
- 静态分析工具（不涉及编译的）
- 编辑、搜索、查看代码
