FRONTEND_DIR := $(HOME)/WebstormProjects/dst-management-platform-web
EMBED_DIR := embedFS/dist
RM := rm -rf

ifeq ($(OS),Windows_NT)
	SHELL := powershell.exe
	RM = rm -r
	FRONTEND_DIR = $(USERPROFILE)\WebstormProjects\dst-management-platform-web
endif

.PHONY: all frontend backend clean

# 构建前端 构建后端
all: frontend-only copy-frontend backend-only

# 构建前端并将产物复制到后端
frontend2backend: frontend-only copy-frontend

# 构建前端
frontend-only:
	@echo "=== Building frontend ==="
	$(RM) $(FRONTEND_DIR)/dist/*
	cd $(FRONTEND_DIR); pnpm run build


# 构建后端
backend-only:
	@echo "=== Building backend ==="
	CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# 复制前端产物到 embedFS/dist（不重新构建前端）
copy-frontend:
	@echo "=== Copying frontend dist ==="
	$(RM) $(EMBED_DIR)/*
	cp -r $(FRONTEND_DIR)/dist/* $(EMBED_DIR)/

help:
	@echo ""
	@echo "make frontend2backend    构建前端并将产物复制到后端"
	@echo "make backend-only        构建后端"
	@echo "make all                 构建前端后构建后端"
	@echo ""