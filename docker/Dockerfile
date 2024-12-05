FROM --platform=$BUILDPLATFORM golang:1.23 AS build

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

# web端口
EXPOSE 80/tcp
# 服务器端口
EXPOSE 10998/udp
EXPOSE 10999/udp
EXPOSE 11000/udp
EXPOSE 11001/udp

ENTRYPOINT ["/root/entry-point.sh"]