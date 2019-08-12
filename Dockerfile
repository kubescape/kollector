FROM alpine:3.5
RUN apk update && apk add ca-certificates
COPY ./dist /.
COPY ./build_tag.txt /

CMD ./k8s-ca-dashboard-aggregator