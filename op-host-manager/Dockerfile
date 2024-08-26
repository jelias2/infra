FROM golang:1.22.0-alpine3.18 AS builder

ARG GITCOMMIT=docker
ARG GITDATE=docker
ARG GITVERSION=docker

RUN apk add make git gcc musl-dev linux-headers

COPY ./op-host-manager /app

WORKDIR /app

RUN make op-host-manager

FROM alpine:3.18

RUN apk add bind-tools jq curl bash

COPY ./op-host-manager/entrypoint.sh /bin/entrypoint.sh

RUN apk update && \
  apk add ca-certificates && \
  chmod +x /bin/entrypoint.sh

EXPOSE 8080

VOLUME /etc/op-host-manager

COPY --from=builder /app/bin/op-host-manager /bin/op-host-manager

CMD ["/bin/op-host-manager"]
