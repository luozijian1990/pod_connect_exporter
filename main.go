package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	ct "github.com/florianl/go-conntrack"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type PodInfo struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Metric struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
}

type Result struct {
	Metric Metric        `json:"metric"`
	Value  []interface{} `json:"value"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type ExporterMetrics struct {
	podConnectStatus *prometheus.Desc
}

var (
	tcpStats = map[uint8]string{
		1: "SYN_SENT",
		3: "ESTABLISHED",
		4: "FIN_WAIT",
		5: "CLOSE_WAIT",
		6: "LAST_ACK",
		7: "TIME_WAIT",
		8: "CLOSE",
	}
	podCIDR        string
	prometheusAddr string
	listenPort     string
)

func (collector *ExporterMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.podConnectStatus
}

func (collector *ExporterMetrics) Collect(ch chan<- prometheus.Metric) {
	collector.getPodConnects(ch)
}

func (collector *ExporterMetrics) getPodConnects(ch chan<- prometheus.Metric) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		log.Println("Could not create nfct:", err)
		return
	}
	defer nfct.Close()
	sessions, err := nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		log.Println("Could not dump sessions:", err)
		return
	}

	var wg sync.WaitGroup

	for _, session := range sessions {
		wg.Add(1)

		go func(session ct.Con) {
			defer wg.Done()
			if session.ProtoInfo == nil {
				return
			}

			srcIP := session.Origin.Src.String()
			srcPort := *session.Origin.Proto.SrcPort
			dstIP := session.Origin.Dst.String()
			dstPort := *session.Origin.Proto.DstPort

			_, ipNet, _ := net.ParseCIDR(podCIDR)
			ip := net.ParseIP(srcIP)

			if ipNet.Contains(ip) {

				queryStr := fmt.Sprintf(`kube_pod_info{pod_ip="%s"}`, srcIP)

				info, err := getPodInfo(queryStr)

				if err != nil {
					return
				}

				if len(info.Data.Result) == 0 {
					return
				}

				podName := info.Data.Result[0].Metric.Pod

				podNameSpace := info.Data.Result[0].Metric.Namespace

				ch <- prometheus.MustNewConstMetric(
					collector.podConnectStatus,
					prometheus.GaugeValue,
					float64(1),
					srcIP,
					strconv.Itoa(int(srcPort)),
					dstIP,
					strconv.Itoa(int(dstPort)),
					tcpStats[*session.ProtoInfo.TCP.State],
					podName,
					podNameSpace,
				)
			}
		}(session)
	}

	wg.Wait()
}

func getPodInfo(query string) (PodInfo, error) {

	prometheusUrl := fmt.Sprintf("http://%s/api/v1/query?query=", prometheusAddr)

	requestUrl := fmt.Sprintf("%s%s", prometheusUrl, url.QueryEscape(query))

	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, requestUrl, nil)

	if err != nil {
		log.Println(err)
		return PodInfo{}, err
	}

	res, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return PodInfo{}, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return PodInfo{}, err
	}

	var info PodInfo

	err = json.Unmarshal(body, &info)
	if err != nil {
		log.Println(err)
		return PodInfo{}, err
	}

	return info, nil
}

func podConnectMetrics() *prometheus.Desc {
	return prometheus.NewDesc(
		"pod_connect",
		"pod connect info",
		[]string{"src_ip", "src_port", "dst_ip", "dst_port", "stats", "pod_name", "pod_namespace"},
		nil,
	)
}

func newMetricsCollector() *ExporterMetrics {
	return &ExporterMetrics{
		podConnectStatus: podConnectMetrics(),
	}
}

func main() {
	flag.StringVar(&podCIDR, "pod-cidr", "10.100.0.0/16", "pod cidr")
	flag.StringVar(&prometheusAddr, "prometheus-addr", "10.101.20.9:9090", "prometheus addr")
	flag.StringVar(&listenPort, "port", "28880", "exporter listen port")
	flag.Parse()

	allMetrics := newMetricsCollector()
	prometheus.MustRegister(allMetrics)
	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: fmt.Sprintf(":%s", listenPort)}

	log.Printf("Starting server on port %s\n", listenPort)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {

			log.Printf("ListenAndServe(): %s\n\n", err)
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt, syscall.SIGKILL)

	// 阻塞,并等待signal信号
	<-sigChan
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
