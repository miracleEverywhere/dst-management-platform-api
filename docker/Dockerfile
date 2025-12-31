# 第1阶段：构建应用程序
FROM --platform=$BUILDPLATFORM golang:1.25 AS build

ARG GOPROXY
ARG GOSUMDB
ARG GOPRIVATE
ARG TARGETARCH

WORKDIR /app

# 设置GO111MODULE环境变量为on，启用Go模块支持 (此行注释掉，可能已默认启用)
# ENV GO111MODULE=on  

# 复制当前目录下的所有文件到容器的/app目录
COPY . .  

RUN go build -o dmp main.go 


# 第2阶段：运行时环境
FROM ubuntu:24.04

WORKDIR /root

# 从构建阶段复制必要的文件
COPY --from=build /app/dmp /root/dmp
COPY --from=build /app/docker/entry-point.sh /root/entry-point.sh

RUN apt-get update && apt-get install -y screen wget \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && chmod +x /root/dmp /root/entry-point.sh \
    && mkdir -p /root/data /root/.klei

# 环境变量
# - 平台暴露端口，默认为80
ENV DMP_PORT=${DMP_PORT:-80}
# 是否以容器方式启动
ENV DMP_IN_CONTAINER=1

# web端口
EXPOSE 80/tcp

# 设置容器启动时执行的脚本
ENTRYPOINT ["/root/entry-point.sh"]  
