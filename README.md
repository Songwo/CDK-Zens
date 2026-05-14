# 缪盒空投台 (MiuBox Airdrop Hub)

> **缪盒空投台** 是一款面向 CDK、兑换码、节点分发和空投领取的运营平台。采用 **Go** + **React** 构建，可选集成 **Redis** 与 **RabbitMQ**。

---

## ✨ 核心特性

- **🛡️ 企业级并发防护**：基于 **Redis** 原子扣减库存，搭配 **RabbitMQ** 异步持久化任务队列，轻松应对瞬时万级流量。
- **🔐 身份断言系统**：内置完整的登录注册流程，支持自定义管理员权限与**社区快捷单点登录 (SSO)**。
- **⏳ 精准调度**：支持设置“即将开启”项目，内置动态倒计时组件，时间一到自动解锁分发通道。
- **🎨 极简未来视觉**：摒弃传统卡片设计，采用全屏化、流式设计语言。标志性的 **Zens-Yellow** 品牌色，搭配流畅的微交互动画。
- **🔌 插件化存储**：默认支持轻量级 JSON 文件持久化，可无缝降级运行（无需外部依赖）。
- **📊 审计追踪**：实时审计日志记录每一个节点的领取动态、IP 寻址及终端指纹。

---

## 🏗️ 技术架构

- **后端**: Go 1.22+ (原生 http 路由优化 / slog 日志 / JWT 鉴权)
- **前端**: React 18 / Vite 7 / Vanilla CSS (极简主义)
- **高性能层**: 
  - **Redis 7**: 原子库存控制器
  - **RabbitMQ**: 异步持久化消费者
- **容器化**: Docker / Docker Compose

---

## 🚀 快速启动

### 方式一：服务器 Docker 部署

当前 Dockerfile 不在镜像内执行 `npm install`，需要先在服务器生成前端产物，再构建镜像：

```bash
cd web
npm install
npm run build
cd ..

chmod +x deploy.sh
./deploy.sh
```

Docker Compose 会自动启动 PostgreSQL，默认访问地址为 `http://服务器IP:8088/`。首次部署无需手动创建数据库。

```text
POSTGRES_DB=cdk_airdrop
POSTGRES_USER=cdk_airdrop
POSTGRES_PASSWORD=cdk_airdrop_change_me
CDK_POSTGRES_DSN=postgres://cdk_airdrop:cdk_airdrop_change_me@postgres:5432/cdk_airdrop?sslmode=disable
```

如果修改了 `POSTGRES_USER` 或 `POSTGRES_PASSWORD`，请同步修改 `CDK_POSTGRES_DSN` 中的用户名和密码。旧的 `CDK_MYSQL_DSN` 仅保留兼容，不再作为 Docker 默认存储。

PostgreSQL 数据保存在 Docker volume `postgres_data` 中；普通 `docker compose down` 不会删除数据。

Redis / RabbitMQ 为可选增强组件，默认不启动。如需启用：

```bash
docker compose --profile cache up -d --build
```

### 方式二：本地开发环境

1. **启动后端**:
   ```bash
   cd server
   go run main.go
   ```

2. **启动前端**:
   ```bash
   cd web
   npm install
   npm run dev
   ```

---

## ⚙️ 环境变量配置

| 变量名 | 说明 | 示例 |
| --- | --- | --- |
| `CDK_AIRDROP_ADDR` | 服务监听地址 | `:8088` |
| `CDK_POSTGRES_DSN` | PostgreSQL 连接串 | `postgres://cdk_airdrop:password@postgres:5432/cdk_airdrop?sslmode=disable` |
| `CDK_MYSQL_DSN` | 旧 MySQL 连接串，仅兼容 | `user:pass@tcp(127.0.0.1:3306)/cdk_airdrop?parseTime=true&charset=utf8mb4&loc=Local` |
| `CDK_AIRDROP_REDIS_URL` | Redis 连接串，可留空 | `redis://127.0.0.1:6379/0` |
| `CDK_AIRDROP_RABBITMQ_URL` | RabbitMQ 连接串，可留空 | `amqp://guest:guest@127.0.0.1:5672/` |
| `CDK_AIRDROP_DATA_FILE` | 状态存储路径 | `./data/state.json` |

---

## 📂 项目结构

```text
cdk-airdrop-station/
├── server/             # Go 后端核心逻辑
│   ├── internal/api    # 接口处理器 (Handler)
│   ├── internal/store  # 存储驱动 (JSON/Redis/MQ)
│   └── main.go         # 入口文件
├── web/                # React 前端工程
│   ├── src/pages       # 管理端与领取端页面
│   ├── src/layouts     # 全域布局组件
│   └── index.css       # 核心 MiuBox 设计系统
├── Dockerfile          # 多阶段构建文件
└── docker-compose.yml  # 编排服务
```

---

## 🤝 许可证

本项目基于 MIT 协议开源。

---

## hCaptcha 本地开发

前端测试 sitekey：

```text
10000000-ffff-ffff-ffff-000000000001
```

后端测试 secret：

```text
0x0000000000000000000000000000000000000000
```

生产环境不要使用测试密钥。

建议先用 GET 验证 JS 资源可访问：

```bash
curl -L --connect-timeout 8 -o /dev/null -w "%{http_code} %{time_total}\n" https://js.hcaptcha.com/1/api.js
```

如果返回 `200`，说明 JS 资源访问正常。`curl -I` 返回 `405` 不代表浏览器 GET 加载失败。

本地建议绑定测试域名：

```text
127.0.0.1 test.local-miubox.com
```

然后访问：

```text
http://test.local-miubox.com:5173
```

并在 hCaptcha 控制台添加 `test.local-miubox.com` 作为测试域名。
