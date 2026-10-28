# Backend

遵循 [根指南](../AGENTS.md)。独立 Go module `backend`。

## 定位

- `cmd/main.go`：密钥校验、SQLite 初始化、Redis 连接（不清空）、HTTP 启动（固定 `:8080`）。
- `internal/routes/routes.go`：依赖装配、REST 路由和 Kafka 启动；完整 `SetupRouter` 依赖真实 Kafka。
- `internal/controller/` → `internal/service/` → `internal/repo/`：JSON/状态码 → 业务接口 → GORM/RepoContainer；接口修改同步 mocks。
- `internal/model/`、`internal/app/db.go`：模型与 AutoMigrate 列表。
- `internal/service/kafka_service.go`：user-request 消息落库、notification 发布。
- `internal/middleware/jwtauth/session_auth.go`：路由实际使用的认证；相邻 `jwtauth.go` 的另一套实现未接入当前 router。
- `test/controller_service_test/`、`test/middleware_controller_test/`：分层集成，一般使用内存 SQLite 与 mocks。

## 约束与验证

房间创建/删除服务显式接收 Principal 并重查 session；新建 CreatorID 从 UserID 派生，DELETE 仅创建者（不要求成员），repo 保留 id/creator_id 条件。旧 schema 通过 InitDB 添加可空列，不回填归属；NULL/0 旧房间拒删，不自动认领。升级/回退和离线审查流程见 [归属策略](../docs/room-ownership.md)；当前只验证临时库，未运行真实迁移。

REST 创建消息不会自动实时广播。JWT Claims 为 `Username` / `SessionID`，HS256、`chords-access` audience；middleware 将已校验 user/session 复制为 Principal。消息、membership 服务显式接收 Principal 并检查有效 session、房间成员及身份/所有权；房间搜索保持开放给登录用户。WS 授权入口 `/api/auth/ws-authorize`，Kafka 命令的 `chords-command` header 绑定 protobuf 内容摘要并在消费时再次授权。分页分支使用 `limit` / `before_id`，查询倒序后反转为页内升序。Redis 参数、Kafka 地址/topic 在 Go 中硬编码，不能只改 YAML。

```bash
cd backend
go test ./...
# 根据改动范围选跑：
go test ./internal/service ./internal/controller
go test ./test/controller_service_test ./test/middleware_controller_test
```

常规测试无需启动 main；启动需配置 `JWT_SECRET`（至少 32 字节），不再清空 Redis。用户输入/输出使用 `internal/dto`；密码在共享服务路径哈希，不能在 controller 预先哈希。登录签发后，以事务替换全部旧 session；失败必须回滚。鉴权/持久化修改应验证失效 session、非法输入和数据库行为。详见 [配置](../docs/configuration.md)、[已知问题](../docs/status.md)。
