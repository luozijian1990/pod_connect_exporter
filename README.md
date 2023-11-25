# Prometheus Pod connect exporter

## 概览

Prometheus Pod connect exporter是一个基于 Golang 开发的工具，旨在监控并导出 Kubernetes Pod 的网络连接状态。它利用 Prometheus 框架，并与 conntrack 工具集成，提供实时的 Pod 连接指标。

## 开发背景
之前观测不到容器内部的网络连接状态，容器内部经常出现`close-wait`导致容器假死，期望可以增加对于容器内部网络连接状况的可观测性，同时业务迁移上云需要快速梳理出应用连接的外部db，cache， mq等，使用`promql` 更方便查询。

## 特性

- 实时跟踪 Pod 网络连接。
- 与 Prometheus 集成，用于监控和报警。
- 提供详细的指标，包括源和目标 IP、端口以及 TCP 连接状态。
- 可定制设置，用于针对特定的 Pod CIDR 和 Prometheus 实例。

## 开始使用

### 前提条件

- 一个正常运行的 Kubernetes 集群。
- 集群内已设置 Prometheus 以进行指标抓取。

### 安装

```bash
# 克隆仓库
git clone https://github.com/your-github-username/pod-connection-exporter.git

# 进入项目目录
cd pod-connection-exporter

# 构建项目（确保已安装 Golang）
go build
```

##  配置
- -pod-cidr：指定要监控的 Pod CIDR（默认为 "10.100.0.0/16"）。
- -prometheus-addr：Prometheus 服务器的地址（默认为 "10.101.20.9:9090"）。
- -port：导出器监听的端口（默认为 "28880"）。

##运行
```bash
./pod-connection-exporter -pod-cidr="your-pod-cidr" --prometheus-addr="your-prometheus-addr" --port="your-listen-port"
```

## Thanks
- 该项目使用了[go-conntrack](https://github.com/florianl/go-conntrack) 
- 该项目使用了[client_golang](https://github.com/prometheus/client_golang)