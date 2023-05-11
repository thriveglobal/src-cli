package usage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/docker/docker/client"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Option = func(config *Config)

type Config struct {
	namespace     string
	pod           string
	container     string
	spy           bool
	docker        bool
	k8sClient     *kubernetes.Clientset
	dockerClient  *client.Client
	metricsClient *metricsv.Clientset
}

type PodUsageMetrics struct {
	cpuLimits        *resource.Quantity
	memLimits        *resource.Quantity
	cpuUsage         resource.Quantity
	memUsage         resource.Quantity
	cpuUsageFraction float64
	memUsageFraction float64
}

func WithNamespace(namespace string) Option {
	return func(config *Config) {
		config.namespace = namespace
	}
}

func WithPod(pod string) Option {
	return func(config *Config) {
		config.pod = pod
	}
}

func WithContainer(container string) Option {
	return func(config *Config) {
		config.container = container
	}
}

func UseSpy(spy bool) Option {
	return func(config *Config) {
		config.spy = true
	}
}

func K8s(ctx context.Context, clientSet *kubernetes.Clientset, metricsClient *metricsv.Clientset,
	restConfig *rest.Config, opts ...Option) error {
	cfg := &Config{
		namespace:     "default",
		pod:           "",
		container:     "",
		spy:           false,
		docker:        false,
		k8sClient:     clientSet,
		dockerClient:  nil,
		metricsClient: metricsClient,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return listResourceUsage(ctx, cfg)
}

func listResourceUsage(ctx context.Context, cfg *Config) error {
	if cfg.pod != "" {
		buf := new(bytes.Buffer)
		printResourceUsage(ctx, cfg, buf)
		fmt.Print(buf.String())
		return nil
	}

	if cfg.container != "" {
		fmt.Printf("listing resource usage with CONTAINER (%s) works now.", cfg.container)
		return nil
	}

	if cfg.spy {
		// Create a ticker that fires every 5 seconds
		ticker := time.NewTicker(1 * time.Second)
		// Create a channel to listen for a signal (like ctrl-c) to stop the ticker
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ticker.C:
				buf := new(bytes.Buffer)
				if err := printResourceUsage(ctx, cfg, buf); err != nil {
					return err
				}
				// Clear the terminal screen
				fmt.Print("\033[H\033[2J")
				// Print the buffer to the terminal
				fmt.Print(buf.String())
			case <-quit:
				ticker.Stop()
				return nil
			}
		}
	}

	buf := new(bytes.Buffer)
	if err := printResourceUsage(ctx, cfg, buf); err != nil {
		errors.Wrap(err, "error while printing resources: ")
	}

	fmt.Print(buf.String())
	return nil
}

func printResourceUsage(ctx context.Context, cfg *Config, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "POD NAME\tCPU USAGE\tCPU USAGE (%)\tMEMORY USAGE\tMEMORY USAGE (%)")

	if cfg.pod != "" {
		pod, err := cfg.k8sClient.CoreV1().Pods(cfg.namespace).Get(ctx, cfg.pod, metav1.GetOptions{})
		if err != nil {
			errors.Wrap(err, "error getting pod: ")
		}

		podUsage, err := getPodUsage(ctx, cfg, *pod)
		printPodUsage(ctx, pod.Name, podUsage, tw)
		tw.Flush()
		return nil
	}

	podInterface := cfg.k8sClient.CoreV1().Pods(cfg.namespace)
	podList, err := podInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "error listing pods: ")
	}

	for _, pod := range podList.Items {
		podUsage, err := getPodUsage(ctx, cfg, pod)
		if err != nil {
			return errors.Wrap(err, "error while getting pod usage metrics: ")
		}

		printPodUsage(ctx, pod.Name, podUsage, tw)
	}

	tw.Flush()
	return nil
}

func getPodUsage(ctx context.Context, cfg *Config, pod corev1.Pod) (PodUsageMetrics, error) {
	rawMetrics, err := cfg.metricsClient.MetricsV1beta1().PodMetricses(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
	if err != nil {
		return PodUsageMetrics{}, errors.Wrap(err, "error getting pod metrics from metrics API: ")
	}

	var podMetrics PodUsageMetrics

	if pod.GetNamespace() == cfg.namespace {
		for _, container := range pod.Spec.Containers {
			podMetrics.cpuLimits = container.Resources.Limits.Cpu()
			podMetrics.memLimits = container.Resources.Limits.Memory()
		}

		for _, container := range rawMetrics.Containers {
			podMetrics.cpuUsage = container.Usage[corev1.ResourceCPU]
			podMetrics.memUsage = container.Usage[corev1.ResourceMemory]
		}
	}
    
    podMetrics.cpuUsageFraction = float64(podMetrics.cpuUsage.MilliValue()) / float64(podMetrics.cpuLimits.MilliValue())
	podMetrics.memUsageFraction = float64(podMetrics.memUsage.Value()) / float64(podMetrics.memLimits.Value())
    
	return podMetrics, nil
}

func printPodUsage(ctx context.Context, podName string, podUsage PodUsageMetrics, tw *tabwriter.Writer) {
	fmt.Fprintf(tw, "%s\t%d\t%.2f%%\t%d\t%.2f%%\n",
		podName,
		podUsage.cpuUsage.Value(),
		podUsage.cpuUsageFraction*100,
		podUsage.memUsage.Value(),
		podUsage.memUsageFraction*100,
	)
}
