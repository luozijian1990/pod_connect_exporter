package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"

	"github.com/prometheus/client_golang/prometheus"
)

var listenPort string

type ContainerInfo struct {
	PID       uint32
	PodName   string
	Namespace string
	Container string
}

type TCPConnection struct {
	Protocol   string
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
	Status     string
}

type ConnectionStat struct {
	RemoteIP   string
	RemotePort int
	Status     string
	Count      int
}

type ConnectionKey struct {
	RemoteIP   string
	RemotePort int
	Status     string
}

type ExporterMetrics struct {
	connectInfo *prometheus.Desc
}

func (collector *ExporterMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.connectInfo
}

func (collector *ExporterMetrics) Collect(ch chan<- prometheus.Metric) {
	collector.collectConnectMetrics(ch)
}

func (collector *ExporterMetrics) collectConnectMetrics(ch chan<- prometheus.Metric) {
	contains := listContains()

	for _, container := range contains {
		connections, err := getTCPConnections(int(container.PID))
		if err != nil {
			continue
		}
		stats := aggregateConnections(connections)

		for _, stat := range stats {

			ch <- prometheus.MustNewConstMetric(
				collector.connectInfo,
				prometheus.GaugeValue,
				float64(stat.Count),
				stat.RemoteIP,
				strconv.Itoa(stat.RemotePort),
				stat.Status,
				container.PodName,
				container.Namespace,
				container.Container)
		}
	}
}

func connDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		"pod_connect_info",
		"Status of pod network connections (established, listening, etc.)",
		[]string{"remote_addr", "remote_port", "status", "pod_name", "pod_namespace", "container"}, // 标签
		nil)
}

func newMetricsCollector() *ExporterMetrics {
	return &ExporterMetrics{
		connectInfo: connDesc(),
	}
}

// 健康检查接口
func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("health"))
}

func main() {

	flag.StringVar(&listenPort, "port", "28880", "exporter listen port")
	flag.Parse()

	allMetrics := newMetricsCollector()
	prometheus.MustRegister(allMetrics)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", health)

	server := &http.Server{Addr: fmt.Sprintf(":%s", listenPort)}

	log.Printf("Starting server on port %s\n", listenPort)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {

			log.Printf("ListenAndServe(): %s\n\n", err)
			panic(err)

		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt, syscall.SIGKILL)

	// 阻塞,并等待signal信号
	sig := <-sigChan
	log.Println(sig)
	log.Println("SIGTERM received, shutting down gracefully...")

	timeout, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	defer cancel()
	// 关闭 http 服务
	if err := server.Shutdown(timeout); err != nil {
		log.Printf("Server Close Error: %s\n\n", err)
	} else {
		log.Println("server Close Successful")
	}

}

func listContains() []ContainerInfo {
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 设置上下文中的命名空间，默认是 "default"
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

	var containersInfos []ContainerInfo

	// 列出所有容器
	containers, err := client.Containers(ctx)
	if err != nil {
		log.Fatal(err)
	}

	//打印全部容器pid与labels
	for _, container := range containers {
		task, err := container.Task(ctx, nil)
		if err != nil {
			continue
		}

		labels, err := container.Labels(ctx)
		if err != nil {
			fmt.Printf("  Error getting labels: %v\n", err)
			continue
		}

		s, b := labels["io.cri-containerd.kind"]

		if b && s == "container" {
			containersInfos = append(containersInfos, ContainerInfo{
				PID:       task.Pid(),
				PodName:   labels["io.kubernetes.pod.name"],
				Namespace: labels["io.kubernetes.pod.namespace"],
				Container: labels["io.kubernetes.container.name"],
			})
		}
	}

	return containersInfos
}

func hexToIP(hexIP string) string {
	// 处理小端序的IPv4地址
	if len(hexIP) == 8 {
		var ipParts [4]int64
		for i := 0; i < 4; i++ {
			// 每两个字符一组，并反转顺序
			pos := 6 - 2*i
			ipPart, _ := strconv.ParseInt(hexIP[pos:pos+2], 16, 64)
			ipParts[i] = ipPart
		}
		return fmt.Sprintf("%d.%d.%d.%d", ipParts[0], ipParts[1], ipParts[2], ipParts[3])
	}
	return "unknown"
}

// 十六进制字符串转为端口号
func hexToPort(hexPort string) int {
	port, _ := strconv.ParseInt(hexPort, 16, 32)
	return int(port)
}

// 解析TCP连接状态
func getTCPStatus(statusHex string) string {
	statusMap := map[string]string{
		"01": "ESTABLISHED",
		"02": "SYN_SENT",
		"03": "SYN_RECV",
		"04": "FIN_WAIT1",
		"05": "FIN_WAIT2",
		"06": "TIME_WAIT",
		"07": "CLOSE",
		"08": "CLOSE_WAIT",
		"09": "LAST_ACK",
		"0A": "LISTEN",
		"0B": "CLOSING",
	}

	if status, ok := statusMap[statusHex]; ok {
		return status
	}
	return "UNKNOWN"
}

// 获取某个进程的TCP连接信息
func getTCPConnections(pid int) ([]TCPConnection, error) {
	var connections []TCPConnection

	// 构建TCP文件路径
	tcpFilePath := filepath.Join("/proc", strconv.Itoa(pid), "net", "tcp")

	// 打开文件
	file, err := os.Open(tcpFilePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件 %s: %v", tcpFilePath, err)
	}
	defer file.Close()

	// 读取并解析文件内容
	scanner := bufio.NewScanner(file)
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过标题行
		if lineCount == 0 {
			lineCount++
			continue
		}

		// 分割行内容
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// 解析本地地址和端口
		localAddrParts := strings.Split(fields[1], ":")
		if len(localAddrParts) != 2 {
			continue
		}
		localIP := hexToIP(localAddrParts[0])
		localPort := hexToPort(localAddrParts[1])

		// 解析远程地址和端口
		remoteAddrParts := strings.Split(fields[2], ":")
		if len(remoteAddrParts) != 2 {
			continue
		}
		remoteIP := hexToIP(remoteAddrParts[0])
		remotePort := hexToPort(remoteAddrParts[1])

		// 跳过本地到本地的连接
		if localIP == remoteIP && localIP == "127.0.0.1" {
			continue
		}

		// 解析状态
		status := getTCPStatus(fields[3])
		// 创建连接信息
		if status == "LISTEN" {
			remotePort = localPort
		}

		conn := TCPConnection{
			Protocol:   "TCP",
			LocalIP:    localIP,
			LocalPort:  localPort,
			RemoteIP:   remoteIP,
			RemotePort: remotePort,
			Status:     status,
		}

		connections = append(connections, conn)
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取文件 %s 时出错: %v", tcpFilePath, err)
	}

	return connections, nil
}

// 聚合TCP连接信息
func aggregateConnections(connections []TCPConnection) []ConnectionStat {
	// 用map统计连接数
	statsMap := make(map[ConnectionKey]int)

	for _, conn := range connections {
		key := ConnectionKey{
			RemoteIP:   conn.RemoteIP,
			RemotePort: conn.RemotePort,
			Status:     conn.Status,
		}
		statsMap[key]++
	}

	// 转换为切片以便排序
	var stats []ConnectionStat
	for key, count := range statsMap {
		stats = append(stats, ConnectionStat{
			RemoteIP:   key.RemoteIP,
			RemotePort: key.RemotePort,
			Status:     key.Status,
			Count:      count,
		})
	}

	// 按计数降序排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats
}
