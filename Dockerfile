FROM alpine:3.5
RUN apk update && apk add ca-certificates

COPY ./dist /.
COPY ./build_tag.txt /

ENV CA_AGGREGATOR_IMAGE_VERSION ${BUILD_NUMBER}
CMD /k8s-ca-dashboard-aggregator
ENTRYPOINT ["/k8s-ca-dashboard-aggregator"]
