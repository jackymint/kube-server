package metrics

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

type NodeMetrics struct {
	Name       string
	CPUUsage   string
	MemUsage   string
	CPUPercent float64
	MemPercent float64
}

type PodMetrics struct {
	Name      string
	Namespace string
	Node      string
	CPUUsage  string
	MemUsage  string
	Status    string
	Restarts  int32
}

type Collector struct {
	client        *kubernetes.Clientset
	metricsClient *metricsv1beta1.Clientset
}

func NewCollector(kubeconfig string) (*Collector, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	metricsClient, err := metricsv1beta1.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Collector{client: client, metricsClient: metricsClient}, nil
}

func (c *Collector) GetNodeMetrics(ctx context.Context) ([]NodeMetrics, error) {
	nodeMetrics, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodes, err := c.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodeCapacity := map[string]struct{ cpu, mem int64 }{}
	for _, n := range nodes.Items {
		nodeCapacity[n.Name] = struct{ cpu, mem int64 }{
			cpu: n.Status.Capacity.Cpu().MilliValue(),
			mem: n.Status.Capacity.Memory().Value(),
		}
	}

	var result []NodeMetrics
	for _, nm := range nodeMetrics.Items {
		cap := nodeCapacity[nm.Name]
		cpuUsage := nm.Usage.Cpu().MilliValue()
		memUsage := nm.Usage.Memory().Value()

		cpuPct := float64(0)
		memPct := float64(0)
		if cap.cpu > 0 {
			cpuPct = float64(cpuUsage) / float64(cap.cpu) * 100
		}
		if cap.mem > 0 {
			memPct = float64(memUsage) / float64(cap.mem) * 100
		}

		result = append(result, NodeMetrics{
			Name:       nm.Name,
			CPUUsage:   fmt.Sprintf("%dm", cpuUsage),
			MemUsage:   fmt.Sprintf("%dMi", memUsage/1024/1024),
			CPUPercent: cpuPct,
			MemPercent: memPct,
		})
	}
	return result, nil
}

func (c *Collector) GetPodMetrics(ctx context.Context, namespace string) ([]PodMetrics, error) {
	podMetrics, err := c.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	podInfo := map[string]struct {
		node     string
		status   string
		restarts int32
	}{}
	for _, p := range pods.Items {
		restarts := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
		}
		podInfo[p.Name] = struct {
			node     string
			status   string
			restarts int32
		}{
			node:     p.Spec.NodeName,
			status:   string(p.Status.Phase),
			restarts: restarts,
		}
	}

	var result []PodMetrics
	for _, pm := range podMetrics.Items {
		cpuTotal := int64(0)
		memTotal := int64(0)
		for _, c := range pm.Containers {
			cpuTotal += c.Usage.Cpu().MilliValue()
			memTotal += c.Usage.Memory().Value()
		}
		info := podInfo[pm.Name]
		result = append(result, PodMetrics{
			Name:      pm.Name,
			Namespace: pm.Namespace,
			Node:      info.node,
			CPUUsage:  fmt.Sprintf("%dm", cpuTotal),
			MemUsage:  fmt.Sprintf("%dMi", memTotal/1024/1024),
			Status:    info.status,
			Restarts:  info.restarts,
		})
	}
	return result, nil
}
