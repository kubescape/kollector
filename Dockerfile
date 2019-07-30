FROM golang:alpine
RUN apk add git
RUN apk update && apk add ca-certificates

RUN go get github.com/gorilla/websocket k8s.io/api/core/v1 k8s.io/apimachinery/pkg/apis/meta/v1 k8s.io/apimachinery/pkg/labels k8s.io/client-go/kubernetes k8s.io/client-go/rest k8s.io/client-go/tools/clientcmd
ADD . src/k8s-ca-dashboard-aggregator
ENV CGO_ENABLED=0
RUN cd src/k8s-ca-dashboard-aggregator && go build main.go
RUN cd src/k8s-ca-dashboard-aggregator && cp main /main
ENTRYPOINT ["./src/k8s-ca-dashboard-aggregator/main"]