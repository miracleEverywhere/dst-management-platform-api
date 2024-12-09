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

# 成功设置时区，但镜像增大50M
# RUN apt-get update && apt-get install -y tzdata
# ENV TZ=Asia/Shanghai
# RUN ln -sf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN chmod +x /root/dmp
RUN chmod +x /root/entry-point.sh
RUN chmod -R +x /root/config/

RUN mkdir /root/config
RUN mkdir /root/.klei


# web端口
EXPOSE 80/tcp
# 服务器端口
EXPOSE 10998/udp
EXPOSE 10999/udp
EXPOSE 11000/udp
EXPOSE 11001/udp

# 设置容器启动时执行的脚本
ENTRYPOINT ["/root/entry-point.sh"]  