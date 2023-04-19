FROM alpine:3.13.5

# 时区设置(TZ=Asia/Shanghai)
RUN echo -e "http://mirrors.aliyun.com/alpine/v3.13/main\nhttp://mirrors.aliyun.com/alpine/v3.13/community" >/etc/apk/repositories && \
    apk update && \
    apk add --no-cache tzdata && \
    apk add --no-cache libc6-compat && \
    apk add --no-cache tini

COPY ./bin/tcp_svr /go/bin/

ENV GOTRACEBACK=crash

WORKDIR /go/logs

ENTRYPOINT ["/sbin/tini", "--", "../bin/tcp_svr"]
