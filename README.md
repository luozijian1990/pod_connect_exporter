# Pod Connect Exporter

## 项目概述

Pod Connect Exporter 是一个基于 Golang 开发的工具，用于监控并导出 Kubernetes Pod 的网络连接状态。它通过读取容器的网络连接信息，并以 Prometheus 指标的形式导出，便于监控和分析容器的网络连接情况。

## 开发背景

在容器化环境中，经常出现因网络连接问题（如 `CLOSE_WAIT` 状态堆积）导致的容器假死现象，而传统的监控手段难以观测容器内部的网络连接状态。同时，在业务迁移上云过程中，需要快速梳理应用连接的外部资源（如数据库、缓存、消息队列等）。本工具旨在解决这些问题，提供容器网络连接的可观测性，并通过 Prometheus 查询语言（PromQL）便于分析。

## 主要功能

- **实时监控**：监控 Kubernetes Pod 的 TCP 连接状态
- **指标导出**：以 Prometheus 指标格式导出连接信息
- **连接分析**：提供连接状态、远程地址、端口等详细信息
- **健康检查**：内置健康检查接口

## 容器运行时支持

当前版本仅支持 **containerd** 容器运行时。在未来的版本中，我们计划添加以下功能：

- **Docker 支持**：添加对 Docker 容器运行时的支持
- **自动检测**：自动检测并使用可用的容器运行时
- **CRI-O 支持**：考虑添加对 CRI-O 容器运行时的支持

如果您需要在使用 Docker 作为容器运行时的环境中部署，请关注项目更新或考虑贡献代码。

## 项目结构

```
.
├── cmd/
│   └── exporter/       # 主程序入口
│       └── main.go     # 主程序代码
├── pkg/
│   ├── container/      # 容器相关功能
│   ├── network/        # 网络连接相关功能
│   └── metrics/        # Prometheus 指标相关功能
├── build.sh            # 构建脚本
├── go.mod              # Go 模块定义
└── README.md           # 项目说明文档
```

## 安装与使用

### 前提条件

- Go 1.21 或更高版本
- 访问 Kubernetes 集群的权限（如果在集群内部署）
- containerd 容器运行时（当前版本仅支持 containerd）
- 运行环境能够访问 containerd socket（默认位置：/run/containerd/containerd.sock）

### 安装步骤

1. 克隆仓库
   ```bash
   git clone https://github.com/luozijian1990/pod_connect_exporter.git
   cd pod_connect_exporter
   ```

2. 构建项目
   ```bash
   # 使用构建脚本
   chmod +x build.sh
   ./build.sh
   
   # 或直接使用 Go 命令
   go build -o pod_connect_exporter ./cmd/exporter
   ```

3. 运行服务
   ```bash
   ./pod_connect_exporter -port=28880
   ```

### 配置选项

| 参数      | 说明                    | 默认值  |
|-----------|------------------------|---------|
| -port     | 服务监听端口            | 28880   |

## API 接口

- `/metrics`: Prometheus 指标接口，返回所有监控的容器连接信息
- `/healthz`: 健康检查接口

## 指标说明

主要指标：`pod_connect_info`

标签：
- `remote_addr`: 远程地址
- `remote_port`: 远程端口
- `status`: 连接状态（如 ESTABLISHED, CLOSE_WAIT 等）
- `pod_name`: Pod 名称
- `pod_namespace`: Pod 所在命名空间
- `container`: 容器名称
- `node`: 节点名称

## 部署示例

在 Kubernetes 中部署本工具，我们使用了 DaemonSet 确保每个节点都运行一个实例，并且挂载了必要的卷：

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: pod-connect-exporter
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: pod-connect-exporter
  template:
    metadata:
      labels:
        app: pod-connect-exporter
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "28880"
    spec:
      hostPID: true
      containers:
      - name: exporter
        image: your-registry/pod-connect-exporter:latest
        securityContext:
          privileged: true
          runAsUser: 0
        ports:
        - containerPort: 28880
          name: metrics
        volumeMounts:
        - name: proc
          mountPath: /host/proc
          readOnly: true
        - name: containerd-sock
          mountPath: /run/containerd/containerd.sock
        env:
        - name: PROC_PATH
          value: /host/proc
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
      volumes:
      - name: proc
        hostPath:
          path: /proc
      - name: containerd-sock
        hostPath:
          path: /run/containerd/containerd.sock
```

完整的部署配置可以在 `deploy/kubernetes/daemonset.yaml` 中找到。

### 关于特权和挂载的说明

本工具需要以下特殊权限和挂载：

1. **hostPID: true** - 允许访问主机PID命名空间
2. **privileged: true** - 获取所需的特权以读取容器进程信息
3. **/proc挂载** - 通过只读方式挂载主机的/proc到容器的/host/proc
4. **containerd.sock挂载** - 用于访问容器运行时API（目前仅支持containerd）

这些权限设置类似于node-exporter等系统监控工具的配置。

### 使用 Docker 作为容器运行时

当前版本仅支持 containerd 容器运行时。如果您的 Kubernetes 集群使用 Docker 作为容器运行时，您需要等待后续版本更新。我们计划在未来版本中添加对 Docker 的支持，包括：

1. 自动检测使用的容器运行时
2. 支持 Docker socket 挂载（/var/run/docker.sock）
3. 适配 Docker API 获取容器信息

如果您急需此功能，欢迎通过 Pull Request 贡献代码。

## 致谢

- 该项目使用了 [containerd](https://github.com/containerd/containerd) 库获取容器信息
- 该项目使用了 [client_golang](https://github.com/prometheus/client_golang) 实现 Prometheus 指标导出

## 许可证

本项目采用 MIT 许可证