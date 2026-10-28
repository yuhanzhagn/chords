# 0002 · 消息持久化、确认与恢复的最小切片

日期：2026-10-28。状态：proposed，只有设计，尚未实施或迁移。依据 [整改计划 S2 / N4](../remediation-plan.md)、[历史审计 A05–A07/A11](../architecture-audit.md) 与 [当前协议](../contracts.md)。本轮 N1/N2 不改变消息协议、数据库或消费提交策略。

## 问题与目标

现有 REST 创建只写数据库；Kafka 创建写库后立即发布 notification，发布失败无法恢复，重复消费可重复落库；出站未回填服务端 ID。30 秒 chords-command 在消费时重新校验，backlog 可能使合法请求过期丢弃，客户端只见 pending。HTTP fanout 对 protobuf 的 bytes 做 base64 包装，WS 允许的 1 MiB 不一定能通过下游 1 MiB body 限制。

目标是一个发送用例维护一次持久事实和同事务待发布记录，允许重复通知，以稳定 ID 去重并从历史恢复。Kafka/HTTP/WS 的网络成功不等于用户已看到消息。不承诺端到端 exactly-once。保留 SQLite 与独立模块；不引入 Kafka 事务、CDC、数据库替换或新身份服务。

## 标识与约束

| 字段 | 拟定含义与生成者 | 存储/重试约束 |
| --- | --- | --- |
| request_id | 客户端在首次发送前生成的随机 128 位 ID；一次逻辑发送始终复用 | 发送记录唯一键 `(authenticated_user_id, request_id)`，不含 session，支持重登重试；绑定版本、room、规范化内容的 SHA-256；同键异内容返回 409，不创建第二条消息 |
| server_message_id | backend 的 Message.ID，数据库分配的持久主键 | 一次请求只分配一次；响应、历史、通知一致；wire uint64，TS 保持字符串，HTTP 新协议用十进制字符串避免 JS 精度丢失 |
| event_id | backend 为一条不可变通知事实生成的随机 128 位 ID | outbox 唯一键；relay 重发复用同 ID，不能每次重试生成；与消息主键分开，未来编辑/删除可生成不同事件 |
| room_seq | backend 在每个房间事务内分配的单调序号 | 后片加入 `(room_id, room_seq)` 唯一键；代表持久顺序，不依赖时钟/Kafka offset；不承诺连续无洞，删除不重编号 |
| temp_id | 当前 optimistic UI 的客户端匹配键 | 兼容期保留；不能作为跨用户全局唯一键，也不能被服务端 ID 覆盖 |

拟建 `send_requests` 记录请求指纹、message_id/event_id、提交时间；`message_outbox` 保存 event_id、room_id、固定的持久化事件快照、schema_version、状态/尝试次数/下次重试时间。请求记录与消息/事件同事务插入，消息后续删除也保留幂等墓碑，不能因删消息或清理 outbox 让旧 request_id 重新创建。首次实现不自动清理幂等键；保留期/归档需另定可证明的重试期限。

并发冲突由数据库唯一约束裁决，不采用先查后插的竞态方案。SQLite busy/事务错误必须有有界重试和明确 503/retryable 结果，不能默默成功。重复请求读取既有结果前仍校验当前 session/成员，避免通过幂等查询泄漏无权资源；拒绝不撤回已经持久化的事实。

## 接纳、授权与客户端可见结果

- `accepted`：backend 已承诺保留本次请求；拟定首版不单独发早期 accepted。gateway enqueue、签发 command token、Kafka producer 成功均不得触发 accepted ACK。
- `persisted`：消息、幂等键、outbox 的 SQLite 事务提交成功。首版 HTTP 成功和后续 WS persisted ACK 都以此为准，带 request_id/server_message_id/event_id、服务端时间，后片再含 room_seq。
- `delivered`：只有特定接收者/连接主动确认 event_id/room_seq 后才可声明其送达；HTTP 204、Hub 入队、broker publish 成功均不算 delivered。首阶段不暴露 delivered，不定义全房间/全设备已读。

新用例在事务中校验不可变 Principal 的 session 和永久成员、本人发送，再插入消息/outbox；授权时点是数据库事务观察到的状态，不把签发命令视为接纳。提交后 relay 发布的是 backend 的持久事实，不能再次因发送者下线/撤销/退房丢弃；接收者投递和历史读取继续执行当前 session/成员授权。撤销不追溯删除已提交消息。禁止把可重放的“持久信任”外放成浏览器任意事件凭据，broker 写入权仍需受控。

