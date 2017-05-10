package kubernetes

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client represents the wrapper of Kubernetes API client
type Client struct {
	clientConfig clientcmd.ClientConfig
	clientset    *kubernetes.Clientset
}

// PodEvent represents Pod termination event
type PodEvent struct {
	Namespace  string
	PodName    string
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int
	Reason     string
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc func(event *PodEvent) error

// NewClient creates Client object using local kubecfg
func NewClient(kubeconfig, context string) (*Client, error) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: context})

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "falied to load local kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load clientset")
	}

	return &Client{
		clientConfig: clientConfig,
		clientset:    clientset,
	}, nil
}

// NewClientInCluster creates Client object in Kubernetes cluster
func NewClientInCluster() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kubeconfig in cluster")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "falied to load clientset")
	}

	return &Client{
		clientset: clientset,
	}, nil
}

// NamespaceInConfig returns namespace set in kubeconfig
func (c *Client) NamespaceInConfig() (string, error) {
	if c.clientConfig == nil {
		return "", errors.New("clientConfig is not set")
	}

	rawConfig, err := c.clientConfig.RawConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to load rawConfig")
	}

	return rawConfig.Contexts[rawConfig.CurrentContext].Namespace, nil
}

// WatchPodEvents watches Pod events
func (c *Client) WatchPodEvents(ctx context.Context, namespace, labels string, succeededFunc, failedFunc NotifyFunc) error {
	watcher, err := c.clientset.Core().Pods(namespace).Watch(v1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return errors.Wrap(err, "cannot create Pod event watcher")
	}

	go func() {
		for {
			select {
			case e := <-watcher.ResultChan():
				if e.Object == nil {
					return
				}

				pod, ok := e.Object.(*v1.Pod)
				if !ok {
					continue
				}

				switch e.Type {
				case watch.Modified:
					if pod.DeletionTimestamp != nil {
						continue
					}

					startedAt := pod.CreationTimestamp.Time

					switch pod.Status.Phase {
					case v1.PodSucceeded:
						for _, cst := range pod.Status.ContainerStatuses {
							if cst.State.Terminated == nil {
								continue
							}

							finishedAt := cst.State.Terminated.FinishedAt.Time

							succeededFunc(&PodEvent{
								Namespace:  pod.Namespace,
								PodName:    pod.Name,
								StartedAt:  startedAt,
								FinishedAt: finishedAt,
								ExitCode:   0,
								Reason:     "",
							})

							break
						}
					case v1.PodFailed:
						for _, cst := range pod.Status.ContainerStatuses {
							if cst.State.Terminated == nil {
								continue
							}

							finishedAt := cst.State.Terminated.FinishedAt.Time

							failedFunc(&PodEvent{
								Namespace:  pod.Namespace,
								PodName:    pod.Name,
								StartedAt:  startedAt,
								FinishedAt: finishedAt,
								ExitCode:   int(cst.State.Terminated.ExitCode),
								Reason:     cst.State.Terminated.Reason,
							})

							break
						}
					}
				}
			case <-ctx.Done():
				watcher.Stop()
				return
			}
		}
	}()

	return nil
}
