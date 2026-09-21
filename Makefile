GO ?= go
LOCAL_BIN ?= $(HOME)/.local/bin
GOLANGCI_LINT ?= $(LOCAL_BIN)/golangci-lint
GOLANGCI_LINT_MODULE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
GOLANGCI_LINT_TIMEOUT ?= 15m
GOSEC ?= $(LOCAL_BIN)/gosec
GOSEC_EXCLUDE_DIRS ?=
GOTOOLCHAIN ?= go1.25.13
AUDIT_GO = env GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO)
STATICCHECK ?= $(LOCAL_BIN)/staticcheck
STATICCHECK_MODULE ?= honnef.co/go/tools/cmd/staticcheck@v0.8.1
GOVULNCHECK ?= $(LOCAL_BIN)/govulncheck
GOVULNCHECK_MODULE ?= golang.org/x/vuln/cmd/govulncheck@latest
SEMGREP ?= $(or $(shell command -v semgrep 2>/dev/null),semgrep)
TRIVY ?= $(or $(shell command -v trivy 2>/dev/null),$(LOCAL_BIN)/trivy)
AUDIT_DIR ?=
AUDIT_SKIP_DIRS ?= .git
RACE_PKGS ?= './...'

.PHONY: audit-init audit-quick audit-gosec audit-staticcheck audit-vet audit-race
.PHONY: audit-govulncheck audit-semgrep audit-trivy-vuln audit-trivy-fs audit-trivy
.PHONY: audit-full audit-report

##@ 代码质量

#   make audit-quick
#   make audit-report
#   make audit-full AUDIT_DIR=tmp/audit-$(shell date +%Y%m%d)

audit-init: ## 安装 Go 系审计工具到 LOCAL_BIN（须与 go.mod 工具链一致）
	mkdir -p "$(LOCAL_BIN)"
	GOBIN="$(LOCAL_BIN)" env GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) install $(GOLANGCI_LINT_MODULE)
	GOBIN="$(LOCAL_BIN)" env GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) install $(STATICCHECK_MODULE)
	GOBIN="$(LOCAL_BIN)" env GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) install github.com/securego/gosec/v2/cmd/gosec@v2.22.2
	GOBIN="$(LOCAL_BIN)" env GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) install $(GOVULNCHECK_MODULE)
	@echo "audit-init: 已安装到 $(LOCAL_BIN)（GOTOOLCHAIN=$(GOTOOLCHAIN)）"
	@echo "audit-init: 可选 — pip install semgrep；Trivy / osv-scanner 见各工具官方文档"

audit-gosec: ## 独立 gosec（未安装时回退 golangci-lint gosec）
	@if [ -x "$(GOSEC)" ]; then \
		"$(GOSEC)" -quiet -tests -conf .gosec.json $(GOSEC_EXCLUDE_DIRS) ./...; \
	elif [ -x "$(GOLANGCI_LINT)" ]; then \
		echo "gosec: 未找到 $(GOSEC)，使用 $(GOLANGCI_LINT) 内置 gosec"; \
		"$(GOLANGCI_LINT)" run ./... --enable-only=gosec --timeout=$(GOLANGCI_LINT_TIMEOUT); \
	else \
		echo "gosec: 请先执行 make audit-init"; exit 1; \
	fi

audit-staticcheck: ## staticcheck（须用与 go.mod 一致的工具链编译）
	@if [ ! -x "$(STATICCHECK)" ]; then echo "staticcheck: 未找到 $(STATICCHECK)，请先 make audit-init"; exit 1; fi
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; "$(STATICCHECK)" ./... | tee "$(AUDIT_DIR)/staticcheck.txt"'; \
	else \
		"$(STATICCHECK)" ./...; \
	fi

audit-vet: ## go vet（GOTOOLCHAIN 与 go.mod 对齐）
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; $(AUDIT_GO) vet ./... 2>&1 | tee "$(AUDIT_DIR)/go-vet.txt"'; \
	else \
		$(AUDIT_GO) vet ./...; \
	fi

