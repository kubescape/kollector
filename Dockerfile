FROM alpine:3.5
EXPOSE 7666

COPY ./dist /.
COPY ./build_tag.txt /

CMD ./k8s-ca-dashboard-aggregator