package spy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/sourcegraph/src-cli/internal/scout"
	"github.com/sourcegraph/src-cli/internal/scout/advise"
	"github.com/sourcegraph/src-cli/internal/scout/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

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

	for _, pod := range pods {
		go getAveragesOverTime(ctx, cfg, pod.Name, pods)
	}

	time.Sleep(500 * time.Second)
	return nil
}

func getAveragesOverTime(ctx context.Context, cfg *scout.Config, podName string, pods []corev1.Pod) error {
	gitserverCpuCh := make(chan float64)
	gitserverMemCh := make(chan float64)
	cpus := []float64{}
	mems := []float64{}
	var cpuAvg float64
	var memAvg float64

	pod, err := kube.GetPod(podName, pods)
	if err != nil {
		return err
	}

	go func() {
		for {
			gitserverCpuCh <- getPodCPUUsage(ctx, cfg, pod)
			time.Sleep(8 * time.Second)
		}
	}()

	go func() {
		for {
			gitserverMemCh <- getPodMemoryUsage(ctx, cfg, pod)
			time.Sleep(8 * time.Second)
		}
	}()

	go func() {
		for cpu := range gitserverCpuCh {
			if reflect.DeepEqual(cpus, []float64{}) || cpus[len(cpus)-1] != cpu {
				cpus = append(cpus, cpu)
				cpuAvg = scout.GetAverage(cpus)
				fmt.Printf("%s: cpu average: %v\n", podName, cpuAvg)
			}
		}
	}()

	go func() {
		for mem := range gitserverMemCh {
			if reflect.DeepEqual(mems, []float64{}) || mems[len(mems)-1] != mem {
				mems = append(mems, mem)
				memAvg = scout.GetAverage(mems)
				fmt.Printf("%s: mem average: %v\n", podName, memAvg)
			}
		}
	}()

	time.Sleep(120 * time.Second)
	return nil
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