处理 backlog 分两步：

1. S2.1 保留当前 Kafka 入站与 30 秒校验，验证通过后才进入共用事务。尚未消费的旧命令仍会过期；明确这是过渡期限制，不能声称 durable acceptance 已覆盖 WS。不得单纯延长有效期、去掉鉴权或把长效用户 token 写入 Kafka。
2. S2.3 将 gateway 的发送改为同步调用可信 backend 的接纳入口（拟 `POST /api/messages/accept`），由 backend 验 session/成员并提交，只有收到 persisted 才向 WS ACK；Kafka 只承担持久事实的异步通知。此时从默认 WS 路径移除“先排 user-request 后接纳”的队列，旧入站适配器留受控兼容窗口，不能同时为同一请求制造两份事件。

断网/超时属于结果未知，客户端保持相同 request_id 重试，不能显示确定失败或新建 ID。401/403、409、413 是可见终态；服务暂时不可用/503 为可重试。若提交成功而响应丢失，重试返回原持久结果。旧命令过期仍可能无 ACK；兼容期 UI 需显示“结果未知，可重试/查询”，不能只依赖 notification 消除 pending。长效 access token 仅在已存在的 gateway→可信 backend 边界使用，不进入 broker/outbox。

## 尺寸准入

拟定第一片 content UTF-8 上限 256 KiB，request_id/temp_id 等输入标识各最多 128 ASCII 字节；服务端 ID/时间由 backend 生成。该上限须经实际编码边界测试确认，并对 REST 与 WS 共用；超限 HTTP 413 / WS 可识别 `message_too_large`（在新错误协议就绪前使用明确的关闭码 1009），不进入异步队列或数据库。

不能只比较 content 大小：共用准入同时编码规范事件并验证 WS protobuf、Kafka record（包括 header）和 HTTP JSON 预算。JSON 中 bytes 至少占 `4 * ceil(content_bytes / 3)`，还包括字符串转义/元数据及 `user_ids` 数组；要求 `len(json.Marshal(FanoutRequest)) <= 1 MiB`。S2.1 对非零 room_id 的现有广播路径，应省略未参与过滤的 user_ids（仍由 Redis 确定具体 gateway），令每个网关仅收到一个固定大小 envelope；不得把 user_ids 分批却继续全房间广播而制造重复。需要覆盖省略前后接收集合相同、每个网关一次投递。room_id=0 的 user/client ID 语义仍需后片澄清，再采用有界批次；本片不扩展该 fallback。入口预算预留已测量的固定 envelope 空间，不能假设房间在线人数有上限。worker 仍检查实际最终 body，但不能把这种下游拒绝作为唯一准入。

gateway HTTP source 后续改为明确的 max+1/MaxBytesReader 超限 413，避免 LimitReader 截断后才以 JSON 400 失败。覆盖内容刚好/超过限制、转义字符、最大合法标识、protobuf varint 最大值、base64 的余数边界和大量接收者；消息在入口可接纳后不得因固定包装超过下游限制而静默丢通知。这些是下一轮设计要求，本轮没有修改现有大小限制。

## 分片、文件入口与验收

### S2.1：下一轮最小实施范围

统一 REST/Kafka 发送用例、SQLite 事务幂等与 outbox、稳定 ID/持久时间、共用尺寸准入和故障回归。沿用 [MessageService](../../backend/internal/service/message_service.go)、[KafkaService](../../backend/internal/service/kafka_service.go)、[message repo](../../backend/internal/repo/message_repo.go)、[controller](../../backend/internal/controller/message_controller.go)、[初始化](../../backend/internal/app/db.go)；拟新增 send-request/outbox 模型、repo 和测试。不可只给 Kafka 写路径加事务而保留 REST 的另一路。

新协议字段从 [event.proto](../../proto/kafka/event.proto) 加法演进：request_id/event_id/server_message_id 等使用新字段号，不重用 Id/TempId 或手改生成文件；四个消费者一起生成/验证。老客户端临时将非空合法 temp_id 映射为 request_id；缺失时只能服务器分配一次请求 ID，明确无法对客户端重发去重，新客户端必须发送 request_id。新字段与旧事件 Id 的含义不能混用；上线需先支持新字段读取再切发送，不能依靠旧 UI 已具备 upsert。

