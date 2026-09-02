# workbench —  常用构建与运行命令（在项目根目录执行：make build）
BINARY := workbench
# 单目录部署产物：内含二进制、configs
DIST := dist/workbench

build: ## 单目录部署：生成 $(DIST)/；拷贝到服务器后先 cd 到该目录，再执行 ./install.sh（写入配置并导库）或 ./$(BINARY)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)/configs $(DIST)/db
	@cp -a configs/config.yaml $(DIST)/configs/
	@cp -a web db $(DIST)/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY) ./cmd/server
	@echo ""
	@echo "部署目录已生成: $(DIST)"
