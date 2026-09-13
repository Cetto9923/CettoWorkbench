# workbench — common build, quality, and run commands (execute at repository root)
BINARY := workbench
DIST := dist/workbench
GOCACHE_DIR := $(CURDIR)/tmp/gocache

.PHONY: build check check-gofmt check-test check-frontend-test check-vet check-whitespace check-file-length check-patterns check-secrets check-architecture check-root-artifacts check-gates

check: check-local-artifacts check-root-artifacts check-gofmt check-test check-frontend-test check-vet check-whitespace check-file-length check-patterns check-secrets check-architecture
	@echo "required regression gates passed; inspect the printed existing-debt counts"

check-gofmt:
	@./scripts/check-gofmt.sh

check-test:
	@mkdir -p $(GOCACHE_DIR)
	@GOCACHE=$(GOCACHE_DIR) go test ./...

check-frontend-test:
	@node tests/unit/frontend/wb-picker-core.regression.test.js
	@node tests/unit/frontend/wb-picker-search.regression.test.js
	@node tests/unit/frontend/wb-picker-phase-a.regression.test.js
	@node tests/unit/frontend/agileteam-adjustment.regression.test.js
	@node tests/unit/frontend/user-picker.test.js
	@node tests/unit/frontend/auth-errors.test.js
	@node tests/unit/frontend/csrf-tokens.test.js
	@node tests/unit/frontend/personal-list.test.js
	@node tests/unit/frontend/demand-detail.test.js
	@node tests/unit/frontend/demand-detail-review.test.js
	@node tests/unit/frontend/home-focus.test.js
	@node tests/unit/frontend/home-list-caption.test.js
	@node tests/unit/frontend/notice-filters.test.js
	@node tests/unit/frontend/priority-helpers.test.js
	@node tests/unit/frontend/navigation-primary-action.test.js
	@node tests/unit/frontend/primary-action-availability.test.js
	@node tests/unit/frontend/html-sanitize.test.js
	@node tests/unit/frontend/render-row-isolated.test.js
	@node tests/unit/frontend/done-status-labels.test.js
	@node tests/unit/frontend/unified-object-id-ui.test.js
	@node tests/unit/frontend/todos-object-chips.test.js
	@node tests/unit/frontend/urge-modal.test.js
	@node tests/unit/frontend/ui-consistency.test.js
	@echo "frontend unit/behavior tests passed"

check-vet:
	@./scripts/check-go-vet.sh

check-whitespace:
	@git diff --check
	@git diff --cached --check
	@if git rev-parse --verify HEAD^ >/dev/null 2>&1; then git diff --check HEAD^ HEAD; else git show --check --format= HEAD; fi
	@echo "whitespace gate passed"

check-file-length:
	@./scripts/check-file-length.sh

check-patterns:
	@./scripts/check-patterns.sh

check-secrets:
	@./scripts/check-secrets.sh

check-architecture:
	@./scripts/check-architecture.sh

check-root-artifacts:
	@bash scripts/check-root-artifacts.sh

check-gates:
	@bash scripts/test-quality-gates.sh

build: ## Build a self-contained deployment directory under $(DIST)
	@rm -rf $(DIST)
	@mkdir -p $(DIST)/configs $(DIST)/db
	@cp -a configs/config.yaml $(DIST)/configs/
	@cp -a web db $(DIST)/
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY) ./cmd/server
	@echo ""
	@echo "deployment directory generated: $(DIST)"

.PHONY: check-local-artifacts
check-local-artifacts:
	@bash scripts/check-local-artifacts.sh
