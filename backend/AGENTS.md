# Backend

遵循 [根指南](../AGENTS.md)。独立 Go module `backend`。

## 定位

- `cmd/main.go`：密钥校验、SQLite 初始化、Redis 连接（不清空）、HTTP 启动（固定 `:8080`）。
- `internal/routes/routes.go`：依赖装配、REST 路由；SetupRouter 不连接 Kafka，message_runtime.go 独立启动 relay/可选旧 consumer。
- `internal/controller/` → `internal/service/` → `internal/repo/`：JSON/状态码 → 业务接口 → GORM/RepoContainer；接口修改同步 mocks。
- `internal/model/`、`internal/app/db.go`、`message_schema.go`：旧模型迁移与显式消息 schema v2（0→1→2）；先拒绝未来版本，旧消息不创建请求/outbox。
- `internal/service/send_message.go`：REST/Kafka 共用发送事务、幂等/墓碑与 Outbox 快照；`kafka_service.go` 只适配内容绑定的命令。
- `internal/middleware/jwtauth/session_auth.go`：路由实际使用的认证；相邻 `jwtauth.go` 的另一套实现未接入当前 router。
- `test/controller_service_test/`、`test/middleware_controller_test/`：分层集成，一般使用内存 SQLite 与 mocks。

## 约束与验证

房间创建/删除服务显式接收 Principal 并重查 session；新建 CreatorID 从 UserID 派生，DELETE 仅创建者（不要求成员），repo 保留 id/creator_id 条件。旧 schema 通过 InitDB 添加可空列，不回填归属；NULL/0 旧房间拒删，不自动认领。升级/回退和离线审查流程见 [归属策略](../docs/room-ownership.md)；当前只验证临时库，未运行真实迁移。

REST/Kafka 创建消息共用 SendMessage，在同一事务内验证 session/成员、写入 messages/send_requests/message_outbox；生产提交后交由 outbox_relay.go 租约领取/后台发布；注入式即时发布也使用同一领取路径。JWT Claims 为 `Username` / `SessionID`，HS256、`chords-access` audience；middleware 将已校验 user/session 复制为 Principal。消息、membership 服务显式接收 Principal 并检查有效 session、房间成员及身份/所有权；房间搜索保持开放给登录用户。WS 授权入口 `/api/auth/ws-authorize`，Kafka 命令的 `chords-command` header 绑定 protobuf 内容摘要并在消费时再次授权。新历史走 message_history.go 的 before_seq/after_seq/snapshot_seq 与同快照高水位；旧 before_id/数组最多100兼容。Redis 参数/topic 保持代码定义；broker 地址由 CHORDS_KAFKA_BROKERS 覆盖，旧命令仅 CHORDS_LEGACY_COMMANDS=1 启用，不能只改 YAML。

```bash
cd backend
go test ./...
# 根据改动范围选跑：
go test ./internal/service ./internal/controller
go test ./test/controller_service_test ./test/middleware_controller_test
```

常规测试无需启动 main；启动需配置 `JWT_SECRET`（至少 32 字节），不再清空 Redis。用户输入/输出使用 `internal/dto`；密码在共享服务路径哈希，不能在 controller 预先哈希。登录签发后，以事务替换全部旧 session；失败必须回滚。鉴权/持久化修改应验证失效 session、非法输入和数据库行为。详见 [配置](../docs/configuration.md)、[已知问题](../docs/status.md)。

S2.1 键为 `(user_id, request_id)`，不含 session；指纹绑定版本/room/实际内容，同键异内容 409，删除保留请求墓碑并返回 deleted=true。事务封装使用同连接显式 BEGIN/COMMIT/ROLLBACK，覆盖真实延迟 FK COMMIT 失败；不要退回单独写库或普通 sql.Tx 提交失败清理。SQLite busy 有界重试/503，不引入全局 mutex。共用准入源在 proto/kafka/admission.go（随 make proto 复制），UTF-8 content 256 KiB、标识 128 可见 ASCII 字节。新 HTTP receipt/history server_message_id 为字符串。迁移仅临时库验证，真实启动/回退边界见 [说明](../docs/message-persistence.md)。

S2.2–S2.4：room_seq 与高水位/消息/请求/outbox 同事务，重试不分新号；删除不回退高水位。relay 只读 v1/v2 原快照，坏版本隔离、删除抑制、mark失败租约到期重发；不重新构造事件。HTTP构造和接纳不依赖broker。测试入口 recovery_test.go、message_schema_test.go、routes/message_runtime_test.go；test/recoveryfixture 仅显式集成测试使用，强制创建临时库。真实迁移/回退未验证，旧 writer 不可直接切回写入。
