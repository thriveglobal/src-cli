package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	"github.com/sourcegraph/sourcegraph/lib/errors"
	"github.com/sourcegraph/src-cli/internal/scout/spy"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

func init() {
	cmdUsage := `'src scout spy' is a tool that tracks resource usage over a length of time.
    Part of the EXPERIMENTAL "src scout" tool.

    Examples
        spy on all pods
        $ src scout spy

        spy on all pods under given namespace
        $ src scout spy --namespace <namespace>
        
        spy on one pod
        $ src scout spy --pod <pod name>

        spy and output results to file
        $ src scout spy --o path/to/file
    `
	flagSet := flag.NewFlagSet("spy", flag.ExitOnError)
	usageFunc := func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of 'src scout %s':\n", flagSet.Name())
		flagSet.PrintDefaults()
		fmt.Println(cmdUsage)
	}

	var (
		kubeConfig *string
		namespace  = flagSet.String("namespace", "", "(optional) specify the kubernetes namespace to use")
		pod        = flagSet.String("pod", "", "(optional) specify a single pod")
		output     = flagSet.String("o", "", "(optional) output to specified file")
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
			return errors.Wrap(err, "failed to load .kube config:")
		}

		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			return errors.Wrap(err, "failed to initiate kubernetes client:")
		}

		metricsClient, err := metricsv.NewForConfig(config)
		if err != nil {
			return errors.Wrap(err, "failed to initiate metrics client")
		}

		var options []spy.Option
		if *namespace != "" {
			options = append(options, spy.WithNamespace(*namespace))
		}
		if *pod != "" {
			options = append(options, spy.WithPod(*pod))
		}
		if *output != "" {
			options = append(options, spy.WithOutput(*output))

		}
		return spy.K8s(
			context.Background(),
			clientSet,
			metricsClient,
			config,
			options...,
		)
	}

	scoutCommands = append(scoutCommands, &command{
		flagSet:   flagSet,
		handler:   handler,
		usageFunc: usageFunc,
	})
}
