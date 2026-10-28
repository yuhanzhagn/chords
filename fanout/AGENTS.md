# Fanout worker

遵循 [根指南](../AGENTS.md)。独立 Go module `fanout`。

- `cmd/worker/main.go`：读取配置，建立 Redis/HTTP client/dispatcher/Kafka group。
- `internal/kafka/consumer.go`：protobuf notification，默认 group fanout-workers、初始 offset newest。
- `internal/registry/redis_registry.go`：SMEMBERS room users，再 MGET user gateway。
- `internal/fanout/dispatcher.go`：按网关聚合并逐个 POST /fanout，默认单请求超时 3 秒；附 `X-Chords-Fanout-Token`，禁止所有重定向，只将 2xx 视为成功。
- `internal/app/config.go` 与 `configs/config.yaml`：配置来源。

gateway/worker 必须通过环境分别注入相同的 `FANOUT_TOKEN`，至少 32 字节、无空格的可见 ASCII，独立随机生成且与 JWT_SECRET 分离；无默认值，YAML 不承载秘密，缺失/非法配置在 Redis/Kafka 前失败。构造器也验证配置。禁止整体打印配置或记录对端响应 body（可能回显秘密）。跨不可信网络需可信 TLS，共享凭据不能防明文窃听/重放。

handler 错误被记录后仍 MarkMessage，没有持久重试、死信或投递幂等保证。修改提交/重试语义须评估重复与丢失。Redis key 规则必须与 connection 一致；room_id 非零时网关按本地房间广播，user_ids 不是当前路径的严格过滤器。

```bash
cd fanout
go test ./...
go build -o /tmp/chords-fanout ./cmd/worker
```

已有 standard-library HTTP/mock registry、配置与真实入口失败顺序行为测试；不连接真实 Kafka/Redis。`fanout/fanout` 是已跟踪历史二进制，普通构建使用临时路径；根 make fanout 供 Compose 演示使用。缺口见 [现状](../docs/status.md)。
