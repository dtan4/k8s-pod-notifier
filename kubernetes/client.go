package kubernetes

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
)

// Client represents the wrapper of Kubernetes API client
type Client struct {
	clientConfig clientcmd.ClientConfig
	clientset    *kubernetes.Clientset
}

// NotifyFunc represents callback function for Pod event
type NotifyFunc func(namespace, podName string, exitCode int, reason string) error

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
					switch pod.Status.Phase {
					case v1.PodSucceeded:
						succeededFunc(pod.Namespace, pod.Name, 0, "")
					case v1.PodFailed:
						for _, cst := range pod.Status.ContainerStatuses {
							if cst.State.Terminated == nil {
								continue
							}

							failedFunc(pod.Namespace, pod.Name, int(cst.State.Terminated.ExitCode), cst.State.Terminated.Reason)
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