audit-quick: ## 快速静态审计（golangci-lint + go vet + staticcheck + gosec）
	@if [ ! -x "$(GOLANGCI_LINT)" ]; then echo "golangci-lint: 未找到 $(GOLANGCI_LINT)，请先 make audit-init（GOTOOLCHAIN=$(GOTOOLCHAIN)）"; exit 1; fi
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; "$(GOLANGCI_LINT)" run ./... --timeout=$(GOLANGCI_LINT_TIMEOUT) | tee "$(AUDIT_DIR)/golangci-lint.txt"'; \
	else \
		"$(GOLANGCI_LINT)" run ./... --timeout=$(GOLANGCI_LINT_TIMEOUT); \
	fi
	@$(MAKE) audit-vet AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-staticcheck AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-gosec AUDIT_DIR="$(AUDIT_DIR)"

audit-race: ## 竞态检测
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; $(AUDIT_GO) test -race $(RACE_PKGS) -count=1 | tee "$(AUDIT_DIR)/race.txt"'; \
	else \
		$(AUDIT_GO) test -race $(RACE_PKGS) -count=1; \
	fi

audit-govulncheck: ## Go 官方依赖漏洞扫描
	@if [ ! -x "$(GOVULNCHECK)" ]; then echo "govulncheck: 未找到 $(GOVULNCHECK)，请先 make audit-init"; exit 1; fi
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; env GOTOOLCHAIN=$(GOTOOLCHAIN) "$(GOVULNCHECK)" ./... | tee "$(AUDIT_DIR)/govulncheck.txt"'; \
	else \
		env GOTOOLCHAIN=$(GOTOOLCHAIN) "$(GOVULNCHECK)" ./...; \
	fi

audit-semgrep: ## Semgrep（p/golang + p/secrets）
	@command -v "$(SEMGREP)" >/dev/null 2>&1 || { echo "semgrep 未安装: pip install semgrep"; exit 1; }
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; "$(SEMGREP)" scan --config p/golang --config p/secrets --error | tee "$(AUDIT_DIR)/semgrep.txt"'; \
	else \
		"$(SEMGREP)" scan --config p/golang --config p/secrets --error; \
	fi

audit-trivy-vuln: ## Trivy 依赖 CVE
	@command -v "$(TRIVY)" >/dev/null 2>&1 || { echo "trivy 未安装，见 Aqua Trivy 官方文档"; exit 1; }
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; "$(TRIVY)" fs --scanners vuln --skip-dirs $(AUDIT_SKIP_DIRS) . | tee "$(AUDIT_DIR)/trivy-vuln.txt"'; \
	else \
		"$(TRIVY)" fs --scanners vuln --skip-dirs $(AUDIT_SKIP_DIRS) .; \
	fi

audit-trivy-fs: ## Trivy 全量（vuln + secret + misconfig）
	@command -v "$(TRIVY)" >/dev/null 2>&1 || { echo "trivy 未安装，见 Aqua Trivy 官方文档"; exit 1; }
	@if [ -n "$(AUDIT_DIR)" ]; then \
		mkdir -p "$(AUDIT_DIR)"; \
		bash -c 'set -o pipefail; "$(TRIVY)" fs --scanners vuln,secret,misconfig --skip-dirs $(AUDIT_SKIP_DIRS) . | tee "$(AUDIT_DIR)/trivy.txt"'; \
	else \
		"$(TRIVY)" fs --scanners vuln,secret,misconfig --skip-dirs $(AUDIT_SKIP_DIRS) .; \
	fi

audit-trivy: ## Trivy 依赖 CVE
	@$(MAKE) audit-trivy-vuln AUDIT_DIR="$(AUDIT_DIR)"

audit-full: ## 全量审计（静态 + race + govulncheck + semgrep + trivy）
	@$(MAKE) audit-quick AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-race AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-govulncheck AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-semgrep AUDIT_DIR="$(AUDIT_DIR)"
	@$(MAKE) audit-trivy AUDIT_DIR="$(AUDIT_DIR)"

audit-report: ## 全量审计并写入 tmp/audit-YYYYMMDD
	@$(MAKE) audit-full AUDIT_DIR="$(or $(AUDIT_DIR),tmp/audit-$(shell date +%Y%m%d))"
