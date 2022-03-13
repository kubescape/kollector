FROM alpine:3.5
RUN apk update && apk add ca-certificates

COPY ./dist /.
COPY ./build_number.txt /
RUN echo $(date -u) > /build_date.txt

ENTRYPOINT ["/k8s-armo-collector"]
