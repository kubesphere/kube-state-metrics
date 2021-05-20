FROM golang:1.13
RUN mkdir -p /go/src/k8s.io/kube-state-metrics
RUN chmod -R 777 /go
COPY / /go/src/k8s.io/kube-state-metrics
COPY /build/arm64/Makefile /go/src/k8s.io/kube-state-metrics/Makefile
WORKDIR /go/src/k8s.io/kube-state-metrics
RUN make build-arm

FROM gcr.io/distroless/static
COPY --from=0 /go/src/k8s.io/kube-state-metrics/kube-state-metrics /kube-state-metrics
USER nobody
ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]
EXPOSE 8080 8081
