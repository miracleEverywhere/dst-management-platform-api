FRONTEND_DIR := $(HOME)/WebstormProjects/dst-management-platform-web
EMBED_DIR := embedFS/dist

.PHONY: all frontend backend clean

all: frontend-only copy-frontend backend-only

frontend2backend: frontend-only copy-frontend

frontend-only:
	@echo "=== Building frontend ==="
	cd $(FRONTEND_DIR) && npx vite build

clean-embed:
	@echo "=== Cleaning embedFS/dist ==="
	rm -rf $(EMBED_DIR)/*

backend-only:
	@echo "=== Building backend ==="
	CGO_ENABLED=0 go build -ldflags '-s -w' -v -o dmp

# 复制前端产物到 embedFS/dist（不重新构建前端）
copy-frontend:
	@echo "=== Copying frontend dist ==="
	rm -rf $(EMBED_DIR)/*
	cp -r $(FRONTEND_DIR)/dist/* $(EMBED_DIR)/
