# workbench —  常用构建与运行命令（在项目根目录执行：make build）
BINARY := workbench
# 单目录部署产物：内含二进制、configs
DIST := dist/workbench

.PHONY: build check lint lint-web quality

check: ## 运行 Go 单元测试与静态检查
	go test ./...
	go vet ./...

lint: ## 检查相对质量门基线新增或改动的 Go 代码
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version | grep -q 'version: 2\.'; then \
		golangci-lint run --config .golangci.yml; \
	else \
		go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0 run --config .golangci.yml; \
	fi

lint-web: ## 检查新增 CSS；基线已有文件见 .stylelintignore
	npx --yes stylelint@17.14.1 "web/**/*.css" --ignore-path .stylelintignore --allow-empty-input
	./tools/check-file-size.sh

quality: ## 运行 Go 与前端质量门，并报告两侧全部发现
	@status=0; \
	$(MAKE) --no-print-directory lint || status=1; \
	$(MAKE) --no-print-directory lint-web || status=1; \
	exit $$status

build: ## 单目录部署：生成 $(DIST)/；拷贝到服务器后先 cd 到该目录，再执行 ./install.sh（写入配置并导库）或 ./$(BINARY)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)/configs $(DIST)/db
	@cp -a configs/config.yaml $(DIST)/configs/
	@cp -a web db $(DIST)/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY) ./cmd/server
	@echo ""
	@echo "部署目录已生成: $(DIST)"
