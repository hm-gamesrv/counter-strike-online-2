# ===================
# 编译
# ===================
FROM golang:1.27-alpine AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# -tags timetzdata：嵌入时区数据库（alpine 无 /usr/share/zoneinfo）
# -ldflags '-s -w'：去掉符号表，减小体积
RUN go build -tags timetzdata -ldflags '-s -w' -trimpath -o /build/cso2-server .

# ===================
# 主镜像
# ===================
FROM alpine:3

ENV TZ=Asia/Shanghai

RUN apk add --no-cache tzdata ca-certificates

RUN addgroup -g 1000 gamesrv \
    && adduser -u 1000 -D -G gamesrv -s /bin/sh gamesrv
RUN mkdir -p /app && chown 1000:1000 /app \
    && mkdir -p /data && chown 1000:1000 /data

COPY --from=builder --chown=1000:1000 /build/cso2-server /app

EXPOSE 1315/tcp 30001/tcp 30002/udp

VOLUME ["/data"]

WORKDIR /app
USER 1000:1000
CMD ["/app/cso2-server"]
