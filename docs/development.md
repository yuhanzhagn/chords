# 开发运行与排障

## 工具与依赖

在仓库根目录开始。Go 推荐预装 1.24.12；backend 声明 go 1.24.6，connection 声明 go 1.24.0 + toolchain go1.24.12，fanout 声明 go1.24.6 + toolchain go1.24.12。较旧 Go 可能通过 GOTOOLCHAIN=auto 下载工具链；无网络时先准备工具链和 module cache。

前端 CI 使用 Node 18，使用 npm 锁文件安装参数；这描述仓库现状，不是新的版本升级计划。Python 容器声明 3.11。还需要 GNU Make / Python 3 执行仓库辅助检查；Docker/Compose 仅在启动演示时需要。

```bash
make help
make check-docs
make install-frontend
make test-go
make test-frontend
make build-frontend
make check-python
```

测试不要求启动真实 Kafka/Redis。`make check` 汇总文档、语法、Go、Jest 与前端 build，前端依赖需事先安装；不会自动修依赖或启动服务。

## Compose 完整开发拓扑

**backend 入口会 FLUSHDB，务必使用隔离开发 Redis。** 命令从根目录执行：

```bash
make demo
```

该目标先构建 Linux 二进制到 backend/app、connection/connection、fanout/fanout，安装/构建前端，再执行前台 `docker compose up --build`。Compose 根 Dockerfile 是运行基础镜像，直接 compose up 无法替代预构建 Go 二进制。

frontend 容器额外执行 npm ci 并运行 CRA dev server；预构建的 frontend/build 不是当前 Compose 的静态服务内容。浏览器访问 localhost:8000；用日志确认 Kafka 消费 group 已 setup 后再发送消息。

```bash
docker compose ps
docker compose logs --tail=100 app connection fanout kafka
docker compose exec redis redis-cli PING
```

可用 `docker compose stop` 停止服务。运行 backend 会生成 SQLite；`fanout/fanout` 是历史已跟踪文件，演示构建后检查 diff，不要把二进制混入文档/业务提交。

## 单服务调试

先确认 [配置参考](configuration.md)。以下命令都要从各服务目录运行，且不是开箱即用的脱离基础设施模式：

```bash
cd backend
go mod download
go run ./cmd
```

```bash
cd connection
go mod download
go run ./cmd/server
```

```bash
cd fanout
go mod download
go run ./cmd/worker
```

默认配置要求 kafka/redis 服务名可以解析。gateway/worker 可修改 YAML 的地址及 advertise_addr；backend 的 Redis/Kafka 仍在 Go 中硬编码，不能只改 backend YAML。宿主机调试需匹配 DNS 与 Kafka advertised listener，或先实现可配置地址；通常优先采用 Compose。不要把此限制误诊为 Go 编译失败。

前端可在独立终端本地运行：

```bash
cp frontend/.env.example frontend/.env.local
cd frontend
npm ci --legacy-peer-deps
npm start
```

CRA 默认 localhost:3000，示例指向 localhost:8000/api；WS 同样依赖 8000 的 gateway 代理。改 env 后重启 CRA。Python 服务独立启动，见 [Python README](../python_ai_service/README.md)。

## 冒烟验证

在演示已启动后，浏览器注册、登录、创建房间、加入永久成员关系、选择房间并发送消息；刷新后确认历史可读。另开用户检查实时收取和重连，当前已知前端缺口见 [现状](status.md)，不能只凭自己的 pending 被确认判断完整聊天成功。

低负载 WS 工具（会写入消息）：

```bash
cd connection
go run ./cmd/wsload -url ws://localhost:8000/ws -users 2 -room 1 -messages 1
```

只对隔离测试环境使用。工具不会注册账户/创建房间，也没有可靠性断言；统计不能代替端到端测试。

## 常见故障

| 现象 | 排查入口 |
| --- | --- |
| 根目录 go test 提示无 module | 改用 make test-go 或进入具体模块 |
| 工具链下载/代理失败 | 各模块 go.mod、GOTOOLCHAIN、预装版本和本地缓存 |
| Redis/Kafka 无法解析 | 服务名是 Docker 网络地址；核对 host/container 网络和 advertised listener |
| JWT 登录成功但 WS 无法按预期鉴权 | WS 认证尚未接线，前端没传 token；不是 REST JWT 已保证 WS |
| fetch URL 出现双 http:// | REACT_APP_URL 应不带协议；查所有直接 fetch |
| WS 连上但收不到消息 | 按 gateway join/group → Redis → user-request → backend → notification → fanout → /fanout → UI 逐跳查日志 |
| pending/其他用户消息或重连异常 | useMessages CONFIRM、useChatSocket 重入逻辑，见状态待办 |
| Jest 缺 protobuf 模块 | 先按 lock 执行 npm ci --legacy-peer-deps，排除陈旧 node_modules |
| kind API 可用但实时不回流 | 清单缺 fanout 与 8082 Service，当前不是完整拓扑 |
| WSL 的 docker 只有提示无集成 | 需要可用 Docker Engine/WSL 集成；不影响纯模块测试 |
