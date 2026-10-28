# 0001 · 身份、会话与房间授权

日期：2026-10-28。状态：已实施的 S1 边界；成员读取与仅创建者删除房间规则经用户明确选择，其余未决产品规则仍为 proposed。依据 [实施计划](../remediation-plan.md)、[历史审计 A01–A04](../architecture-audit.md)。

## 已确定规则

- 失败登录不得改变会话；成功登录继续替换全部旧 session。签发先于持久化，撤销和新建 session 在同一 SQLite 事务内提交；不改变为多设备策略。session/JTI 使用加密随机值。
- REST Principal 从已验证 User/Session 复制为只读值，通过 request context 传递；不能绑定 body 的用户/session 字段。消息、membership 服务显式接收 Principal，并在业务操作前复查 session。
- 用户明确选择：仅永久成员可读历史和订阅。发送也要求永久成员；消息只能以本人身份发送、由本人删除。历史全量、limit、before_id 分支均执行同一检查。
- 当前 UI 是搜索/发现房间后自行加入。搜索、列表、房间元数据对登录用户保留；membership 添加及列表查询只允许本人，不能伪造别人的加入。仅新增房间 CreatorID 归属字段；没有引入通用 admin/role/invite 系统。
- 用户在原规划 chat 明确选择“仅创建者可删，补齐归属与旧数据策略”。新房间从可信 Principal 派生 CreatorID；创建/删除服务重查 session，删除要求创建者，并在 repo 写条件中保留归属检查；不要求创建者另行成为成员。旧表加可空 creator_id，NULL/0 归属不明则拒绝普通删除；不猜测创建者、不自动认领。兼容、回退与可审查的离线指定归属流程见 [房间归属](../room-ownership.md)。
- 用户创建接受独立 DTO，响应只含公开资料；注册和 POST /users 共用一次哈希路径，模型本身也禁止序列化密码/session。

## WS 与 Kafka 实施方案

浏览器不能设置任意握手 Authorization header。选择带 5 秒超时、4 KiB 上限的 JSON auth 首帧，随后 ready，再开始现有 protobuf。认证前不注册连接；不把长效 token 写入 URL、subprotocol、Kafka 或日志。只解析签名无法处理 session 撤销，因此 gateway 不持有签名密钥，调用 backend `/api/auth/ws-authorize`（HTTP 超时 3 秒）。外部浏览器应使用 HTTPS/WSS，跨主机认证 RPC 应使用可信 TLS 网络。

连接身份在加入 MemoryStore 前固定，删除旧的 SetClientUserID 接口；客户端 user_id 为 0 时由服务端补全，其他人 ID 导致 1008 关闭。每连接限流在逐帧认证 RPC 之前执行。join/leave/message 逐帧查询 session/永久成员；房间广播投递前再查询，空闲连接每 5 秒复查 session。无效凭据/权限按 1008 关闭，前端提示错误并停止重试；backend 不可达、5xx 或 429 按 1013 关闭并退避重连。认证数据库故障通过 503 保留为基础设施错误。

Kafka 保留原 event.proto，不增加任何秘密字段。backend 为 message 的 protobuf SHA-256 摘要签发 30 秒有效的 `chords-command` JWT，绑定 user/session/room。gateway 只在 user-request header `chords-command` 传递；async/retry 保留 context/header。backend 消费时验证签名、audience、过期、摘要、身份与房间，再执行显式 Principal 的消息服务，复查 session/永久成员。notification 与前端无该 header。JWT access 使用不同 audience `chords-access`；命令凭据不能作为 access token 使用。

撤销语义以各入口授权检查时刻为准：后续检查拒绝失效会话，Kafka 即使已排队也重新检查。已经通过检查的在途/排队操作可能完成，不提供撤销与写库/投递的线性化保证。空闲关闭检测约 5 秒加 RPC 延迟；出站排队数据不能追溯收回。没有提供即时推送式撤销协议。

## 取舍与限制

- 逐帧和逐接收者 RPC 增加 backend 负载、广播延迟及可用性依赖。当前先采用可验证、拒绝失败的边界；未经测量不声称具备扩容能力。后续可设计有明确失效期限的授权缓存或撤销事件，但不能永久缓存身份。
- 30 秒授权可限制凭据泄漏窗口；Kafka backlog 超过期限会丢弃命令，缺失/无效凭据也会被消费。当前仍无 ACK/outbox/幂等补发，属于 S2 范围。相同有效命令在窗口内重放仍可能重复落库。
- 授权/落库失败不再发布 notification；数据库 ID 回填、发布失败恢复、可靠消费提交仍未实现。不能把这次改动宣称为消息可靠性整改完成。
- 旧 token 缺少 audience，发布需重新登录并同步部署前后端。运行必须从环境提供至少 32 字节 JWT_SECRET；Compose/kind 示例引用外部配置，不提交密钥。gateway 只配置可信 backend URL。
- 内部 /fanout 已在本轮补专用环境凭据 FANOUT_TOKEN：与 JWT_SECRET/user/command 凭据分离，无默认值，至少 32 字节无空格可见 ASCII；配置及构造器校验，读取 body 前用标准库安全比较拒绝不可信调用。worker 禁止重定向，非 2xx 返回错误；配置和对端 body 不进入日志。服务认证不会替代 session/成员投递检查。开发 HTTP 无防窃听/重放能力，跨不可信网络需可信 TLS；broker ACL/网络治理仍未完成，不能宣称完整生产安全。轮换单凭据需协调停机/拒绝窗口，当前失败消费可能丢通知。

## 未决产品规则

房间删除已按用户选择实施，不再等待权限选择。跨用户邀请也未实现，不能把伪造 membership 当作邀请功能。POST /users 当前仍为任意登录用户可创建账户，是否需要管理权限另行定义。S1b 因这些未决项及真实链路未验证保持部分完成。

## 验证与回退

[账户回归](../../backend/test/controller_service_test/account_security_test.go) 验证失败登录、DTO、哈希、恶意模型字段；[事务回归](../../backend/internal/service/auth_atomic_test.go) 验证签发/插入/撤销失败回滚及并发替换；[授权回归](../../backend/test/controller_service_test/authorization_security_test.go) 覆盖 REST 与签名命令的身份/成员/作者/撤销边界；[WS 回归](../../connection/cmd/server/auth_test.go) 使用真实本机 WS 和模拟 backend；前端 hook 测试验证 auth/ready 顺序与错误处理。[HTTP fanout→WS 回归](../../connection/cmd/server/fanout_integration_test.go) 使用生产 WS mux/pipeline/source/Hub 与模拟 backend；另有 worker 真实 HTTP/mock registry、重定向、配置保密与真实入口失败顺序测试。测试不连接真实 Kafka/Redis。

[房间归属回归](../../backend/test/controller_service_test/chatroom_ownership_test.go) 验证真实 token/Principal 的创建者、其他成员/非成员、匿名、伪造字段及未知归属规则；[迁移回归](../../backend/internal/app/db_ownership_test.go) 对全新和旧 schema 的临时 SQLite 执行真实 InitDB，确认空归属、已有归属、成员记录和重复初始化兼容。

回退需要协调 gateway/backend/frontend 的认证协议与配置，不应单端绕过鉴权；本次新增可空 creator_id 的本地 schema 逻辑并验证临时库，没有执行真实数据库迁移。回退不得恢复无创建者校验的旧删除入口，应保留列和值并按 [归属回退说明](../room-ownership.md) 处理。真实基础设施和浏览器完整聊天链路仍需隔离环境验收。
