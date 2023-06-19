package spy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sourcegraph/src-cli/internal/scout"
	"github.com/sourcegraph/src-cli/internal/scout/advise"
	"github.com/sourcegraph/src-cli/internal/scout/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodMetrics map[string]PodUsage
type PodUsage struct {
	CpuUsage []float64
	MemUsage []float64
}

func K8s(
	ctx context.Context,
	k8sClient *kubernetes.Clientset,
	metricsClient *metricsv.Clientset,
	restConfig *rest.Config,
	opts ...Option,
) error {
	cfg := &scout.Config{
		Namespace:     "default",
		Pod:           "",
		Output:        "",
		RestConfig:    restConfig,
		K8sClient:     k8sClient,
		MetricsClient: metricsClient,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	pods, err := kube.GetPods(ctx, cfg)
	if err != nil {
		fmt.Println("failed to get pods")
		os.Exit(1)
	}

	metMap := initPodMetMap(pods)
	for i := 0; i < 5; i++ {
		pingPods(ctx, cfg, pods, metMap)
	}

	for k, v := range metMap {
		fmt.Printf("%s: Memory:%.2f, CPU:%.2f", k, scout.GetAverage(v.CpuUsage), scout.GetAverage(v.MemUsage))
	}

	return nil
}

func pingPods(ctx context.Context, cfg *scout.Config, pods []corev1.Pod, metMap map[string]PodUsage) {
	for _, pod := range pods {
		updateMets(ctx, cfg, pod, metMap)
	}
}

func updateMets(ctx context.Context, cfg *scout.Config, pod corev1.Pod, metMap map[string]PodUsage) {
	met := metMap[pod.Name]

	cpuUsage := getPodCPUUsage(ctx, cfg, pod)
	memUsage := getPodMemoryUsage(ctx, cfg, pod)

	met.CpuUsage = append(met.CpuUsage, cpuUsage)
	met.MemUsage = append(met.MemUsage, memUsage)

	metMap[pod.Name] = met
}

func initPodMetMap(pods []corev1.Pod) map[string]PodUsage {
	metMap := make(map[string]PodUsage)
	for _, pod := range pods {
		podUsage := PodUsage{
			CpuUsage: []float64{},
			MemUsage: []float64{},
		}
		metMap[pod.Name] = podUsage
	}
	return metMap
}

func getPodCPUUsage(ctx context.Context, cfg *scout.Config, pod corev1.Pod) float64 {
	var cpuUsage float64
	usageMetrics, err := advise.GetUsageMetrics(ctx, cfg, pod)
	if err != nil {
		fmt.Println("could not get usage metrics")
		os.Exit(1)
	}

	for _, container := range usageMetrics {
		cpuUsage += container.CpuUsage
	}

	return cpuUsage
}

func getPodMemoryUsage(ctx context.Context, cfg *scout.Config, pod corev1.Pod) float64 {
	var memUsage float64
	usageMetrics, err := advise.GetUsageMetrics(ctx, cfg, pod)
	if err != nil {
		fmt.Println("could not get usage metrics")
		os.Exit(1)
	}

	for _, container := range usageMetrics {
		memUsage += container.MemoryUsage
	}

	return memUsage
}

func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}
