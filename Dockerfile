FROM golang:1.19 AS builder
WORKDIR /app

# ✅ 关键：先拷贝依赖文件
COPY go.mod go.sum ./

# ✅ 先下载依赖（缓存层）
RUN go mod download

# ✅ 再拷贝代码
COPY . .

# ✅ 编译
RUN ./build.sh

FROM ubuntu:latest
WORKDIR /app
COPY --from=builder /app/teamgramd/ /app/
COPY --from=builder /app/data/langpack/ /app/data/langpack/
RUN apt update -y && apt install -y ffmpeg curl && chmod +x /app/docker/entrypoint.sh \
    && curl -fSL -o /app/bin/GeoLite2-City.mmdb \
       https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb
ENTRYPOINT /app/docker/entrypoint.sh
