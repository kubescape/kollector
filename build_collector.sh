#!/usr/bin/env bash
set -ex

export WTAG=test

# dep ensure
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-armo-collector .
chmod +x k8s-armo-collector

docker build --no-cache -f Dockerfile.test -t quay.io/armosec/k8s-armo-collector-ubi:$WTAG .
rm -rf k8s-armo-collector
# docker push quay.io/armosec/k8s-armo-collector-ubi:$WTAG
 
echo "update collector"

kubectl -n cyberarmor-system patch  deployment ca-dashboard-aggregator -p '{"spec": {"template": {"spec": { "containers": [{"name": "ca-dashboard-aggregator", "imagePullPolicy": "Never"}]}}}}' || true
kubectl set image deployment/ca-dashboard-aggregator -n cyberarmor-system ca-dashboard-aggregator=quay.io/armosec/k8s-armo-collector-ubi:$WTAG  
kubectl delete pod -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep ca-dashboard-aggregator |  awk '{print $1}')
kubectl logs -f -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep ca-dashboard-aggregator |  awk '{print $1}')