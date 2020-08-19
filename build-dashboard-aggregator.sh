#!/usr/bin/env bash
set -ex

export WTAG=test

# dep ensure
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-ca-dashboard-aggregator .
chmod +x k8s-ca-dashboard-aggregator

docker build --no-cache -f Dockerfile.test -t dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:$WTAG .
rm -rf k8s-ca-dashboard-aggregator
# docker push dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:$WTAG
 
echo "update dashboard-aggregator"

kubectl set image deployment/ca-dashboard-aggregator -n cyberarmor-system ca-aggregator=dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:$WTAG || true
kubectl delete pod -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep k8s-ca-dashboard-aggregator |  awk '{print $1}')
kubectl logs -f -n cyberarmor-system $(kubectl get pod -n cyberarmor-system | grep k8s-ca-dashboard-aggregator |  awk '{print $1}')