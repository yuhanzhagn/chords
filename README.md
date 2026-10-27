# Chords

实时聊天室 monorepo：三个独立 Go 模块（REST backend、WebSocket connection、fanout worker）、React 前端，以及独立 Python TF-IDF 检索服务。

支持注册/登录、聊天室成员关系、消息持久化、历史分页与实时分发。当前处于开发阶段；认证接线、投递可靠性与部署完整性仍有缺口，见 [现状与待办](docs/status.md)。

## 开始阅读

| 目的 | 文档 |
| --- | --- |
| Agent 接手任务 | [AGENTS.md](AGENTS.md)，再读目标目录指南 |
| 理解系统 | [架构](docs/architecture.md)、[协议](docs/contracts.md) |
| 安装与运行 | [开发运行](docs/development.md)、[配置](docs/configuration.md) |
| 验证与协作 | [测试](docs/testing.md)、[贡献](CONTRIBUTING.md)、[Agent 流程](docs/agent-workflow.md) |
| 浏览全部文档 | [文档索引](docs/README.md) |

## 仓库地图

| 目录 | 职责与入口 |
| --- | --- |
| [backend](backend/AGENTS.md) | Go / Gin / GORM；REST、JWT/session、SQLite、Kafka 处理；`cmd/main.go` |
| [connection](connection/AGENTS.md) | Go / gorilla/websocket；连接、订阅、在线注册、pipeline；`cmd/server/main.go` |
| [fanout](fanout/AGENTS.md) | Go / Sarama；消费 notification、查询 Redis、HTTP 推送；`cmd/worker/main.go` |
| [frontend](frontend/AGENTS.md) | React 19 / TypeScript / CRA；`src/App.tsx` |
| [proto](proto/AGENTS.md) | 共享 `kafka/event.proto` 与生成约定 |
| [python_ai_service](python_ai_service/README.md) | 独立 FastAPI / TF-IDF 检索与抽取式回答；`app/main.py` |
| [deploy](deploy/AGENTS.md) | kind/Kubernetes 开发实验，仅 backend、connection、Redis、Kafka |

实时消息主路径：`browser → connection → Kafka(user-request) → backend → Kafka(notification) → fanout → connection → browser`。REST 创建消息当前只写数据库。

## 快速检查

需要 GNU Make、Go（推荐预装 1.24.12，精确要求见各 `go.mod`）、Node/npm 和 Python 3。前端 CI 声明 Node 18。

```bash
make help
make check-docs
make test-go
make install-frontend       # npm ci --legacy-peer-deps
make test-frontend
make build-frontend
make check-python
```

根目录没有 `go.mod`，不能在这里直接 `go test ./...`。Python 检查仅验证语法，服务依赖与启动见独立 README。

## 完整开发演示

**仅使用隔离开发环境：backend 每次启动都会清空连接到的 Redis DB。**

```bash
make demo
```

该目标构建 Linux Go 二进制、安装/构建前端，再以前台方式启动 Docker Compose。需要 Docker Engine、Compose 和 Linux 构建环境。

浏览器入口为 [localhost:8000](http://localhost:8000)；Traefik 转发 `/api`、`/ws` 和 `/`。gateway `/fanout` 在容器内部监听 8082。Redis/Kafka 映射宿主机 6379/9092，但 Kafka 广播地址仍为 `kafka:9092`。

Compose 未包含 Python 服务；kind 清单尚未覆盖完整聊天链路。详细步骤与排障见 [开发运行](docs/development.md)，本次实际验证记录见 [测试说明](docs/testing.md)。
