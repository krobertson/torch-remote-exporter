FROM golang:alpine AS builder

ARG TARGETOS
ARG TARGETARCH

ENV GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    GO111MODULE=on \
    CGO_ENABLED=0

WORKDIR /src
RUN mkdir -p /src/out/$GOOS/$GOARCH

COPY . /src/

RUN go build -trimpath -o /src/out/$TARGETOS/$TARGETARCH/exporter .

FROM alpine:latest

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder /src/out/$TARGETOS/$TARGETARCH/exporter /bin/exporter

EXPOSE 9090

ENTRYPOINT ["/bin/exporter"]
