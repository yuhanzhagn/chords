# Connection gateway

遵循 [根指南](../AGENTS.md)。独立 Go module `connection`。

## 实际调用链

`cmd/server/main.go` 装配 codec、Hub、Redis registry、async/retry Kafka sink、WS mux 和 fanout HTTP server。

- `/ws`：全局建连限流 → upgrade → readPump 解码 → 每连接限流 → 房间/在线注册 → 仅 message 进入 sink。
- join/leave 更新本地 group 与 Redis；当前不发送到 Kafka。
- `/fanout`：`internal/source/http_source.go` → codec → room 广播或 client ID 定向发送。
- `internal/gateway/`：Hub、MemoryStore、pumps、send channel、断线处理。
- `internal/handler/`、`internal/sink/`、`internal/adapter/kafka/`：pipeline、async/retry/multi、Kafka 编码发布。
- `internal/app/config.go`：YAML 默认值、presence TTL/刷新间隔。`cmd/wsload/` 会生成真实消息，仅用于开发环境。

## 关键约束

- `jwtMiddleware` 参数未加入实际 handler chain；身份取自事件 UserId。认证变更要联查 backend claims 和前端 token 传递。
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
