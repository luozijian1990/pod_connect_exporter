package container

import (
	"context"
	"fmt"

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
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd: %v", err)
	}
	defer client.Close()

	// 设置上下文中的命名空间，默认是 "k8s.io"
	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")

	var containersInfos []ContainerInfo

	// 列出所有容器
	containers, err := client.Containers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %v", err)
	}

	// 收集容器信息
	for _, container := range containers {
		task, err := container.Task(ctx, nil)
		if err != nil {
			continue
		}

		labels, err := container.Labels(ctx)
		if err != nil {
			fmt.Printf("Error getting labels: %v\n", err)
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

	return containersInfos, nil
}
