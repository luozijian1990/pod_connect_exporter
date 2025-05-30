package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"pod_connect_exporter/pkg/metrics"
	"pod_connect_exporter/pkg/network"
)

var (
	listenPort string
	version    = "dev"
	buildTime  = "unknown"
	nodeName   = "unknown"
)

// 健康检查接口
func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("health"))
}

func main() {
	// 解析命令行参数
	flag.StringVar(&listenPort, "port", "28880", "exporter listen port")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Printf("Pod Connect Exporter version %s, build time: %s\n", version, buildTime)
		return
	}

	// 设置proc路径（如果环境变量存在）
	if procPath := os.Getenv("PROC_PATH"); procPath != "" {
		log.Printf("Using custom proc path: %s", procPath)
		network.SetProcPath(procPath)
	}

	// 获取节点名称
	if name := os.Getenv("NODE_NAME"); name != "" {
		nodeName = name
		log.Printf("Running on node: %s", nodeName)
	}

	// 初始化指标收集器
	allMetrics := metrics.NewMetricsCollector()
	prometheus.MustRegister(allMetrics)

	// 设置HTTP处理器
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", health)

	server := &http.Server{Addr: fmt.Sprintf(":%s", listenPort)}

	log.Printf("Starting Pod Connect Exporter v%s on port %s\n", version, listenPort)

	// 在goroutine中启动服务器
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("ListenAndServe(): %s\n\n", err)
			panic(err)
		}
	}()

	// 处理信号以优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt, syscall.SIGKILL)

	// 阻塞,并等待signal信号
	sig := <-sigChan
	log.Println(sig)
	log.Println("SIGTERM received, shutting down gracefully...")

	// 创建一个带超时的上下文用于关闭
	timeout, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 关闭 http 服务
	if err := server.Shutdown(timeout); err != nil {
		log.Printf("Server Close Error: %s\n\n", err)
	} else {
		log.Println("Server Close Successful")
	}
}
