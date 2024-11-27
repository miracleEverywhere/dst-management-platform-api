FROM ubuntu:24.04

RUN dpkg --add-architecture i386 && apt update && apt install -y lib32gcc1 && install -y lib32gcc-s1 && apt install -y libcurl4-gnutls-dev:i386 && apt install -y screen && apt install -y unzip

WORKDIR /root

COPY dmp /root/dmp
COPY entry-point.sh /root/entry-point.sh

RUN chmod +x /root/dmp
RUN chmod +x /root/entry-point.sh

EXPOSE 80/tcp
EXPOSE 10889/udp
EXPOSE 11000/tcp
EXPOSE 11001/tcp

ENTRYPOINT ["/root/entry-point.sh"]