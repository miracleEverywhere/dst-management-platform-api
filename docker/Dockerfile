# 第1阶段：构建应用程序
FROM --platform=$BUILDPLATFORM golang:1.25 AS build

WORKDIR /app

# 复制当前目录下的所有文件到容器的/app目录
COPY . .  

RUN go build -o dmp main.go 


# 第2阶段：运行时环境
FROM ubuntu:24.04

WORKDIR /root

ENV HOME=/root

# 从构建阶段复制必要的文件
COPY --from=build /app/dmp /root/dmp
COPY --from=build /app/docker/entry-point.sh /root/entry-point.sh
COPY --from=build /app/embedFS/luajit /root/dmp_files/luajit
COPY --from=build /app/embedFS/shell/manual_install.sh /root/manual_install.sh

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        screen \
        wget \
        lib32gcc-s1 \
        libcurl4:i386 \
        libc6:i386 \
        libstdc++6:i386 \
    && chmod +x /root/dmp /root/entry-point.sh /root/manual_install.sh \
    && DMP_IN_CONTAINER=1 HOME=/root bash /root/manual_install.sh \
    && rm -f /root/manual_install.sh \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /root/data /root/.klei

# 环境变量
# - 平台暴露端口，默认为80
ARG DMP_PORT=80
ENV DMP_PORT=${DMP_PORT}
# 是否以容器方式启动
ENV DMP_IN_CONTAINER=1

# web端口
EXPOSE 80/tcp

# 设置容器启动时执行的脚本
ENTRYPOINT ["/root/entry-point.sh"]  
