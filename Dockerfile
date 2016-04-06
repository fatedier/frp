FROM golang:1.5

MAINTAINER fatedier

RUN echo "[common]\nbind_addr = 0.0.0.0\nbind_port = 7000\n[test]\npasswd = 123\nbind_addr = 0.0.0.0\nlisten_port = 80" > /usr/share/frps.ini

ADD ./ /usr/share/frp/

RUN cd /usr/share/frp && make

EXPOSE 80
EXPOSE 7000

CMD ["/usr/share/frp/bin/frps", "-c", "/usr/share/frps.ini"]
