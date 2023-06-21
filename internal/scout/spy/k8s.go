package spy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/src-cli/internal/scout"
	"github.com/sourcegraph/src-cli/internal/scout/advise"
	"github.com/sourcegraph/src-cli/internal/scout/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type ResourceAverages struct {
	PodName         string
	CpuAverageUsage float64
	MemAverageUsage float64
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

	// @TODO rewrite this to include windows file path
	filePath := "/tmp/resource-averages.txt"
	f, err := os.Create(filePath)
	defer f.Close()

	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}

	for _, opt := range opts {
		opt(cfg)
	}

	pods, err := kube.GetPods(ctx, cfg)
	if err != nil {
		fmt.Println("failed to get pods")
		os.Exit(1)
	}

	t := tablewriter.NewWriter(f)
	t.SetHeader([]string{"Pod", "CPU AVG%", "MEM AVG%"})

	raCh := make(chan ResourceAverages)
	for _, pod := range pods {
		go getAveragesOverTime(ctx, cfg, pod, raCh)
	}

	i := 1
	for ra := range raCh {
		t.Append([]string{
			ra.PodName,
			fmt.Sprintf("%.2f%%", ra.CpuAverageUsage),
			fmt.Sprintf("%.2f%%", ra.MemAverageUsage),
		})

		if i == len(pods) {
			t.Render()
		} else {
			i++
		}
	}

	return nil
}

func getAveragesOverTime(ctx context.Context, cfg *scout.Config, pod corev1.Pod, ch chan ResourceAverages) error {
	cpuCh := make(chan float64)
	memCh := make(chan float64)
	cpus := []float64{}
	mems := []float64{}
	var cpuAvg float64
	var memAvg float64
	var ra ResourceAverages

	ctx, cancel := context.WithCancel(ctx)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			ra.PodName = pod.Name
			ra.CpuAverageUsage = cpuAvg
			ra.MemAverageUsage = memAvg
			ch <- ra
			cancel()
		}
	}()

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("cleaning up %s...\n", pod.Name)
				time.Sleep(2 * time.Second)
				os.Exit(0)
			default:
				cpuCh <- getPodCPUUsage(ctx, cfg, pod)
				time.Sleep(15 * time.Second)
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("cleaning up %s...\n", pod.Name)
				time.Sleep(2 * time.Second)
				os.Exit(0)
			default:
				memCh <- getPodMemoryUsage(ctx, cfg, pod)
				time.Sleep(15 * time.Second)
			}
		}
	}(ctx)

	go func() {
		for cpu := range cpuCh {
			if reflect.DeepEqual(cpus, []float64{}) || cpus[len(cpus)-1] != cpu {
				cpus = append(cpus, cpu)
				cpuAvg = scout.GetAverage(cpus)
			}
		}
	}()

	go func() {
		for mem := range memCh {
			if reflect.DeepEqual(mems, []float64{}) || mems[len(mems)-1] != mem {
				mems = append(mems, mem)
				memAvg = scout.GetAverage(mems)
			}
		}
	}()

	return nil
}

func getPodCPUUsage(ctx context.Context, cfg *scout.Config, pod corev1.Pod) float64 {
	var cpuUsage float64
	usageMetrics, err := advise.GetUsageMetrics(ctx, cfg, pod)
	if err != nil {
		fmt.Printf("%s: failed to get usage metrics: %s\n", pod.Name, err)
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
		fmt.Printf("%s: failed to get usage metrics: %s\n", pod.Name, err)
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
		os.Exit(1)
	}
	return
}
