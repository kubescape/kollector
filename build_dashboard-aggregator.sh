#!/usr/bin/env bash
set -ex

export WTAG=test

# dep ensure
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-ca-dashboard-aggregator .
chmod +x k8s-ca-dashboard-aggregator

docker build --no-cache -f Dockerfile.test -t quay.io/armosec/k8s-ca-dashboard-aggregator-ubi:$WTAG .
rm -rf k8s-ca-dashboard-aggregator
# docker push quay.io/armosec/k8s-ca-dashboard-aggregator:$WTAG
 
echo "update dashboard-aggregator"

kubectl set image deployment/ca-dashboard-aggregator -n cyberarmor-system ca-aggregator=quay.io/armosec/k8s-ca-dashboard-aggregator-ubi:$WTAG || true
kubectl delete pod -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep ca-dashboard-aggregator |  awk '{print $1}')
kubectl logs -f -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep ca-dashboard-aggregator |  awk '{print $1}')