# 第1阶段：构建应用程序
FROM --platform=$BUILDPLATFORM golang:1.23 AS build

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

RUN chmod +x /root/dmp
RUN chmod +x /root/entry-point.sh

RUN mkdir /root/config
RUN mkdir /root/.klei

# 环境变量
# - 平台暴露端口，默认为80
ENV DMP_PORT=${DMP_PORT:-80}

# web端口
EXPOSE 80/tcp
# 服务器端口
# 多服务器模式下Master节点master_port端口，默认为10888
EXPOSE 10888/udp
EXPOSE 10998/udp
EXPOSE 10999/udp
EXPOSE 11000/udp
EXPOSE 11001/udp
# steam验证端口，udp和tcp都需要。默认为27017、8767，多服务器配置模式下默认为27018、8768和27019、8769
EXPOSE 27017/tcp
EXPOSE 27017/udp
EXPOSE 8767/tcp
EXPOSE 8767/udp
EXPOSE 27018/tcp
EXPOSE 27018/udp
EXPOSE 8768/tcp
EXPOSE 8768/udp
EXPOSE 27019/tcp
EXPOSE 27019/udp
EXPOSE 8769/tcp
EXPOSE 8769/udp

# 设置容器启动时执行的脚本
ENTRYPOINT ["/root/entry-point.sh"]  