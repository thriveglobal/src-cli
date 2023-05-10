package usage

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Option = func(config *Config)

type Config struct {
	namespace    string
	pod          string
	container    string
	docker       bool
	k8sClient    *kubernetes.Clientset
	dockerClient *client.Client
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

func K8s(ctx context.Context, clientSet *kubernetes.Clientset, restConfig *rest.Config, opts ...Option) error {
	cfg := &Config{
		namespace:    "default",
		pod:          "",
		container:    "",
		docker:       false,
		k8sClient:    clientSet,
		dockerClient: nil,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return listResourceUsage(ctx, cfg)
}

func listResourceUsage(ctx context.Context, cfg *Config) error {
	// code for listing resource usage
	// plug into prom API?
	if cfg.pod != "" {
		fmt.Printf("listing resource usage with POD (%s) works now.", cfg.pod)
		return nil
	}
    
	if cfg.container != "" {
		fmt.Printf("listing resource usage with CONTAINER (%s) works now.", cfg.container)
		return nil
	}
	// code for listing resource usage
	// plug into prom API?
	fmt.Println("listing resource usage works now.")
	return nil
}
