package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	k8s "github.com/dtan4/k8s-pod-notifier/kubernetes"
	"github.com/dtan4/k8s-pod-notifier/slack"
	flag "github.com/spf13/pflag"
)

func main() {
	var (
		kubeContext   string
		kubeconfig    string
		labels        string
		namespace     string
		slackAPIToken string
		slackChannel  string
	)

	flags := flag.NewFlagSet("k8s-pod-notifier", flag.ExitOnError)
	flags.Usage = func() {
		flags.PrintDefaults()
	}

	flags.StringVar(&kubeContext, "context", "", "Kubernetes context")
	flags.StringVar(&kubeconfig, "kubeconfig", "", "Path of kubeconfig")
	flags.StringVarP(&labels, "labels", "l", "", "Label filter query")
	flags.StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	flags.StringVar(&slackAPIToken, "slack-api-token", "", "Slack API token")
	flags.StringVar(&slackChannel, "slack-channel", "", "Slack channel to post")

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

	if slackAPIToken == "" {
		if os.Getenv("SLACK_API_TOKEN") == "" {
			fmt.Fprintln(os.Stderr, "Slack API TOKEN must be set (SLACK_API_TOKEN, --slack-api-token)")
			os.Exit(1)
		}

		slackAPIToken = os.Getenv("SLACK_API_TOKEN")
	}

	k8sClient, err := k8s.NewClient(kubeconfig, kubeContext)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if namespace == "" {
		namespaceInConfig, err := k8sClient.NamespaceInConfig()
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

	slackClient := slack.NewClient(slackAPIToken)

	channelID, err := slackClient.GetChannelID(slackChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	succeededFunc := func(namespace, podName string, exitCode int, reason string) error {
		title := fmt.Sprintf("Pod SUCCEEDED: %s - %s", namespace, podName)

		if err := slackClient.PostMessageWithAttachment(channelID, "good", title, "", map[string]string{
			"Result": "Succeeded",
		}); err != nil {
			return err
		}

		fmt.Println("success: " + strings.Join([]string{namespace, podName, strconv.Itoa(exitCode), reason}, "\t"))
		return nil
	}
	failedFunc := func(namespace, podName string, exitCode int, reason string) error {
		title := fmt.Sprintf("Pod FAILED: %s - %s", namespace, podName)

		if err := slackClient.PostMessageWithAttachment(channelID, "danger", title, "", map[string]string{
			"ExitCode": strconv.Itoa(exitCode),
			"Reason":   reason,
		}); err != nil {
			return err
		}

		fmt.Println("failed:  " + strings.Join([]string{namespace, podName, strconv.Itoa(exitCode), reason}, "\t"))
		return nil
	}

	fmt.Println("Watching...")

	if err := k8sClient.WatchPodEvents(ctx, namespace, labels, succeededFunc, failedFunc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-sigCh
}
