# 变量定义
BINARY_NAME=pod_connect_exporter
VERSION=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
DOCKER_REPO=your-registry
DOCKER_TAG=${VERSION}

# 颜色输出
COLOR_RESET=\033[0m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m

# 默认目标
.PHONY: all
all: build

# 构建二进制文件
.PHONY: build
build:
	@echo "${COLOR_GREEN}Building ${BINARY_NAME}...${COLOR_RESET}"
	@go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/exporter
	@echo "${COLOR_GREEN}Build complete!${COLOR_RESET}"

# 运行测试
.PHONY: test
test:
	@echo "${COLOR_GREEN}Running tests...${COLOR_RESET}"
	@go test -v ./...

# 清理构建产物
.PHONY: clean
clean:
	@echo "${COLOR_GREEN}Cleaning up...${COLOR_RESET}"
	@rm -f ${BINARY_NAME}
	@go clean

# 运行程序
.PHONY: run
run: build
	@echo "${COLOR_GREEN}Running ${BINARY_NAME}...${COLOR_RESET}"
	@./${BINARY_NAME}

# 构建 Docker 镜像
.PHONY: docker-build
docker-build:
	@echo "${COLOR_GREEN}Building Docker image...${COLOR_RESET}"
	@docker build -t ${DOCKER_REPO}/${BINARY_NAME}:${DOCKER_TAG} .

# 推送 Docker 镜像
.PHONY: docker-push
docker-push: docker-build
	@echo "${COLOR_GREEN}Pushing Docker image...${COLOR_RESET}"
	@docker push ${DOCKER_REPO}/${BINARY_NAME}:${DOCKER_TAG}

# 生成 Kubernetes 部署文件
.PHONY: k8s-deploy
k8s-deploy:
	@echo "${COLOR_GREEN}Generating Kubernetes deployment file...${COLOR_RESET}"
	@sed "s|your-registry/pod-connect-exporter:latest|${DOCKER_REPO}/${BINARY_NAME}:${DOCKER_TAG}|g" deploy/kubernetes/daemonset.yaml > deploy/kubernetes/daemonset-${VERSION}.yaml
	@echo "${COLOR_GREEN}Generated at deploy/kubernetes/daemonset-${VERSION}.yaml${COLOR_RESET}"

# 显示帮助信息
.PHONY: help
help:
	@echo "${COLOR_YELLOW}Available targets:${COLOR_RESET}"
	@echo "  ${COLOR_GREEN}all${COLOR_RESET}          - Default target, builds the binary"
	@echo "  ${COLOR_GREEN}build${COLOR_RESET}        - Build the binary"
	@echo "  ${COLOR_GREEN}test${COLOR_RESET}         - Run tests"
	@echo "  ${COLOR_GREEN}clean${COLOR_RESET}        - Clean build artifacts"
	@echo "  ${COLOR_GREEN}run${COLOR_RESET}          - Build and run the binary"
	@echo "  ${COLOR_GREEN}docker-build${COLOR_RESET} - Build Docker image"
	@echo "  ${COLOR_GREEN}docker-push${COLOR_RESET}  - Build and push Docker image"
	@echo "  ${COLOR_GREEN}k8s-deploy${COLOR_RESET}   - Generate Kubernetes deployment file"
	@echo "  ${COLOR_GREEN}help${COLOR_RESET}         - Show this help" 