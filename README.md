# 基于conntrack获取pod的网络连接情况

- 使用 `go-conntrack` 来获取连接跟踪信息
- 使用 `prometheus` 暴露采集指标

## 开发背景
之前观测不到容器内部的网络连接状态，容器内部经常出现`close-wait`导致容器假死，期望可以增加对于容器内部网络连接状况的可观测性，同时业务迁移上云需要快速梳理出应用连接的外部db，cache， mq等，使用`promql` 更方便查询。

## 参数配置

| 参数名 | 参数说明 | 
| --- | --- |
| podCIDR | pod网段 |
| prometheusAddr | Prometheus 查询地址，需要通过源ip回调查询pod元数据 |
| listenPort | 监听接口 |