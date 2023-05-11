package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/sourcegraph/sourcegraph/lib/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/sourcegraph/src-cli/internal/scout/resource"
	"github.com/sourcegraph/src-cli/internal/scout/usage"
)

func init() {
	cmdusage := `'src scout usage' is a tool that provides an overview of current resource 
    usage across an instance of Sourcegraph. Part of the EXPERIMENTAL "src scout" tool.
    
    Examples
        List pods and resource usage in a Kubernetes deployment:
        $ src scout usage

        List containers and resource usage in a Docker deployment:
        $ src scout usage --docker

        Add namespace if using namespace in a Kubernetes cluster
        $ src scout usage --namespace sg

        List usage for a specified pod
        $ src scout usage --pod <pod name>

        List usage for a specified container
        $ src scout usage --container <container name>
    `

	flagSet := flag.NewFlagSet("usage", flag.ExitOnError)
	usageFunc := func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of 'src scout %s':\n", flagSet.Name())
		flagSet.PrintDefaults()
		fmt.Println(cmdusage)
	}

	var (
		kubeConfig *string
		namespace  = flagSet.String("namespace", "", "(optional) specify the kubernetes namespace to use")
		pod        = flagSet.String("pod", "", "(optional) list usage for a specific pod")
		container  = flagSet.String("container", "", "(optional) list usage for a specific container")
		spy        = flagSet.Bool("spy", false, "(optional) watch cpu and mem usage in real time")
		docker     = flagSet.Bool("docker", false, "(optional) using docker deployment")
	)

	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flagSet.String(
			"kubeconfig",
			filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file",
		)
	} else {
		kubeConfig = flagSet.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	handler := func(args []string) error {
		if err := flagSet.Parse(args); err != nil {
			return err
		}

		config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			return errors.New(fmt.Sprintf("%v: failed to load kubernetes config", err))
		}

		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			return errors.New(fmt.Sprintf("%v: failed to load kubernetes config", err))
		}

		metricsClient, err := metricsv.NewForConfig(config)
		if err != nil {
			return errors.New(fmt.Sprintf("%v: failed to load metrics config", err))
		}

		var options []usage.Option

		if *namespace != "" {
			options = append(options, usage.WithNamespace(*namespace))
		}

		if *pod != "" {
			options = append(options, usage.WithPod(*pod))
		}

		if *container != "" {
			options = append(options, usage.WithContainer(*container))
		}
        
        if *spy {
            options = append(options, usage.UseSpy(*spy))
        }

		if *docker {
			dockerClient, err := client.NewClientWithOpts(client.FromEnv)
			if err != nil {
				return errors.Wrap(err, "error creating docker client: ")
			}

			return resource.Docker(context.Background(), dockerClient)
		}

		return usage.K8s(context.Background(), clientSet, metricsClient, config, options...)
	}

	scoutCommands = append(scoutCommands, &command{
		flagSet:   flagSet,
		handler:   handler,
		usageFunc: usageFunc,
	})
}
