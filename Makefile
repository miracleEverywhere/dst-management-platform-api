#################################################################
#																#
#	This file is intended for the development environment only	#
#		The actual build is performed in GitHub Actions			#
#			该文件仅用于开发环境，实际编译在Github Action中			#
#																#
#################################################################

FRONTEND_DIR := $(HOME)/WebstormProjects/dst-management-platform-web
EMBED_DIR := embedFS/dist
LITE_ASSET := lite/dist-lite
OUT_DIR := .
RM := rm -rf
CGO := CGO_ENABLED=0


ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL := powershell.exe
	RM = rm -r
	FRONTEND_DIR = $(USERPROFILE)\WebstormProjects\dst-management-platform-web
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        DETECTED_OS := Linux
    else ifeq ($(UNAME_S),Darwin)
        DETECTED_OS := MacOS
        OUT_DIR = $(HOME)/dmp
    else
        $(error Unsupported OS: $(UNAME_S))
    endif
endif

.PHONY: all frontend2backend backend-only frontend-only lite help codes codes-verify check

# 构建前端 构建后端
all: frontend-only copy-frontend backend-only

# 构建前端并将产物复制到后端
frontend2backend: frontend-only copy-frontend

lite-frontend2backend: lite-frontend-only lite-copy-frontend

# 构建前端
frontend-only:
	@echo "=== 正在编译前端, 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(FRONTEND_DIR)/dist/*
	cd $(FRONTEND_DIR); pnpm run build

# 构建后端（会先校验错误码表与文档是否同步）
backend-only: codes-verify
	@echo "=== 正在编译后端, 当前系统为: $(DETECTED_OS) ==="
	mkdir -p $(OUT_DIR)
	$(CGO) go build -ldflags '-s -w' -v -o $(OUT_DIR)/dmp

# 复制前端产物到 embedFS/dist（不重新构建前端）
copy-frontend:
	@echo "=== 正在复制前端, 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(EMBED_DIR)/*
	cp -r $(FRONTEND_DIR)/dist/* $(EMBED_DIR)/

lite: codes-verify
	@echo "=== 正在编译后端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(CGO) go build -ldflags '-s -w' -v -tags=lite -o ~/dmp-lite

lite-frontend-only:
	@echo "=== 正在编译前端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(FRONTEND_DIR)/dist-lite/*
	cd $(FRONTEND_DIR); pnpm run build:lite

lite-copy-frontend:
	@echo "=== 正在复制前端(lite), 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(LITE_ASSET)/*
	cp -r $(FRONTEND_DIR)/dist-lite/* $(LITE_ASSET)/

# 重新生成业务错误码文档
codes:
	@echo "=== 正在生成业务错误码文档 ==="
	go run ./cmd/codedoc

# 校验错误码表：码唯一、号段正确、词条存在、无死码、文档已同步
codes-verify:
	@echo "=== 正在校验业务错误码 ==="
	go run ./cmd/codedoc -verify

# 提交前的完整检查
check: codes-verify
	@echo "=== 正在检查格式与静态问题 ==="
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './embedFS/*' -not -path './lite/dist-lite/*'))" || \
		(echo "以下文件未通过 gofmt:"; gofmt -l $$(find . -name '*.go' -not -path './embedFS/*' -not -path './lite/dist-lite/*'); exit 1)
	go vet ./...
	go vet -tags=lite ./...

help:
	@echo "=== 正在打印帮助信息, 当前系统为: $(DETECTED_OS) ==="
	@echo "make frontend2backend    构建前端并将产物复制到后端"
	@echo "make backend-only        构建后端"
	@echo "make lite                构建后端(lite)"
	@echo "make all                 构建前端后构建后端"
	@echo "make codes               重新生成业务错误码文档(docs/error-codes.md)"
	@echo "make codes-verify        校验错误码表与文档是否同步"
	@echo "make check               gofmt + go vet(含 lite) + 错误码校验"