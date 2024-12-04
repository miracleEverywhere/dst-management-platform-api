FROM --platform=$BUILDPLATFORM golang:1.23 AS build

ARG GOPROXY
ARG GOSUMDB
ARG GOPRIVATE
ARG TARGETARCH

WORKDIR /go_build

ENV GO111MODULE=on
# GOPROXY=https://goproxy.cn,direct

# 更新 apt 包列表
RUN apt-get update
# 安装 wget
RUN apt-get install -y wget
# 清理 apt 缓存
RUN apt-get clean

COPY . .

RUN go build -o dmp main.go

FROM ubuntu:24.04

# 创建 app 用户和组
RUN groupadd -r app && useradd -r -g app -m -s /bin/bash app

# 设置工作目录为 app 用户的家目录
WORKDIR /home/app

# 复制可执行文件和入口脚本到 app 用户的家目录
COPY --from=build /go_build/dmp /home/app/dmp
COPY --from=build /go_build/docker/entry-point.sh /home/app/entry-point.sh

# 更改文件权限为 app 用户所有
RUN chown -R app:app /home/app

# 设置文件权限
RUN chmod +x /home/app/dmp
RUN chmod +x /home/app/entry-point.sh

RUN mkdir /home/app/.klei

# 切换到 app 用户
USER app

# web端口
EXPOSE 80/tcp
# 服务器端口
EXPOSE 10998/udp
EXPOSE 10999/udp
EXPOSE 11000/udp
EXPOSE 11001/udp

ENTRYPOINT ["/home/app/entry-point.sh"]