第一片可保留事务提交后的现有即时 publish 作为过渡：发布成功后再标记该 event_id 的 outbox 状态，失败保留 pending；publish 与标记之间崩溃可能重复。此时尚无后台恢复器，必须明确 pending 仅被持久保存，S2.2 才恢复发布；S2.1/S2.2 应作为同一可靠性发布门槛，不单独宣传可恢复聊天。不得同时保留绕过 outbox 的第二套 notification 事实生成器。

验收：同一键同内容串行/并发重试只有一条消息、一条幂等记录和一条 outbox；同键异内容 409；不同用户同键互不冲突；注入任一 insert/commit 失败全回滚；事务提交后发布失败事实不丢；响应丢失重试返回原 ID/时间；无权/撤销不写入；边界大小在 REST/WS 两入口一致拒绝。使用内存 SQLite + 故障 trigger/mock producer；不访问真实库。

### S2.2：relay 与恢复

后台 relay 对 pending outbox 有界扫描、租约/领取、重试退避；以 room_id 为 Kafka key，保存固定 event_id。先单 SQLite writer/单 relay 模式，不宣称多节点数据库能力。producer 确认后才更新发布状态，失败/重启继续尝试。验证提交后进程退出、broker 故障、publish 后标记前崩溃、重复通知和永久坏记录的隔离/观测。保留至少一次发布，不用 fanout 的 MarkMessage 小补丁冒充整体可靠性。

### S2.3：同步接纳与 ACK / 前端幂等

入口见 [gateway auth](../../connection/cmd/server/auth.go)、[pipeline](../../connection/cmd/server/main.go)、[Kafka sink](../../connection/internal/adapter/kafka/event_sink.go)、[WS hook](../../frontend/src/hooks/useChatSocket.tsx)、[消息状态](../../frontend/src/hooks/useMessages.tsx)。后端接纳返回 persisted，WS 发可关联 request_id 的结果；前端按 server_message_id/event_id upsert，temp_id 只匹配 pending。验证 ack 丢失、重连/重登重试、撤销前后、backend 不可用、旧 backlog 过期与明确未知结果；关闭旧异步入口前须有兼容窗口/回退方案，不能绕过 N1 服务凭据或成员投递检查。

### S2.4：房间顺序、分页与补拉

数据库事务分配 room_seq，历史支持 `before_seq`（倒序查、页内升序）与 `after_seq`（升序补拉），旧 before_id 暂保留且不与新游标混用。分页同事务快照/高水位返回 next_cursor/has_more，按房间过滤；删除可产生洞，不能把洞判为丢消息。relay/broker/WS 允许乱序与重复，客户端去重并按 room_seq 排序，重连从最后确认游标拉取。日志保留/删除语义及游标过旧需返回可识别 reset，不偷偷重置到 newest。

已有 Message 的 seq 迁移按每房间稳定 ID 顺序回填并记录为“历史规范序”，不能声称恢复真实旧发送时序；旧消息不伪造 request_id、不自动批量创建 outbox 触发历史广播。迁移须在临时数据库副本验证行数、唯一索引、外键/删除、旧分页兼容，再由用户另行授权真实数据迁移。房间删除 CreatorID 已按用户选择实施，旧未知归属拒删；其迁移与本消息序号设计相互独立。

## 验证、迁移与回退

下一片至少执行 `make test-backend`、`make test-connection`、`make test-fanout`；协议/前端受影响执行 `make test-frontend`、`make build-frontend` 和四份生成产物对比；涉及 WS 生命周期执行 `cd connection && go test -race ./cmd/server ./internal/gateway ./internal/sink ./internal/source`；最后 `make check-docs`、`git diff --check`。本机 mock 故障通过后，再在隔离 Kafka/Redis 环境验证重启与消费，不能以单元测试代替运行证据。

新增表/字段/索引需显式版本化迁移与回滚兼容检查，不能依赖启动 AutoMigrate 未经验证地处理真实库。SQLite 写锁、幂等键空间增长、消息删除墓碑、重复发布与旧 UI 均为发布风险；先扩展读兼容，再启用新写路径。回退应暂停 relay/新写入口并保留 pending/outbox/幂等事实，不删除它们后回到可能重复的旧写入路径。当前请求未授权执行该迁移、部署或上述 S2 代码。
