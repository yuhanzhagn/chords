# Fanout worker

遵循 [根指南](../AGENTS.md)。独立 Go module `fanout`。

- `cmd/worker/main.go`：读取配置，建立 Redis/HTTP client/dispatcher/Kafka group。
- `internal/kafka/consumer.go`：protobuf notification，默认 group fanout-workers、初始 offset newest。
- `internal/registry/redis_registry.go`：SMEMBERS room users，再 MGET user gateway。
- `internal/fanout/dispatcher.go`：按网关聚合并逐个 POST /fanout，默认单请求超时 3 秒。
- `internal/app/config.go` 与 `configs/config.yaml`：配置来源。

handler 错误被记录后仍 MarkMessage，没有持久重试、死信或投递幂等保证。修改提交/重试语义须评估重复与丢失。Redis key 规则必须与 connection 一致；room_id 非零时网关按本地房间广播，user_ids 不是当前路径的严格过滤器。

```bash
cd fanout
go test ./...
go build -o /tmp/chords-fanout ./cmd/worker
```

目前没有 `_test.go`，go test 只提供编译检查；新增行为需补实质测试。`fanout/fanout` 是已跟踪历史二进制，普通构建使用临时路径；根 make fanout 供 Compose 演示使用。缺口见 [现状](../docs/status.md)。
