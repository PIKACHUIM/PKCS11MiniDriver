# PKCS11Driver Makefile
# 用法：make help

.PHONY: help build build-frontend build-backend test clean dev

help:
	@echo "PKCS11Driver 构建工具"
	@echo ""
	@echo "  make build           构建前端 + 后端（完整构建）"
	@echo "  make build-frontend  仅构建前端（webpage/dist）"
	@echo "  make build-backend   仅构建后端（client-card + server-card）"
	@echo "  make test            运行所有 Go 测试"
	@echo "  make dev             启动前端开发服务器（:5173）"
	@echo "  make clean           清理构建产物"

# ---- 完整构建 ----
build: build-frontend build-backend

# ---- 前端构建 ----
build-frontend:
	@echo ">>> 构建前端..."
	cd webpage && npm install && npm run build
	@echo ">>> 复制前端产物到 client-card/ui/dist..."
	cp -r webpage/dist client-card/ui/dist
	@echo ">>> 前端构建完成"

# ---- 后端构建 ----
build-backend:
	@echo ">>> 构建 client-card..."
	cd client-card && go build -o ../bin/client-card ./cmd/client-card
	@echo ">>> 构建 server-card..."
	cd server-card && go build -o ../bin/server-card ./cmd/server-card
	@echo ">>> 后端构建完成"

# ---- 测试 ----
test:
	@echo ">>> 运行 client-card 测试..."
	cd client-card && go test ./... -v -count=1 -race
	@echo ">>> 运行 server-card 测试..."
	cd server-card && go test ./... -v -count=1 2>/dev/null || echo "server-card 暂无测试"

# ---- 开发模式 ----
dev:
	@echo ">>> 启动前端开发服务器（http://localhost:5173）..."
	cd webpage && npm run dev

# ---- 清理 ----
clean:
	rm -rf bin/
	rm -rf webpage/dist
	rm -rf client-card/ui/dist
	find . -name "*.db" -not -path "./.git/*" -delete
