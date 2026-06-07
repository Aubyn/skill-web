# skill-web — 单命令全量构建
# make         构建前端 + 后端（默认）
# make clean   清理构建产物

GO       ?= go
PNPM     ?= pnpm
BINARY    = skill-web
FRONTEND  = frontend

.PHONY: build clean frontend

build: frontend
	$(GO) build -o $(BINARY) .

frontend:
	cd $(FRONTEND) && $(PNPM) install && $(PNPM) build

clean:
	rm -f $(BINARY)
	rm -rf $(FRONTEND)/dist
