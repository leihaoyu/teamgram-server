FROM golang:1.19 AS builder
WORKDIR /app
COPY . .
RUN ./build.sh

FROM ubuntu:latest
WORKDIR /app
COPY --from=builder /app/teamgramd/ /app/
RUN apt update -y && apt install -y ffmpeg curl && chmod +x /app/docker/entrypoint.sh \
    && curl -fSL -o /app/bin/GeoLite2-City.mmdb \
       https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb
ENTRYPOINT /app/docker/entrypoint.sh
