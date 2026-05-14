# --- Build Frontend ---
FROM node:20-alpine AS frontend-builder
WORKDIR /web
# 开启 npm 腾讯云内网/公网镜像加速 (解决网络重置问题)
RUN npm config set registry http://mirrors.cloud.tencent.com/npm/
ARG VITE_HCAPTCHA_SITE_KEY
ENV VITE_HCAPTCHA_SITE_KEY=$VITE_HCAPTCHA_SITE_KEY
COPY web/package*.json ./
RUN npm install
COPY web/ .
RUN npm run build

# --- Build Backend ---
FROM golang:1.24-alpine AS backend-builder
WORKDIR /server
# 开启 Go 国内镜像加速 (使用阿里云镜像更稳定)
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ .
RUN go build -o miubox main.go

# --- Final Image ---
FROM alpine:latest
# 替换为腾讯云 Alpine 镜像源
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tencent.com/g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binaries and assets
COPY --from=backend-builder /server/miubox .
COPY --from=frontend-builder /web/dist ./dist

# Create data directory
RUN mkdir -p /app/data

# Environment variables
ENV CDK_AIRDROP_ADDR=":8088"
ENV CDK_AIRDROP_DATA_FILE="/app/data/state.json"
ENV CDK_AIRDROP_PUBLIC_DIR="/app/dist"

EXPOSE 8088

CMD ["./miubox"]
