FROM golang:1.21-alpine as builder

ENV RELEASE=$image_version

ENV GO111MODULE=

ENV CGO_ENABLED=0

WORKDIR /work
ADD . .

RUN apk update && apk upgrade && \
    apk add --no-cache git busybox-static

WORKDIR /work
RUN go build -ldflags "-s" -o build/kollector
RUN addgroup -S ks && adduser -S ks -G ks

FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
USER ks
WORKDIR /home/ks/

COPY --from=builder /work/build/kollector /usr/bin/kollector
COPY --from=builder  /etc/ssl/certs  /etc/ssl/certs
COPY --from=builder  /bin/busybox.static /bin/busybox
COPY ./kollector-entrypoint.sh /home/ks/

ENTRYPOINT ["/bin/busybox","sh","/home/ks/kollector-entrypoint.sh"]

