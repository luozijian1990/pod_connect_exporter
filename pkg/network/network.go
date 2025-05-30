package network

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 默认/proc路径
var procPath = "/proc"

// SetProcPath 设置自定义的proc路径
func SetProcPath(path string) {
	if path != "" {
		procPath = path
	}
}

// TCPConnection 表示一个TCP连接
type TCPConnection struct {
	Protocol   string
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
	Status     string
}

// ConnectionStat 表示连接统计信息
type ConnectionStat struct {
	RemoteIP   string
	RemotePort int
	Status     string
	Count      int
}

// ConnectionKey 用于聚合连接的键
type ConnectionKey struct {
	RemoteIP   string
	RemotePort int
	Status     string
}

// HexToIP 将十六进制格式的IP地址转换为点分十进制格式
func HexToIP(hexIP string) string {
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

// HexToPort 将十六进制字符串转为端口号
func HexToPort(hexPort string) int {
	port, _ := strconv.ParseInt(hexPort, 16, 32)
	return int(port)
}

// GetTCPStatus 解析TCP连接状态
func GetTCPStatus(statusHex string) string {
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

// GetTCPConnections 获取某个进程的TCP连接信息
func GetTCPConnections(pid int) ([]TCPConnection, error) {
	var connections []TCPConnection

	// 使用自定义的proc路径构建TCP文件路径
	tcpFilePath := filepath.Join(procPath, strconv.Itoa(pid), "net", "tcp")

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
		localIP := HexToIP(localAddrParts[0])
		localPort := HexToPort(localAddrParts[1])

		// 解析远程地址和端口
		remoteAddrParts := strings.Split(fields[2], ":")
		if len(remoteAddrParts) != 2 {
			continue
		}
		remoteIP := HexToIP(remoteAddrParts[0])
		remotePort := HexToPort(remoteAddrParts[1])

		// 跳过本地到本地的连接
		if localIP == remoteIP && localIP == "127.0.0.1" {
			continue
		}

		// 解析状态
		status := GetTCPStatus(fields[3])
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

// AggregateConnections 聚合TCP连接信息
func AggregateConnections(connections []TCPConnection) []ConnectionStat {
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
