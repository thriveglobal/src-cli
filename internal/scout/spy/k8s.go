package spy

import (
	"context"
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

	for _, opt := range opts {
		opt(cfg)
	}

	pods, err := kube.GetPods(ctx, cfg)
	if err != nil {
		fmt.Println("failed to get pods")
		os.Exit(1)
	}

	raCh := make(chan ResourceAverages)
	for _, pod := range pods {
		go getAveragesOverTime(ctx, cfg, pod, raCh)
	}

	err = outputTableToFile(raCh, len(pods))
	if err != nil {
		return errors.Wrap(err, "failed to output table to file")
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
				cpuCh <- getPodUsage(ctx, cfg, pod, scout.CPU)
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
				memCh <- getPodUsage(ctx, cfg, pod, scout.MEMORY)
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

func outputTableToFile(ch chan ResourceAverages, numOfRows int) error {
	f, err := os.Create("/tmp/resource-averages.txt")
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer f.Close()

	t := tablewriter.NewWriter(f)
	t.SetHeader([]string{"Pod", "CPU AVG%", "MEM AVG%"})

	i := 1
	for ra := range ch {
		t.Append([]string{
			ra.PodName,
			fmt.Sprintf("%.2f%%", ra.CpuAverageUsage),
			fmt.Sprintf("%.2f%%", ra.MemAverageUsage),
		})

		if i == numOfRows {
			t.Render()
		} else {
			i++
		}
	}
	return nil
}

func getPodUsage(ctx context.Context, cfg *scout.Config, pod corev1.Pod, kind string) (usage float64) {
	usageMetrics, err := advise.GetUsageMetrics(ctx, cfg, pod)
	if err != nil {
		fmt.Printf("%s: failed to get usage metrics: %s\n", pod.Name, err)
		os.Exit(1)
	}

	for _, container := range usageMetrics {
		if kind == scout.CPU {
			usage += container.CpuUsage
		} else if kind == scout.MEMORY {
			usage += container.MemoryUsage
		} else {
			fmt.Printf("%s is an invalid argument for 'kind', use '%s' or '%s'", kind, scout.MEMORY, scout.CPU)
			os.Exit(1)
		}
	}

	return
}
