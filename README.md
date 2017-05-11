# k8s-pod-notifier

[![Build Status](https://travis-ci.org/dtan4/k8s-pod-notifier.svg?branch=master)](https://travis-ci.org/dtan4/k8s-pod-notifier)
[![Docker Repository on Quay](https://quay.io/repository/dtan4/k8s-pod-notifier/status "Docker Repository on Quay")](https://quay.io/repository/dtan4/k8s-pod-notifier)

Notify Pod status to Slack

## Requirements

- Kubernetes 1.3 or above
- Slack API (OAuth2) access token
  - Permission scopes `channels:read` and `chat:write:bot` are required

## Installation

### From source

```bash
$ go get -d github.com/dtan4/k8s-pod-notifier
$ cd $GOPATH/src/github.com/dtan4/k8s-pod-notifier
$ make deps
$ make install
```

### Run in a Docker container

Docker image is available at [quay.io/dtan4/k8s-pod-notifier](https://quay.io/repository/dtan4/k8s-pod-notifier).

```bash
# -t is required to colorize logs
$ docker run \
    --rm \
    -t \
    -v $HOME/.kube/config:/.kube/config \
    quay.io/dtan4/k8s-pod-notifier:latest
```

## Usage

### In Kubernetes cluster

Just add `--in-cluster` flag.

Deployment manifest sample:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: k8s-pod-notifier
spec:
  minReadySeconds: 30
  strategy:
    type: RollingUpdate
  replicas: 1
  template:
    metadata:
      name: k8s-pod-notifier
      labels:
        name: k8s-pod-notifier
        role: daemon
    spec:
      containers:
      - image: quay.io/dtan4/k8s-pod-notifier:latest
        name: k8s-pod-notifier
        env:
        - name: SLACK_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: k8s-pod-notifier
              key: slack-api-token
        - name: SLACK_CHANNEL
          valueFrom:
            secretKeyRef:
              name: k8s-pod-notifier
              key: slack-channel
        command:
          - "/k8s-pod-notifier"
          - "--in-cluster"
          - "--fail"
          - "--labels"
          - "role=job"
```

### Local machine

k8s-pod-notifier uses `~/.kube/config` as default. You can specify another path by `KUBECONFIG` environment variable or `--kubeconfig` option. `--kubeconfig` option always overrides `KUBECONFIG` environment variable.

```bash
$ export SLACK_API_TOKEN=xxxxx
$ export SLACK_CHANNEL=notifications
$ KUBECONFIG=/path/to/kubeconfig k8s-pod-notifier
# or
$ k8s-pod-notifier --kubeconfig=/path/to/kubeconfig
```

### Options

|Option|Description|Required|Default|
|---------|-----------|-------|-------|
|`--context=CONTEXT`|Kubernetes context|||
|`--in-cluster`|Execute in Kubernetes cluster|||
|`--kubeconfig=KUBECONFIG`|Path of kubeconfig||`~/.kube/config`|
|`--labels=LABELS`|Label filter query (e.g. `app=APP,role=ROLE`)|||
|`--namespace=NAMESPACE`|Kubernetes namespace||All namespaces|
|`--success`|Notify success of Pod only|||
|`--fail`|Notify failure of Pod only|||
|`--slack-api-token=SLACK_API_TOKEN`|Slack API token|Required, or set `SLACK_API_TOKEN` env||
|`--slack-channel=SLACK_CHANNEL`|Slack channel to post|Required, or set `SLACK_CHANNEL` env||
|`-h`, `-help`|Print command line usage|||
|`-v`, `-version`|Print version|||

## Development

Go 1.7 or above is required.
Clone this repository and build using `make`.

```bash
$ go get -d github.com/dtan4/k8s-pod-notifier
$ cd $GOPATH/src/github.com/dtan4/k8s-pod-notifier
$ make
```

## Author

Daisuke Fujita ([@dtan4](https://github.com/dtan4))

## License

[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)
