@echo off
chcp 65001 > nul
echo ========================================================
echo         CDK 空投台 - 一键 Docker 部署脚本
echo ========================================================

:: 检查 .env 文件是否存在
if not exist ".env" (
    echo [INFO] 未找到 .env 文件，正在从 .env.example 复制...
    copy .env.example .env
    echo [INFO] 已创建默认 .env 文件，请按需修改。
)

echo [INFO] 正在构建并启动 Docker 容器...
docker-compose up -d --build

if %ERRORLEVEL% equ 0 (
    echo.
    echo ========================================================
    echo 部署成功！
    echo.
    echo 服务已在后台运行：
    echo - CDK 空投台: http://localhost:8088
    echo - Redis: localhost:6379
    echo - RabbitMQ: localhost:5672 (UI: 15672)
    echo.
    echo 提示：
    echo - 查看运行日志：docker-compose logs -f
    echo - 停止服务：docker-compose down
    echo ========================================================
) else (
    echo.
    echo [ERROR] 部署失败，请检查 Docker 是否正在运行以及端口是否被占用。
)
pause
