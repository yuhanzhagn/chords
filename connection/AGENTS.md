# Connection gateway

遵循 [根指南](../AGENTS.md)。独立 Go module `connection`。

## 实际调用链

`cmd/server/main.go` 装配 codec、Hub、Redis registry、async/retry Kafka sink、WS mux 和 fanout HTTP server。

- `/ws`：全局建连限流 → upgrade → 5 秒内首帧认证（未认证不注册）→ ready → readPump 解码/每连接限流 → 每帧 backend 授权 → 房间/在线注册 → 仅 message 进入 sink。
- join/leave 更新本地 group 与 Redis；当前不发送到 Kafka。
- `/fanout`：`internal/source/http_source.go` → 专用服务凭据校验 → codec → room 广播或 client ID 定向发送；用户/session/成员投递检查仍执行。
- `internal/gateway/`：Hub、MemoryStore、pumps、send channel、断线处理。
- `internal/handler/`、`internal/sink/`、`internal/adapter/kafka/`：pipeline、async/retry/multi、Kafka 编码发布。
- `internal/app/config.go`：YAML 默认值、presence TTL/刷新间隔。`cmd/wsload/` 会生成真实消息，仅用于开发环境。

## 关键约束

- gateway/worker 必须分别从环境读取同一 `FANOUT_TOKEN`（至少 32 字节、无空格的可见 ASCII，独立随机生成，与 JWT_SECRET 分离）；无默认值、不从 YAML 读取。缺失/非法在连接真实依赖前启动失败。`X-Chords-Fanout-Token` 在读 body 前校验，不能用 access/command JWT 代替。配置不可整体打印；不记录秘密。跨不可信网络需可信 TLS；共享凭据本身不防窃听/重放。

- `cmd/server/auth.go` 调用 backend `/api/auth/ws-authorize`；`auth.backend_url` 默认 `http://app:8080`，可用 `CHORDS_AUTH_BACKEND_URL` 覆盖。连接身份在注册前固定，事件 UserId 只允许 0 或本人；每帧、每次投递重新校验，空闲每 5 秒检查 session。
- Kafka sink 从 context 取得短效授权，写 `chords-command` header；不转发长效 access token，不改共享 protobuf。无效凭据/权限按 1008 关闭；认证服务临时失败按 1013 关闭，允许重连。生产 mux 必须提供 Authenticator。
- `Client.ID` 是连接 ID，`Client.UserID` 是业务用户。一个用户可有多连接，不能在其中一条关闭时清除其他连接仍使用的 presence。
- 保留限长、ping/pong、read/write deadlines、关闭 channel 发送保护与最后连接清理；并发/生命周期改动覆盖这些场景。
- Redis room users 是在线路由集合，与 SQLite 永久成员关系不同。
- advertise_addr 必须指向 worker 可访问的具体网关；多副本共享负载均衡地址不能保证投递给连接所属实例。
- `/fanout` 非零 room_id 优先于 user_ids；fallback 实际按 client ID 查找。见 [协议](../docs/contracts.md)。

```bash
cd connection
go test ./...
# 根据改动范围：
go test ./cmd/server ./internal/gateway ./internal/source
go test -race ./cmd/server ./internal/gateway ./internal/sink
```

测试无需真实 Kafka/Redis；主程序需要。辅助 Kafka source、MessageService 或 middleware 的存在不代表已由 main 接线，先从入口确认。
