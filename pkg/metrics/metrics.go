package metrics

import (
	"log"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"

	"pod_connect_exporter/pkg/container"
	"pod_connect_exporter/pkg/network"
)

// 获取节点名称，默认为"unknown"
func getNodeName() string {
	if name := os.Getenv("NODE_NAME"); name != "" {
		return name
	}
	return "unknown"
}

// ExporterMetrics 定义了导出器的指标
type ExporterMetrics struct {
	connectInfo *prometheus.Desc
}

// NewMetricsCollector 创建一个新的指标收集器
func NewMetricsCollector() *ExporterMetrics {
	return &ExporterMetrics{
		connectInfo: prometheus.NewDesc(
			"pod_connect_info",
			"Status of pod network connections (established, listening, etc.)",
			[]string{"remote_addr", "remote_port", "status", "pod_name", "pod_namespace", "container", "node"}, // 添加node标签
			nil),
	}
}

// Describe 实现了Prometheus收集器接口
func (collector *ExporterMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.connectInfo
}

// Collect 实现了Prometheus收集器接口
func (collector *ExporterMetrics) Collect(ch chan<- prometheus.Metric) {
	collector.collectConnectMetrics(ch)
}

// collectConnectMetrics 收集容器连接指标
func (collector *ExporterMetrics) collectConnectMetrics(ch chan<- prometheus.Metric) {
	containers, err := container.ListContainers()
	if err != nil {
		log.Printf("Error listing containers: %v", err)
		return
	}

	// 获取节点名称
	nodeName := getNodeName()

	for _, cont := range containers {
		connections, err := network.GetTCPConnections(int(cont.PID))
		if err != nil {
			log.Printf("Error getting TCP connections for container %s: %v", cont.Container, err)
			continue
		}
		stats := network.AggregateConnections(connections)

		for _, stat := range stats {
			ch <- prometheus.MustNewConstMetric(
				collector.connectInfo,
				prometheus.GaugeValue,
				float64(stat.Count),
				stat.RemoteIP,
				strconv.Itoa(stat.RemotePort),
				stat.Status,
				cont.PodName,
				cont.Namespace,
				cont.Container,
				nodeName)
		}
	}
}
