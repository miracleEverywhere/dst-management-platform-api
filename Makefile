FRONTEND_DIR := $(HOME)/WebstormProjects/dst-management-platform-web
EMBED_DIR := embedFS/dist
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
        CGO = CGO_ENABLED=1
        OUT_DIR = $(HOME)/dmp
    else
        $(error Unsupported OS: $(UNAME_S))
    endif
endif

.PHONY: all frontend2backend backend-only frontend-only help

# 构建前端 构建后端
all: frontend-only copy-frontend backend-only

# 构建前端并将产物复制到后端
frontend2backend: frontend-only copy-frontend


# 构建前端
frontend-only:
	@echo "=== 正在编译前端, 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(FRONTEND_DIR)/dist/*
	cd $(FRONTEND_DIR); pnpm run build

# 构建后端
backend-only:
	@echo "=== 正在编译后端, 当前系统为: $(DETECTED_OS) ==="
	mkdir -p $(OUT_DIR)
	$(CGO) go build -ldflags '-s -w' -v -o $(OUT_DIR)/dmp

# 复制前端产物到 embedFS/dist（不重新构建前端）
copy-frontend:
	@echo "=== 正在复制前端, 当前系统为: $(DETECTED_OS) ==="
	$(RM) $(EMBED_DIR)/*
	cp -r $(FRONTEND_DIR)/dist/* $(EMBED_DIR)/

help:
	@echo "=== 正在打印帮助信息, 当前系统为: $(DETECTED_OS) ==="
	@echo "make frontend2backend    构建前端并将产物复制到后端"
	@echo "make backend-only        构建后端"
	@echo "make all                 构建前端后构建后端"