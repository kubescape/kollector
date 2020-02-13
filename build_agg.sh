#!/usr/bin/env bash

# dep ensure
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-ca-dashboard-aggregator .
chmod +x k8s-ca-dashboard-aggregator

# docker build --no-cache -t dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator-t:5 .
docker build --no-cache -t dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:localv11 .

rm -rf k8s-ca-dashboard-aggregator

docker push dreg.eust0.cyberarmorsoft.com:443/k8s-ca-dashboard-aggregator:localv11
