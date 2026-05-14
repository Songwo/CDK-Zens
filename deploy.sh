#!/bin/bash
echo "========================================================"
echo "        CDK 空投台 - 一键 Docker 部署脚本"
echo "========================================================"

# 检查 .env 文件是否存在
if [ ! -f ".env" ]; then
    echo "[INFO] 未找到 .env 文件，正在从 .env.example 复制..."
    cp .env.example .env
    echo "[INFO] 已创建默认 .env 文件，请按需修改。"
fi

ensure_env() {
    local key="$1"
    local value="$2"
    if ! grep -q "^${key}=" .env; then
        printf "\n%s=%s\n" "$key" "$value" >> .env
    fi
}

ensure_env "POSTGRES_DB" "cdk_airdrop"
ensure_env "POSTGRES_USER" "cdk_airdrop"
ensure_env "POSTGRES_PASSWORD" "cdk_airdrop_change_me"
ensure_env "CDK_POSTGRES_DSN" "postgres://cdk_airdrop:cdk_airdrop_change_me@postgres:5432/cdk_airdrop?sslmode=disable"

if [ ! -f "web/dist/index.html" ]; then
    echo "[ERROR] 未找到 web/dist/index.html。"
    echo "[ERROR] 当前 Dockerfile 不在镜像内执行 npm install/build，请先在服务器执行："
    echo "        cd web && npm install && npm run build && cd .."
    exit 1
fi

echo "[INFO] 正在构建并启动 Docker 容器..."
docker compose up -d --build

if [ $? -eq 0 ]; then
    sleep 2
    CONTAINER_ID=$(docker compose ps -q miukey 2>/dev/null)
    STATUS=$(docker inspect -f '{{.State.Status}}' "$CONTAINER_ID" 2>/dev/null)

    if [ "$STATUS" != "running" ]; then
        echo ""
        echo "[ERROR] 容器未正常运行，当前状态：${STATUS:-unknown}"
        echo "[ERROR] 最近日志如下："
        docker compose logs --tail=120 miukey postgres
        echo ""
        echo "常见原因："
        echo "- .env 中 CDK_POSTGRES_DSN 与 POSTGRES_USER/POSTGRES_PASSWORD 不一致。"
        echo "- ./server/data/state.json 不是合法 JSON，或 ./server/data 不可写。"
        echo "- 8088 端口已被其他进程占用。"
        exit 1
    fi

    echo ""
    echo "========================================================"
    echo "部署成功！"
    echo ""
    echo "服务已在后台运行："
    echo "- CDK 空投台: http://localhost:8088"
    echo ""
    echo "提示："
    echo "- 查看运行日志：docker compose logs -f miukey postgres"
    echo "- 停止服务：docker compose down"
    echo "- 如需启用 Redis/RabbitMQ：docker compose --profile cache up -d --build"
    echo "========================================================"
else
    echo ""
    echo "[ERROR] 部署失败，请检查 Docker 是否正在运行以及端口是否被占用。"
fi
