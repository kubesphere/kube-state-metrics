
```shell
git clone https://github.com/kubesphere/kube-state-metrics.git

cd kube-state-metrics 

git checkout -b ks-v1.9.7 origin/ks-v1.9.7

# docker buildx build -f build/arm64/Dockerfile --output=type=registry --platform linux/arm64 -t kubesphere/kube-state-metrics:v1.9.7-arm64 .
docker buildx build -f build/arm64/Dockerfile --output type=docker,dest=kube-state-metrics:v1.9.7-arm64.tar --platform linux/arm64 -t kubesphere/kube-state-metrics:v1.9.7-arm64 .
```
