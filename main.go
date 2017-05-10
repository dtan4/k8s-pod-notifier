package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	k8s "github.com/dtan4/k8s-pod-notifier/kubernetes"
	flag "github.com/spf13/pflag"
)

func main() {
	var (
		kubeContext string
		kubeconfig  string
		labels      string
		namespace   string
	)

	flags := flag.NewFlagSet("k8s-pod-notifier", flag.ExitOnError)
	flags.Usage = func() {
		flags.PrintDefaults()
	}

	flags.StringVar(&kubeContext, "context", "", "Kubernetes context")
	flags.StringVar(&kubeconfig, "kubeconfig", "", "Path of kubeconfig")
	flags.StringVarP(&labels, "labels", "l", "", "Label filter query")
	flags.StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")

	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if kubeconfig == "" {
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		} else {
			kubeconfig = k8s.DefaultConfigFile()
		}
	}

	client, err := k8s.NewClient(kubeconfig, kubeContext)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if namespace == "" {
		namespaceInConfig, err := client.NamespaceInConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if namespaceInConfig == "" {
			namespace = k8s.DefaultNamespace()
		} else {
			namespace = namespaceInConfig
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := client.WatchPodEvents(ctx, namespace, labels); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-sigCh
}
