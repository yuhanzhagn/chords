# Agent 工作指南

Chords 是实时聊天室 monorepo：三个独立 Go module、React SPA，以及独立的 Python TF-IDF 检索服务。先读本文件，再读目标目录的 `AGENTS.md`；只加载与当前任务相关的文档。

## 定位与入口

| 任务 | 起点 | 补充文档 |
| --- | --- | --- |
| REST、登录、持久化 | `backend/internal/routes/routes.go` → controller → service → repo | [backend 指南](backend/AGENTS.md) |
| WS、连接生命周期、限流 | `connection/cmd/server/main.go`、`connection/internal/gateway/` | [connection 指南](connection/AGENTS.md) |
| Kafka 出站、Redis 路由、HTTP 分发 | `fanout/internal/fanout/dispatcher.go` | [fanout 指南](fanout/AGENTS.md) |
| 页面、消息状态、API/WS 客户端 | `frontend/src/App.tsx`、`frontend/src/hooks/` | [frontend 指南](frontend/AGENTS.md) |
| 事件 schema、生成代码 | `proto/kafka/event.proto` | [协议说明](docs/contracts.md)、[proto 指南](proto/AGENTS.md) |
| 搜索与抽取回答 | `python_ai_service/app/` | [Python 指南](python_ai_service/AGENTS.md) |
| 容器、CI、集群 | `docker-compose.yml`、`.github/workflows/`、`deploy/k8s/` | [开发运行](docs/development.md)、[部署指南](deploy/AGENTS.md) |

全貌见 [架构](docs/architecture.md)；已确认的缺口见 [现状与待办](docs/status.md)。文档描述当前实现，未来设计会明确标注。

## 工作约定

1. 先看 `git status --short`，保留已有修改；阅读入口及邻近测试后再改。修复应能关联到具体行为和验收方式。
2. 根目录没有 `go.mod`；在对应 Go module 内运行命令。保持三份 `go.mod/go.sum` 独立，不为普通任务添加 `go.work` 或合并模块。
3. Go 保持现有边界与接口注入；前端保持 React/CRA、TypeScript 和现有 UI 风格。不要顺手升级框架、依赖或批量格式化无关文件。
4. `proto/kafka/event.proto` 是事件定义源；不要手改 `*.pb.go` 和 `frontend/src/proto/kafka/event.ts`。跨服务协议变更必须检查四个消费者并重新生成。
5. 配置、API、消息或启动方式变化时，同步更新对应文档及示例。记录真实执行结果；无法执行、无测试与通过测试是不同状态。
6. 不提交二进制、SQLite、Python 数据、虚拟环境、`node_modules`、构建产物、真实凭据。已有被跟踪的 `fanout/fanout` 是历史产物；不要在普通构建中覆盖它。

## 验证入口

```bash
make help                 # 所有可用目标
make check-docs           # 本地 Markdown 链接存在性；只需 Python 3
make test-go              # 三个 Go module 的 go test ./...
make test-frontend        # Jest 非交互模式；先 make install-frontend
make build-frontend       # CRA 生产构建
make check-python         # Python 源码语法检查，不代表服务运行成功
```

按变更范围选择验证，详情见 [测试说明](docs/testing.md)。业务修复补充有意义的回归测试；纯文档变更不要求启动整套服务。不要默认执行安装、`go mod tidy`、代码生成、部署或压测。

## 开始运行前必须了解

- backend 启动不清 Redis；启动前必须提供至少 32 字节的 `JWT_SECRET`。常规测试使用隔离测试密钥，无需启动服务。
- WS 首帧认证由 backend 校验 session，身份固定；历史、发送、订阅仅成员可用。Kafka 入站需短效命令授权 header；协议与未决权限见 `docs/contracts.md`、ADR。
- 默认 Kafka/Redis 地址是 Docker 服务名 `kafka`/`redis`；backend broker 可用 `CHORDS_KAFKA_BROKERS` 覆盖，其他模块见配置。宿主机 `go run` 不会自动适配。
- 消息 schema v2 使用永久请求墓碑/room_seq/Outbox relay；默认 WS 同步接纳后才 ACK，旧 Kafka 命令消费默认关闭。迁移仅临时库验证，真实升级/回退见 `docs/message-persistence.md`。
- 前端 `REACT_APP_URL` 取 `localhost:8000/api`（不带协议），代码会拼接 `http://`；WS 地址另行硬编码。
- Kubernetes 清单缺少 fanout；Go 测试通过不代表部署具备完整聊天链路。

## 完成任务

确认变更与任务一致、相关检查已执行、文档与实现一致。交付说明包括行为变化、验证命令/结果、剩余限制；后续任务交接按 [协作流程](docs/agent-workflow.md) 提供文件入口与验收条件。
