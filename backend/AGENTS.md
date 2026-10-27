# Backend

遵循 [根指南](../AGENTS.md)。独立 Go module `backend`。

## 定位

- `cmd/main.go`：SQLite 初始化、Redis 连接及清空、HTTP 启动（固定 `:8080`）。
- `internal/routes/routes.go`：依赖装配、REST 路由和 Kafka 启动；完整 `SetupRouter` 依赖真实 Kafka。
- `internal/controller/` → `internal/service/` → `internal/repo/`：JSON/状态码 → 业务接口 → GORM/RepoContainer；接口修改同步 mocks。
- `internal/model/`、`internal/app/db.go`：模型与 AutoMigrate 列表。
- `internal/service/kafka_service.go`：user-request 消息落库、notification 发布。
- `internal/middleware/jwtauth/session_auth.go`：路由实际使用的认证；相邻 `jwtauth.go` 的另一套实现未接入当前 router。
- `test/controller_service_test/`、`test/middleware_controller_test/`：分层集成，一般使用内存 SQLite 与 mocks。

## 约束与验证

REST 创建消息不会自动实时广播。JWT Claims 为 `Username` / `SessionID`，与 gateway 的 `user_id` 设计尚未统一。分页分支使用 `limit` / `before_id`，查询倒序后反转为页内升序。Redis 参数、Kafka 地址/topic 在 Go 中硬编码，不能只改 YAML。

```bash
cd backend
go test ./...
# 根据改动范围选跑：
go test ./internal/service ./internal/controller
go test ./test/controller_service_test ./test/middleware_controller_test
```

常规测试无需启动 main；main 会 `FLUSHDB`。鉴权/持久化修改应验证失效 session、非法输入和数据库行为。详见 [配置](../docs/configuration.md)、[已知问题](../docs/status.md)。
