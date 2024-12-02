FROM --platform=$BUILDPLATFORM golang:1.23 as build

ARG GOPROXY
ARG GOSUMDB
ARG GOPRIVATE
ARG TARGETARCH

WORKDIR /app

ENV GO111MODULE=on
# GOPROXY=https://goproxy.cn,direct

COPY . .

RUN go build -o dmp main.go

FROM ubuntu:24.04

WORKDIR /root

COPY --from=build /app/dmp  /root/dmp
COPY --from=build /app/docker/entry-point.sh /root/entry-point.sh

RUN chmod +x /root/dmp
RUN chmod +x /root/entry-point.sh
RUN mkdir /root/.klei

EXPOSE 80/tcp
EXPOSE 10889/udp
EXPOSE 11000/udp
EXPOSE 11001/udp

ENTRYPOINT ["/root/entry-point.sh"]
