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
		inCluster     bool
		kubeconfig    string
		labels        string
		namespace     string
		notifyFail    bool
		notifySuccess bool
		slackAPIToken string
		slackChannel  string
	)

	flags := flag.NewFlagSet("k8s-pod-notifier", flag.ExitOnError)
	flags.Usage = func() {
		flags.PrintDefaults()
	}

	flags.StringVar(&kubeContext, "context", "", "Kubernetes context")
	flags.BoolVar(&inCluster, "in-cluster", false, "Execute in Kubernetes cluster")
	flags.StringVar(&kubeconfig, "kubeconfig", "", "Path of kubeconfig")
	flags.StringVarP(&labels, "labels", "l", "", "Label filter query")
	flags.StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	flags.BoolVar(&notifyFail, "fail", false, "Notify failure of Pod only")
	flags.BoolVar(&notifySuccess, "success", false, "Notify success of Pod only")
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

	// Neither --fail nor --success was specified => Notify both fail and success
	if !(notifyFail || notifySuccess) {
		notifyFail = true
		notifySuccess = true
	}

	var k8sClient *k8s.Client

	if inCluster {
		c, err := k8s.NewClientInCluster()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if namespace == "" {
			namespace = k8s.DefaultNamespace()
		}

		k8sClient = c
	} else {
		c, err := k8s.NewClient(kubeconfig, kubeContext)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if namespace == "" {
			namespaceInConfig, err := c.NamespaceInConfig()
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

		k8sClient = c
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

	succeededFunc := func(event *k8s.PodEvent) error {
		title := fmt.Sprintf("Pod SUCCEEDED: %s - %s", event.Namespace, event.PodName)

		if err := slackClient.PostMessageWithAttachment(channelID, "good", title, "", []*slack.AttachmentField{
			&slack.AttachmentField{
				Title: "StartedAt",
				Value: event.StartedAt.String(),
			},
			&slack.AttachmentField{
				Title: "FinishedAt",
				Value: event.FinishedAt.String(),
			},
		}); err != nil {
			return err
		}

		fmt.Println("success: " + strings.Join([]string{event.Namespace, event.PodName, strconv.Itoa(event.ExitCode), event.Reason}, "\t"))

		return nil
	}
	failedFunc := func(event *k8s.PodEvent) error {
		title := fmt.Sprintf("Pod FAILED: %s - %s", event.Namespace, event.PodName)

		if err := slackClient.PostMessageWithAttachment(channelID, "danger", title, "", []*slack.AttachmentField{
			&slack.AttachmentField{
				Title: "StartedAt",
				Value: event.StartedAt.String(),
			},
			&slack.AttachmentField{
				Title: "FinishedAt",
				Value: event.FinishedAt.String(),
			},
			&slack.AttachmentField{
				Title: "ExitCode",
				Value: strconv.Itoa(event.ExitCode),
			},
			&slack.AttachmentField{
				Title: "Reason",
				Value: event.Reason,
			},
		}); err != nil {
			return err
		}

		fmt.Println("failed:  " + strings.Join([]string{event.Namespace, event.PodName, strconv.Itoa(event.ExitCode), event.Reason}, "\t"))

		return nil
	}

	fmt.Println("Watching...")

	if err := k8sClient.WatchPodEvents(ctx, namespace, labels, notifySuccess, notifyFail, succeededFunc, failedFunc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-sigCh
}
