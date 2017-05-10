FROM alpine:3.5

RUN apk add --no-cache --update ca-certificates

COPY bin/k8s-pod-notifier /k8s-pod-notifier

ENTRYPOINT ["/k8s-pod-notifier"]
