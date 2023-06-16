package spy

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/lib/errors"
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
		return errors.Wrap(err, "could not get list of pods")
	}

	for _, pod := range pods {
		fmt.Println(pod.Name)
	}
	return nil
}

/* func pingPod(ctx context.Context, cfg *scout.Config, pod corev1.Pod) error {
	var pings []float64
	usageMetrics, err := advise.GetUsageMetrics(ctx, cfg, pod)
	if err != nil {
		return errors.Wrap(err, "failed to get usage metrics:")
	}
    return nil
} */
