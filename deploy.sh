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

if [ ! -f "web/dist/index.html" ]; then
    echo "[ERROR] 未找到 web/dist/index.html。"
    echo "[ERROR] 当前 Dockerfile 不在镜像内执行 npm install/build，请先在服务器执行："
    echo "        cd web && npm install && npm run build && cd .."
    exit 1
fi

echo "[INFO] 正在构建并启动 Docker 容器..."
docker compose up -d --build

if [ $? -eq 0 ]; then
    echo ""
    echo "========================================================"
    echo "部署成功！"
    echo ""
    echo "服务已在后台运行："
    echo "- CDK 空投台: http://localhost:8088"
    echo ""
    echo "提示："
    echo "- 查看运行日志：docker compose logs -f miukey"
    echo "- 停止服务：docker compose down"
    echo "- 如需启用 Redis/RabbitMQ：docker compose --profile cache up -d --build"
    echo "========================================================"
else
    echo ""
    echo "[ERROR] 部署失败，请检查 Docker 是否正在运行以及端口是否被占用。"
fi
