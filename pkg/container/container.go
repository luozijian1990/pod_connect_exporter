package container

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

// ContainerInfo 存储容器信息
type ContainerInfo struct {
	PID       uint32
	PodName   string
	Namespace string
	Container string
}

// ListContainers 列出所有容器
func ListContainers() ([]ContainerInfo, error) {
	startTime := time.Now()
	log.Printf("开始连接containerd并获取容器列表...")

	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		log.Printf("连接containerd失败: %v", err)
		return nil, fmt.Errorf("failed to connect to containerd: %v", err)
	}
	defer client.Close()
	log.Printf("成功连接到containerd")

	// 设置上下文中的命名空间，默认是 "k8s.io"
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

	var containersInfos []ContainerInfo

	// 列出所有容器
	containers, err := client.Containers(ctx)
	if err != nil {
		log.Printf("获取容器列表失败: %v", err)
		return nil, fmt.Errorf("failed to list containers: %v", err)
	}
	log.Printf("获取到 %d 个容器", len(containers))

	// 收集容器信息
	containerCount := 0
	skipCount := 0
	for _, container := range containers {
		id := container.ID()
		task, err := container.Task(ctx, nil)
		if err != nil {
			log.Printf("跳过容器 %s: 无法获取任务信息: %v", id, err)
			skipCount++
			continue
		}

		labels, err := container.Labels(ctx)
		if err != nil {
			log.Printf("跳过容器 %s: 无法获取标签: %v", id, err)
			skipCount++
			continue
		}

		s, b := labels["io.cri-containerd.kind"]

		pid := task.Pid()

		if pid == 0 {
			log.Printf("跳过容器 %s: 没有PID", id)
			skipCount++
			continue
		}

		if b && s == "container" {
			podName := labels["io.kubernetes.pod.name"]
			namespace := labels["io.kubernetes.pod.namespace"]
			containerName := labels["io.kubernetes.container.name"]

			containersInfos = append(containersInfos, ContainerInfo{
				PID:       pid,
				PodName:   podName,
				Namespace: namespace,
				Container: containerName,
			})

			containerCount++
			log.Printf("添加容器: [%s/%s/%s] PID: %d", namespace, podName, containerName, pid)
		} else {
			log.Printf("跳过容器 %s: 不是标准Kubernetes容器", id)
			skipCount++
		}
	}

	duration := time.Since(startTime)
	log.Printf("容器信息收集完成，共获取 %d 个容器（跳过 %d 个），耗时: %v", containerCount, skipCount, duration)

	return containersInfos, nil
}
