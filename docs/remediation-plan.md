# 架构整改实施计划

本文件保留分轮历史记录；最新进展见末尾 2026-10-30 S2.2–S2.4 连续实施记录。

日期：2026-10-28。依据 [架构审计](architecture-audit.md)，代码基线为 `08c2f3a`。当前审计报告与文档索引尚未提交。用户已授权规划，并新建一个 chat 开始实施；该授权不包含新的 commit/push、部署、生产数据操作或依赖/框架迁移。

## 执行方式

保留 backend、connection、fanout 的职责及独立 Go module。按依赖顺序完成下面里程碑；每个工作项采用“复现 → 安全/正确行为回归 → 最小一致修复 → 相关检查 → 一次聚焦自审”。以实际代码和验证更新状态，不能仅因为写好计划或旧探针 PASS 就标记完成。

新 chat 使用本项目现有本地工作区，确保可读取未提交审计与本计划；本 chat 完成交接后不再并行编辑。先完成 S1a，再推进 S1b。其余阶段作为后续路线，不自动扩大本次 chat 为数据库迁移或基础设施部署任务。

## 阶段总览

| 阶段 | 审计项 | 工作与交付 | 前置/完成关卡 | 状态 |
| --- | --- | --- | --- | --- |
| S1a：账户与启动安全 | A02/A03/A04 | 修复错误登录撤销、DTO/密码处理、正常启动 FLUSHDB；回归与文档 | 错误密码不改变会话；用户响应无密码/哈希/秘密；所有创建入口哈希；启动不清共享状态 | 已完成本地代码与回归；见 2026-10-28 实施记录 |
| S1b：身份与业务授权 | A01 及 A02 相关凭据边界 | Principal、REST 资源授权、WS 身份/订阅授权、前端凭据兼容；短 ADR | 无用户冒用；无匿名业务订阅；权限决策和兼容性明确 | 部分完成；成员/创建者删除规则已实施，账户管理/邀请仍未决，真实链路未验证 |
| S2：持久化与消息协议 | A05/A06/A07/A11 | 统一发送用例、幂等请求、事务 outbox/relay、确认语义、房间顺序/游标、消息尺寸预算 | DB/broker 故障和重复请求下持久事实可恢复；已提交消息有稳定服务端 ID；边界尺寸入口处理 | S2.1–S2.4 本地实现/隔离验证完成；真实环境/迁移/部署仍未验证，见末尾记录 |
| S3：连接、路由与恢复 | A08/A09/A10 | 具体 gateway 租约、本地索引、统一 Disconnect/draining、并发修复、前端 upsert/重连补拉 | 同用户多标签/多网关正常；race 回归通过；旧实例不能误删新租约；断线可补齐 | 待实施；部分生命周期修复可独立，恢复协议依赖 S2 |
| S4：部署、可观测性与容量 | A12/A13 | 完整构建/部署、持久化策略、readiness、指标、准入保护、索引与有界分发 | 重建保留数据；完整跨实例收发；按 SLO/负载给出容量与故障记录 | 待实施；多实例发布依赖 S2/S3 |

不估算没有测量依据的工期/QPS。S4 中准入“20 并发实际 5”可作为独立小修提前安排；不能以此取代前面安全修复。

## S1a：新 chat 立即实施

1. 阅读根及 backend 指南、审计 A02–A04，以及 controller/service/repo/session/模型/启动入口和相关测试。
2. 把审计场景转为期望正确行为的常驻测试。已有 `/tmp/chords-architecture-audit/` 可作线索；其测试断言的是已观察缺陷，不可原样加入 CI 当作安全回归。
3. 登录：错误密码、未知用户名和绑定失败不撤销已有会话。保留目前“成功登录会撤销旧会话”的既有意图；把密码校验和 session 更新放进服务层，必要事务应保证新 session 创建/签发失败不会留下错误撤销状态。覆盖并发与失败，不能把该修复改成新的多设备产品策略。
4. 用户：独立输入/输出 DTO；输出不含 Password、哈希、RefreshToken 或内部 session；所有创建入口使用统一哈希路径并正确返回哈希错误。禁止从输入任意绑定 ID、Sessions 等内部字段；注册/登录与现有前端响应保持兼容。
5. 启动：移除正常启动 ClearRedis/FLUSHDB，先检查 InitRedis 错误再使用 client。不要为此次修复增加默认可执行的重置接口；若保留开发重置能力，必须显式、隔离且最小范围。
6. 对照验收自审，修正问题并补文档。根据实际改变同步 backend/根 AGENTS、configuration、contracts、status 及本计划；审计报告作为历史证据保留，注明整改引用即可。

S1a 验收必须包括：错误密码后原 token 仍有效；成功登录会话策略保持既有意图；用户列表/创建响应无凭据；注册与 POST /users 均不存明文或重复哈希；恶意内部字段不被模型绑定；Redis 初始化失败不 panic，正常启动不清其他服务状态。

## S1b：在 S1a 后推进，先明确最小授权规则

- 建立统一且不可变的 Principal，REST middleware 不再丢弃认证结果；业务方法不能只靠 controller 传入未经校验的 user_id。
- 明确可由当前功能推导的规则：用户不能冒用别人的发送身份、删除别人的消息或伪造他人的 membership；维持现有房间搜索/发现入口。不新增 admin/owner/invite 系统来绕过缺失产品需求。
- 在 ADR 中区分事实、暂定实现与待用户明确的产品规则：公开历史/房间是否允许非成员读取、房间删除者权限、是否允许邀请他人。遇到这类真正影响产品行为的问题，可异步提出具体选择，继续已明确的修复；不能以自己的新策略宣称 A01 全部完成。
- WS 设计必须把 backend claims、浏览器握手能力、session 撤销和房间权限一起考虑，禁止只加入一个未绑定身份的 JWT middleware。不要默认用长效 JWT URL query；选择有明确泄漏边界的握手/短效票据/首帧认证方案并记录理由。无需仅为此引入新认证产品。
- 跨 Kafka 入站的身份需由可信 gateway 派生；backend 消费命令继续检查业务权限。失效 session 的实际语义要清楚，不能只解析签名并永久缓存认证状态。
- 同步前端凭据传递和错误处理、协议/配置/测试；需要跨服务 schema 变化时遵循 proto 指南重新生成四份产物，不手改生成代码。

S1b 验收：Alice 不能以 Bob 身份发送/删 Bob 消息；匿名/无效凭据不能获得 WS 业务订阅；已定义的房间策略在 REST 与 WS 一致；session 撤销和连接身份不可变；现有有效用户登录与正常聊天链路兼容。未确定的公开权限策略必须列出，不能默认为已解决。

## 后续阶段的设计边界

S2 先定义 persisted/delivered 与重放保证，再设计 outbox。当前可保留 user-request topic 并先做轮询 relay；是否 CDC、是否缩短 Kafka 入站路径应另行有证据决策。不在 S1 引入 PostgreSQL/Kafka 事务或大改消息 schema。

S3 以当前房间广播为依据评估 room→gateway 租约，区分 user ID、connection ID 和实例 incarnation。并发修复应覆盖真实生产访问路径，不能以 recover 掩盖 race。补拉游标等依赖 S2 的承诺。

S4 先确定演示与多实例两种部署目标；若选择共享数据库，单独记录迁移/回滚计划。没有 Docker/集群/负载工具就记录未验证，不能把清单解析成功当作部署成功。

## 验证与交付

```bash
make test-backend
make test-connection
make test-fanout
make test-frontend
make build-frontend
make check-docs
```

按实际影响选择命令；S1a 至少完成 backend 全模块检查及有意义的回归，影响前端再执行前端测试/构建。WS/并发修改后按需执行 `cd connection && go test -race ./cmd/server ./internal/gateway ./internal/sink` 并补审计遗漏场景。最终 `git diff --check`，保留已有审计文档修改。

之前基线：三个 Go 模块检查、前端 build/test/lint、文档检查已通过；fanout 仍没有常驻行为测试；Python 运行未验证。本机 WSL Docker 集成不可用的记录应重新检查当前环境，不要直接当成永久阻塞。

交付包括实际修改、测试命令与结果、未完成项和更新后的阶段状态。新的 commit/push、部署和数据迁移需要对应明确授权；当前用户请求只授权开 chat 并开始实现。无需为例行代码实现和本地测试重复询问许可。

## 2026-10-28 实施记录

S1a 已完成：controller 不再在验密前撤销 session；服务层先签发、事务撤销旧会话/创建新会话。签发、撤销、插入失败回滚测试与并发替换测试通过。DTO 和 User 模型均屏蔽凭据；register/POST users 共用明文到 bcrypt 的服务路径，拒绝哈希失败，恶意 ID/Sessions 不写入。移除 ClearRedis/FLUSHDB，初始化失败返回 nil/error 并关闭 client；启动前要求 JWT_SECRET。正常启动保留 Redis 由调用路径审阅确认，没有运行完整 main 或共享 Redis。

S1b 已实施的边界：REST 不可变 Principal；消息按有效 session/成员/本人/作者授权；membership 只可操作本人；用户明确选择仅成员历史和订阅。WS 5 秒首帧 auth/ready、固定身份、逐帧/投递/空闲 session 检查；Kafka 使用 30 秒、绑定内容摘要的 header 授权并在 backend 重新验证，不变更 event.proto。前端同步认证、重连 ready 后加入所选房间和可见错误；辅助 wsload 改为环境凭据。配置、示例和 [ADR 0001](decisions/0001-identity-and-room-authorization.md) 已同步。

S1b 仍部分完成：房间删除已按用户明确选择实施仅创建者可删，旧数据保留未知归属并拒删；账户创建是否要管理权限、邀请模型未定义。内部 fanout 服务认证已由本轮 N1 完成；真实 Redis/Kafka/浏览器完整链路仍未验证。短效命令过期会被消费丢弃，逐接收者授权 RPC 的性能未测量；均不可隐去。后续 S2–S4 保持路线图，未实施 outbox、数据库/框架迁移、部署或压测。

本次实际验证命令与结果见 [测试记录](testing.md#2026-10-28-s1-实施验证)。未 commit/push，保留原审计文档与计划改动。

## 2026-10-28 下一轮：内部投递鉴权与集成验证

用户已授权“规划下一步，唤起新 chat 执行”。上一轮实施 chat 已完成并处于 idle；当前工作区保留全部 S1a/S1b 代码与文档，仍未提交。下一轮继续使用同一本地 checkout，不创建 worktree，不覆盖前轮成果；本规划 chat 完成交接后不并行修改。

### N1 · 当前执行目标：内部 /fanout 服务鉴权

状态：本地实施和回归完成。使用环境 FANOUT_TOKEN / X-Chords-Fanout-Token；配置与生产构造器拒绝缺失/短值/非法字符，真实入口子进程证明错误在 Kafka/Redis 前返回。gateway 在 body 前鉴权；worker 独立 client 禁止所有重定向，非 2xx 返回错误且不记录响应 body。配置整结构日志已移除；Compose/Secret/模块文档已接线。未改变 payload、路由、用户投递授权或失败消费提交策略。

- 保留已有用户 Principal、WS 首帧认证、session/成员投递检查与 Kafka 命令授权；服务凭据是额外的 worker→gateway 边界，不能替代用户/房间授权。
- 建议采用最小可验证的服务专用共享凭据方案，由 worker 与 gateway 的环境注入；与 JWT_SECRET/access/command token 分离，无默认开发密钥，缺失/过短应在启动时明确失败。具体配置名称和构造器可由实现者按现有 typed config 风格确定。不得把服务密钥传给 backend 之外的无关角色、浏览器、URL、Kafka payload 或日志。
- gateway 在读 body/解码/广播之前拒绝无凭据、错误凭据或错误用途的 token；有效凭据才进入原 fanout 验证路径。认证比较使用标准库安全能力，不自创加密协议；无需新增认证服务或新依赖。
- worker 携带凭据并正确传播 401/403/配置错误；不通过 HTTP redirect 向其他目标转发服务凭据。保持现有分发路由和失败消费提交语义，不把本阶段误称为可靠性修复。
- 补齐 source/dispatcher/config/startup 行为测试，确保生产入口无法无意启用匿名模式。日志和错误不能泄漏凭据。共享 token 不能解决明文网络窃听和重放，部署文档必须说明跨不可信主机需 TLS/可信网络。
- 同步 .env.example、Compose（仅 connection/fanout 注入专用凭据）、现有 Kubernetes gateway Secret 引用、模块 AGENTS、configuration/contracts/development/testing 与 ADR。Kubernetes 尚缺 fanout Deployment 的事实保留，不扩成完整部署任务。

N1 验收：缺失/错误 token 不触发 Hub 或订阅投递；客户端 access token 和 command JWT 不能替代服务凭据；有效 token 保持原方法/payload 错误和正常投递；配置缺失失败；worker 请求带正确头、3xx 不泄漏凭据，4xx 不被当成功。

### N2 · 同轮完成：现有身份边界的跨层回归

状态：本地实施和回归完成。[跨层测试](../connection/cmd/server/fanout_integration_test.go) 通过真实 HTTP source → Hub → protobuf WS，覆盖合法成员、匿名/用户 token push、非成员订阅、成员关系移除及撤销 session。worker 独立测试验证真实 HTTP 请求头/payload/4xx/5xx 与全部常见 redirect 状态不转发。backend 为 mock，registry 为内存替身；无真实 broker/Redis/浏览器端到端保证。实际命令结果见 [本轮验证](testing.md#2026-10-28-n1n2-实施验证)。

在现有真实本机 WS 与 mock backend 夹具上，加入 HTTP fanout 服务认证与实际投递：已认证并有权限的客户端可收到合法内部消息，匿名 push、非成员/已撤销 session 不应收到受保护消息。允许使用本机 httptest、内存 registry、可注入依赖；不为本项启动共享 Redis、真实 Kafka 或生产进程。

分别验证 worker 端 HTTP 请求与 gateway 端认证/投递，明确 mock 边界。覆盖现有 WS auth/ready、身份不可变、session 撤销和成员检查，防止服务认证引入回归。该结果称为本机跨层集成，不是 Kafka/Redis/浏览器完整端到端验证。

按影响运行 make test-connection、make test-fanout（需新增常驻行为测试）、相关 race、make test-backend，以及受影响前端检查；最后 make check-docs / git diff --check。可做 Compose/YAML/Kubernetes 离线解析，工具不可用按实记录；不自动部署。

### N3 · 产品规则与阶段状态

状态：用户已在原规划 chat 明确选择“仅创建者可删，补齐归属与旧数据策略”，本轮追加代码与临时数据库验证已完成，不再等待选择。

- 新房间 CreatorID 从 Principal 派生，服务层复查 session；DELETE 只允许创建者，其他成员/非成员均拒绝，body 不能决定归属，repo 条件删除防止读后归属变化。
- creator_id 可空且有索引；旧 schema 通过真实 InitDB 兼容增加，NULL/0 未知归属拒绝普通删除。不推测/自动认领，不改真实数据库。临时新/旧库、重复初始化、未知与已知归属、原 membership 保留均已回归。
- 归属语义、响应字段/状态码、回退与人工离线审查方案见 [房间归属](room-ownership.md)、[ADR 0001](decisions/0001-identity-and-room-authorization.md)。前端只提交 name、读取 ID/Name，暂无删除控件，本轮无前端代码变动。
- POST /users 是否需要管理员、跨用户邀请仍未定义；S1b 因这些未决项和真实链路缺证保持部分完成。授权仅包括本地代码/迁移逻辑及临时库验证，不包含真实/生产数据操作或部署。

### N4 · 作为下一阶段交付：S2 最小实施切片

状态：设计已交付，代码未实施。具体语义、文件入口、迁移/兼容风险、故障验收与分片见 [proposed ADR 0002](decisions/0002-message-persistence-and-recovery.md)。推荐下一轮只实施 S2.1：统一发送用例 + SQLite 事务幂等/outbox + 尺寸准入 + 故障回归；relay/ACK/顺序游标分片推进。S2.1 尚不具备完整恢复能力，不能单独宣称 S2 完成。N4 未实施消息 schema/outbox 或生成产物；仅 N3 增加了房间创建者的可空数据库列并验证临时库。

N1/N2 完成后，S2 设计遵循以下约束，不在本轮自动实施 Outbox 或数据库迁移：

1. 先定义 request_id / event_id / server_message_id / room_seq 的含义、唯一约束、重试与历史游标。
2. 明确 accepted/persisted/delivered 区别，优先实现“只提交一次持久事实 + 可恢复发布”；通知可丢时由历史补拉恢复，不承诺端到端 exactly-once。
3. 首片建议为共用发送用例 + SQLite 事务幂等键/outbox + 故障回归，后片再做 relay/ACK/顺序与游标整合；每片写出依赖、迁移、兼容、验证关卡。
4. 必须处理现有 30 秒 chords-command 与 backlog 的关系：不能直接把用户 token 期限延长当作可靠性方案，也不能删除 session/成员校验；明确授权时点、接纳后的持久信任和过期/撤销结果如何被客户端观察。
5. 处理 A11 跨 protobuf/JSON/base64 的消息尺寸预算与明确入口错误。保留当前 fanout 失败仍推进消费等现状直到有完整可靠性方案，不顺手改半个提交策略。

本轮完成说明需分开列 N1/N2 实际代码与测试、未决产品规则、N4 仅规划内容、真实基础设施限制。所有前轮未提交改动均保留；当前授权不包含 commit/push、部署或生产数据操作。

## 本轮交付与真实限制

N1/N2/N3 已完成本地实现与回归，保留所有上一轮未提交改动。[验证记录](testing.md) 包含三模块测试、connection race、真实入口失败顺序、HTTP/WS 跨层与临时新/旧 SQLite schema 兼容。Kubernetes 仅离线渲染及 YAML 断言通过；Compose config 因 WSL Docker 集成缺失未成功，未启动依赖、部署、执行真实数据库迁移或 commit/push。前端本轮无改动，未重复构建。

N4 仅交付 proposed S2 设计：下一片统一发送用例、事务幂等/outbox 与尺寸准入；relay、ACK/同步接纳、房间序号/补拉分片推进。现有 30 秒命令过期和失败消费提交语义仍在。账户创建管理权限、邀请规则仍未定，S1b 保持部分完成；仅创建者删除规则已落实，不再列为未决。

## 2026-10-29 下一轮：S2.1 消息幂等与事务 Outbox

用户已授权“规划下一步，自动开启 chat 执行”。重新核对仓库：上一轮账户/会话/房间归属/内部 fanout 鉴权成果已在 `616b9b1` 提交并推送，交接前工作区干净；不再沿用“全部未提交”的旧说明。上一执行 chat 已 idle。新增本计划后由新 chat 在当前本地 checkout 实施，规划 chat 不并行改动。

### 本轮范围与执行顺序

依据 [ADR 0002](decisions/0002-message-persistence-and-recovery.md) 的 S2.1 实施，不再次只交付设计。模型和事务不变量先行，按“可失败的行为回归 → 数据库约束/最小修复 → 接口整合 → 相关验证 → 一次聚焦自审”推进：

1. **统一发送与身份边界**：REST 与 Kafka 适配层调用同一发送用例；不可变 Principal、有效 session、永久成员、本人发送等既有检查在事务边界内保留。保留 30 秒命令内容摘要校验，不能用延长期限/删除鉴权解决可靠性。
2. **事务幂等事实**：新增 send_requests 与 message_outbox（或等价明确模型/repo）；`(authenticated_user_id, request_id)` 唯一。请求指纹绑定版本、room、实际持久内容，同键同内容返回原持久结果，同键异内容 409；不同用户同键独立。消息、请求记录、不可变事件快照同一 SQLite 事务提交，任一步失败全部回滚。并发由数据库唯一约束裁决，SQLite busy 有界重试/可观察失败，不能依赖进程全局锁掩盖竞态。
3. **稳定回执与删除墓碑**：server_message_id、event_id、持久化时间由 backend 生成且重试保持；temp_id 只用于 pending 匹配。消息删除不应删除幂等墓碑或让旧请求重建消息；重放仍校验当前权限，再返回稳定结果或明确已删除终态，记录选择。不清理旧幂等记录，不伪造旧消息 request_id，不为历史数据批量产生广播。
4. **统一 outbox 事实出口**：允许提交后通过原即时发布路径尝试投递，但必须来自已提交的同一 outbox 快照；成功后才标记发布，失败保留 pending，不回滚已经持久的消息。REST/Kafka 不保留绕过 outbox 的第二个事件构造器。发布成功但标记前崩溃允许重复，固定 event_id 不变。后台扫描/重启补发由 S2.2 实施，本轮不声称 pending 会自动恢复。
5. **加法协议与兼容**：按 ADR 新增 request_id/event_id/server_message_id 等字段，不复用旧字段号/Id/TempId。修改源 proto 后生成四份产物，TS 64 位字符串保持，HTTP 新回执避免精度丢失。旧合法非空 temp_id 可映射 request_id；完全无标识的旧入口分配一次服务端 ID 并明确不能保证客户端重试去重。新客户端发送稳定随机 request_id，重试复用；本轮只做协议必要兼容，不扩成完整 ACK/upsert/重连视图重构。
6. **尺寸准入**：以 ADR 的 256 KiB UTF-8 content、最多 128 ASCII 字节输入标识为候选，实测 protobuf/Kafka header/HTTP JSON+base64 envelope 后确定统一预算。REST 超限 413，WS 在排队前明确拒绝（错误协议未就绪时 1009）。room_id 非零路径省略不参与过滤的 user_ids，但仍按 registry 选择网关，每网关每事件一次；验证大量接收者不会让 envelope 无界增长，禁止分批 user_ids 却反复广播全房间。gateway body 超限返回明确 413，不用截断 JSON 400；不扩大 room_id=0 的 ID fallback 语义。
7. **迁移与文档**：新增表/字段/索引提供显式版本和兼容/回退说明，在新库、旧 schema 的临时数据库副本验证。禁止打开或升级实际 backend/mydb.sqlite/生产数据。更新 ADR 中精确的已实施范围与 proposed 后片、contracts/configuration/testing/status/模块指南和本计划。

### 验收关卡

- 同请求串行和并发重试只有一条业务消息、一条请求记录、一条 outbox，返回相同 ID/event/time；不同用户同键隔离，同键异内容 409。
- 请求/消息/outbox insert 或 commit 故障全回滚；提交后 producer 故障仍保留 pending 和可查询持久结果；响应丢失重试不重复写库。删除后旧 request_id 不重建消息。
- 未认证、撤销、非成员或冒用身份不写入；重试读取仍执行当前授权；已有用户/房间/fanout 认证不倒退。
- 新/旧协议的 request_id/temp_id/服务端 ID 语义明确，Kafka 摘要签名覆盖实际发送的新事件；四份生成消费者兼容，HTTP/TS 不丢 64 位精度。
- 边界大小、UTF-8/转义/base64 余数、最大标识/varint、大量接收者均覆盖；每个被选择网关只投递一次，token/命令秘密不进 outbox 或通知。
- 新库与旧 schema 兼容只在临时库验证，无旧消息重播/归属猜测；数据库唯一约束和 SQLite busy/重试在真实并发事务中验证，不能只测 mock。

### 测试与交付

至少运行三个 Go 模块测试。协议/前端影响必须运行前端测试/生产构建；WS/并发影响运行相关 connection race，backend 新事务并发测试也按需启用 race。执行文档链接、生成 diff 和 git diff --check；测试使用隔离密钥、内存/临时 SQLite 与 mock producer，不依赖真实 broker/Redis。工具缺失和外部环境限制分别记录，不能以跳过断言或重写旧安全测试掩盖回归。

S2.1 结束须列清：本轮实际实现及测试、S2.2 pending 后台恢复缺失、S2.3 ACK/同步接纳未实施、S2.4 room_seq/分页补拉未实施。过期或未持久接纳的 Kafka 命令、原有消费推进与 fanout 丢弃语义继续作为限制，除非另有完整方案/验证；不能宣称端到端 exactly-once、可靠 delivered 或完整 S2 已完成。

保留 SQLite、三份独立 Go module、现有 React/CRA 和认证方案；不引入 PostgreSQL、CDC、Kafka 事务、角色/邀请系统或新部署。账户管理/邀请规则未决不阻塞本片，也不捆绑改变。新的 commit/push、部署和真实数据迁移未授权；代码、临时库迁移逻辑和本地验证可直接执行，无需重复确认。

## 2026-10-29 S2.1 实施记录

状态：本片本地代码与验收完成。工作区保留交接前的计划修改及本轮实现，未 commit/push；没有打开实际 backend/mydb.sqlite，没有部署或真实数据迁移。

- REST/Kafka 统一调用 SendMessage，同连接 SQLite 事务复查 session/成员，messages/send_requests/message_outbox 一起提交。用户/请求复合主键裁决并发，不含 session；版本 1 指纹绑定 room 和实际内容。同键同内容返回原 ID/时间，异内容 409，跨用户独立；重登可重试，撤销/退房后拒绝读取。删除保留墓碑，返回 deleted=true，不重建。
- Outbox 持久化一次生成的事件 ID、protobuf 快照/schema_version 与发布状态。提交后即时 publish 使用该快照、room key，成功确认后标记；producer/mark 失败保留 pending，重试使用相同字节。无后台扫描/重启恢复。
- protobuf 加法字段 8–11 为 request_id/event_id/server_message_id/persisted_at，四份产物按现有脚本和锁文件重生成；TS 头同步至已锁的 2.11.5，没有升级依赖。命令摘要覆盖实际新字段；HTTP receipt/history 新增精确字符串 ID。新客户端为逻辑 payload 生成随机 request_id，重试复用，temp_id 仍匹配 pending；旧 temp_id 和无标识路径按 ADR 兼容。
- 共用 UTF-8 256 KiB/128 字节可见 ASCII 准入验证实际 protobuf、header/record 和 JSON/base64 包装。最大内容实测 protobuf 262529、Kafka record 266695、HTTP envelope 351395 字节（最大 ID/varint 与最坏转义标识），低于 1 MiB。REST 超限 413，WS 入队前 1009，gateway body max+1 明确 413。十万接收者验证 registry 选择的每个 gateway 每事件一次；非零 room body 省略无效 user_ids，fallback 未扩展。
- 显式消息 schema v1 与新/旧临时库验证通过；不为旧消息伪造 request 或 outbox、不历史重播，未来版本/不完整 schema 拒绝。回退保留事实表并停旧写入口，见 [迁移说明](message-persistence.md)。

真实 COMMIT 失败回归先发现当前驱动/sql.Tx 会遗留活动 SQLite 事务；已用固定连接的显式 BEGIN/COMMIT/ROLLBACK 修正，清理失败丢弃连接。回归覆盖三处 INSERT 和延迟 FK 引发的实际 COMMIT 失败全回滚；24 路并发、WAL 写锁/rollback-journal COMMIT 锁下 busy 明确可重试（临时封顶并还原该连接 busy_timeout）、响应丢失、重登/撤销/删除墓碑、producer 与发布标记失败、REST/Kafka 共用键、64 位精度及尺寸边界均通过。完整命令见 [测试记录](testing.md)。

S2.2 relay/恢复、S2.3 同步接纳/persisted ACK/完整 upsert、S2.4 room_seq/分页补拉保持未实施。S2.1/S2.2 是同一可靠性发布门槛；当前不宣称 pending 自动恢复、端到端 exactly-once 或可靠 delivered。旧 30 秒命令过期、原失败消费推进/fanout 丢弃、用户/client fallback 和 UI 不追加其他用户新消息限制仍在。真实 Kafka/Redis/浏览器完整链路未验证。

## 2026-10-29 长任务：连续完成 S2.2–S2.4

用户已明确要求“开一个长任务一起跑完”。本轮授权替代此前“每次只完成一个 S2 切片”的停止点：新建一个实施 chat，保留当前全部未提交 S2.1 成果，连续完成后台恢复、同步持久化 ACK/前端幂等、房间顺序/分页与补拉。本轮范围是剩余 S2，不扩为 S3 presence 重构、S4 集群/容量、角色邀请、数据库替换或生产运维。

起点：当前 HEAD `616b9b1`；S2.1 实现、测试、四份生成物及迁移文档在本地工作区尚未提交。上一 S2.1 chat 已结束。规划 chat 完成交接后不并行编辑。只增加本节计划不代表实现完成，执行 chat 应逐项更新真实检查点。

### 不变量与工作方式

- 复用 S2.1 的请求幂等/删除墓碑、稳定 event/message/time 与同连接事务回滚。保留已验证的 SQLite COMMIT 失败清理、busy timeout 有界等待和真实并发约束，不能为新 worker 简化掉这些边界。
- 保留 Principal/session/永久成员、创建者删除、WS 首帧认证、独立 FANOUT_TOKEN/body 前鉴权与禁止重定向。服务凭据不替代接收者授权，任何秘密不进入通知、outbox、浏览器 URL 或日志。
- 明确持久化确认与客户端送达不同。目标是业务事实幂等、至少一次事件发布、客户端去重与权威历史恢复；不承诺网络端到端 exactly-once，不把 HTTP 204/Hub 入队称为 delivered。
- 按数据库/协议/客户端依赖组织实现，可提前做后片需要的 schema/类型，避免人为顺序制造不一致；仍保留下面三组验收检查点。不要每完成一片就结束任务或再要求新开 chat，压缩上下文后依据代码、计划和测试继续。
- 每组工作建立行为与故障回归、完成最小一致实现、运行受影响检查并做一次聚焦自审。临时环境缺工具时完成不依赖它的实现与验证，单列未验证项；核心功能未做不能只以环境限制结题。

### L1 · S2.2：后台 relay 与进程恢复

1. pending 有界扫描、可持久化领取/租约或等价单 relay 所有权、重试退避、失败分类/可观察状态。坏快照/不支持版本需隔离而非永远阻塞整批；不把失败记录直接标成功或无记录丢弃。
2. 每次发布使用原 event_id、room key、持久化快照。broker 确认后更新状态；publish 成功/mark 前崩溃可重复但不生成新业务消息。即时 publish 与后台 relay 共用领取/发布协调或去掉独立快路径，不保留两个无协调发布器。
3. 启动恢复 pending，producer 不可用可重试；把 relay 生命周期与 HTTP 应用装配清晰接线。尤其避免当前 SetupRouter 创建 producer 失败导致 broker 故障时 HTTP 接纳无法启动；业务提交不应依赖 notification broker 当时可用。后台连接、发送和退出都有有界等待/cancel，不因坏依赖无限阻塞关机。
4. 明确“消息删除但原通知 pending”的处理：请求墓碑保留；不得通过补发重建已删除业务消息。记录对 pending 的抑制/墓碑策略及已在途通知的视图一致性边界，结合 L3 的权威历史恢复处理。旧快照不能无记录被改写成另一事实。
5. 首版范围是 SQLite 单 writer/单 relay 模式，租约仍覆盖重启和重叠退出；不能宣称多 backend 共享数据库能力。提供合理配置及日志/统计，避免同时做完整监控平台。

验收：提交后退出/重启能续发；broker 故障恢复后续发；publish 后 mark 失败/崩溃重发同一事件；过期领取能恢复；坏记录不饿死健康记录；正常停机不泄漏 goroutine/连接；消息删除不释放请求键。通过真实临时 SQLite + 可控 producer/时钟验证，不只测试内存 mock 状态。

### L2 · S2.3：同步接纳、persisted ACK 与客户端 upsert

1. gateway 的默认 message 路径调用可信 backend 接纳用例，backend 完成授权与事务后返回可关联回执，WS 此后才发 persisted ACK。不得在 gateway 排队、Kafka producer 成功或签发 token 时提前确认。
2. 不让同一默认请求同时走旧 user-request 和同步入口。旧 Kafka 命令入口可保留受控/显式兼容策略，但保留摘要、audience、过期与 session/成员校验；不能单纯延长 30 秒期限或丢弃鉴权解决 backlog。默认 WS 路径不再因命令在 broker 排队过期而丢失未确认结果。
3. ACK/错误协议绑定 request_id/server_message_id/event_id/持久时间，L3 后能带 room_seq；HTTP/WS 的终态、可重试、结果未知语义一致。401/403、409、413 可识别；503/连接断开/响应丢失保持未知或可重试并复用原 request_id。
4. 前端按持久 ID/event_id upsert，temp_id 仅确认本人 pending；加入其他用户的新消息，重复 ACK/通知/历史不重复。ACK 先到或通知先到均正确，非当前房间事件不污染视图。64 位 ID/seq 传输为字符串，不经过 Number 舍入。
5. pending/重试状态按用户和逻辑请求隔离，在重连/重新登录时保留可重试 ID；切换账号不得发送上个账号的请求或使用旧 token。自动重试有界并可取消，明确页面刷新支持边界；必要的本地存储不存凭据，退出/切换用户行为有测试。

验收：事务已提交但 ACK 丢失后重试只一条消息；未提交超时可重试；通知/ACK 乱序和重复正确；远端新消息追加；未 OPEN 不伪装发送成功；重连/重登、拒绝/撤销/非成员、backend 5xx、旧回调/账号切换有回归。增加本机真实 HTTP + SQLite backend 应用用例 + gateway WS 的跨层验证，不把所有关键边界都 mock 掉。

### L3 · S2.4：房间序号、历史游标与补拉恢复

1. room_seq 在消息事务内按房间分配，持久化 `(room_id, room_seq)` 唯一约束；与请求幂等、outbox 快照同一提交，重试不再分配新序号。删除允许有洞，不重排；时钟、Message.ID 或 Kafka offset 不冒充业务房间序。
2. 序号、server_message_id、新协议字段从 schema 源加法演进并重新生成四份产物；更新 admission 最大 envelope 验证，不复用现有字段号。旧通知/快照 schema v1 兼容显式处理：不得悄悄重写不可变旧 outbox，也不能让旧 pending 永远卡死。无法含 seq 的兼容通知触发权威补拉或等价明确策略。
3. 历史支持 before_seq 与 after_seq、同快照高水位、next_cursor/has_more；旧 before_id 暂兼容，不混用新旧游标；限制最大页长，排序/索引/游标一致，所有分支检查当前 session/成员。序号和游标校验拒绝非法范围，不能从查询参数绕过授权或舍入。
4. 客户端重连、重新入房及序号缺口时按高水位有界追平，多页恢复不会漏掉离线期间超过一页的消息；乱序/重复事件合并并按持久序排序。删除造成的洞不被误判成必须无限等待的消息，高水位能在尾消息被删后仍正确推进。
5. 覆盖“当前已连接、最后一条通知丢失，没有下一条事件”的恢复触发；若提供自动恢复，用有限频率的高水位/聚焦校验等明确机制，不只依赖下一条通知发现 gap。历史加载期间实时回流、旧房间/旧账号的迟到响应须取消/忽略，不能用 LOAD 覆盖已确认新消息。
6. 对游标过旧、清理/重置返回可识别结果；没有自动清理策略时明确支持边界，不编造已实现保留期。已删除消息不应因晚到快照/历史请求被永久复活；明确权威历史对账范围，不扩成新的已读/编辑产品。
7. 显式版本化 v1→后续迁移，在临时旧库按房间稳定 ID 回填历史规范序，不能声称恢复旧真实发送顺序。不伪造旧请求、不新增历史 outbox 或广播；已存在请求墓碑、pending/published、CreatorID/membership/旧行内容时间均保留。新库、旧库、重复初始化、未来/损坏 schema 拒绝及回退须验证；不可只改 MessageSchemaVersion 跳过旧版步骤。

验收：并发/重试/回滚仍只一条持久事实及稳定 room_seq；跨房间独立，旧数据/旧快照可处理；相同时间戳、大于 JS 安全整数的 ID/seq、分页并发发送、删除洞、乱序重复、离线多页、尾通知丢失、切房间/账号迟到响应均有实质测试。所有迁移仅临时文件/副本。

### 最终集成、完成定义与权限

本任务必须连续完成 L1–L3 的代码、兼容接线和本地验证，不能只完成 relay 再把其余列回“后续”。每个检查点更新本计划/ADR；最终汇总一次源码自审，关注事务与迁移的部分变更、旧快照、过期/撤销、重复发布、ACK 顺序、游标高水位和 UI/数据库不一致，修复后重跑受影响检查。

至少运行三个 Go 模块测试、前端测试/构建/lint、受影响 backend/connection 的 race、生成产物与 admission 副本一致性、文档链接、git diff --check。保留 S2.1 回归（真实 COMMIT 失败/锁等待/24并发等），不移除断言以适配新代码。提供可重复本机跨层故障验证；真实 Kafka/Redis/浏览器若当前环境可用，只在隔离临时配置/数据中验证。若 WSL Docker 不可用，提供清晰复现步骤并把该层列为未验证，不宣称生产/真实完整链路已通过。

当前授权包含必要本地代码、schema/协议演进、临时数据库迁移及自动化故障验证；不包含 commit/push、生产/项目实际数据库操作、发布、部署集群、默认安装基础设施、PostgreSQL/CDC 替换或新角色/邀请规则。不启动默认 main 触碰实际 mydb.sqlite。保持三个 module 与 React/CRA，不另开 chat 或自动拆 subagent。长任务完成时明确实现/测试/环境限制，不以 checklist 已写、token/context 不足或外部工具缺失冒充功能完成。

## 2026-10-30 S2.2–S2.4 连续实施记录

状态：**L1、L2、L3 本地实现与隔离验证全部完成**。使用原本地checkout，保留全部未提交S2.1成果；未commit/push，未另开chat/worktree/subagent，未打开实际数据库、部署或安装基础设施。前面的S2.1停止点和proposed说明是分轮历史，本节与[当前ADR](decisions/0002-message-persistence-and-recovery.md)描述最新实现。

| 检查点 | 实际交付与证据 |
| --- | --- |
| L1 / S2.2 | 持久pending有界扫描/条件领取/30秒租约/token标记、尝试与指数退避、坏快照/版本隔离、删除抑制、v1原字节兼容；请求内生产即时发布移除。router不连接broker，lazy producer、relay、可选consumer与HTTP/数据库生命周期接线。recovery_test覆盖新relay恢复、broker恢复、mark失败原字节重发、过期领取、旧owner fencing、坏记录不饿死健康记录、取消；runtime/producer测试验证不可用依赖仍接纳及有界关闭 |
| L2 / S2.3 | 默认gateway同步POST既有/api/messages，后端授权事务提交后才persisted ACK；同一请求不写user-request。send_error区分确定拒绝与unknown，校验回执身份/房间/请求/时间/序号；旧Kafka入口默认关闭但保留30秒/audience/摘要/成员/session边界。前端稳定ID/event/request upsert、本人temp fallback、远端追加、5次同键重试/手动重试/取消、同账号内存重登恢复与切账号清空；真实HTTP+临时SQLite+WS跨层故障实验通过 |
| L3 / S2.4 | room_sequences高水位与消息/请求/outbox同事务，正(room_id,room_seq)唯一；before/after/snapshot_seq、同快照高水位/next/has_more、最大100和旧ID兼容/混游标拒绝。客户端10页一批/1秒续跑、固定上界多页补齐、按seq BigInt排序、15秒/聚焦/重连/通知对账，尾通知丢失与删除视图恢复；旧房间/账号迟到响应隔离。0→1→2显式迁移与准确v1原DDL测试、失败回滚、v1 pending原字节不变；不造历史广播 |

最终[验证记录](testing.md)：make test-go三模块通过；backend受影响包race通过；connection全相关包race（启用跨模块实验）通过；frontend 4 suites/18 tests、build/lint/严格tsc通过；四份生成物/三份准入与源逐字节比对、文档链接和diff whitespace通过。保留原S2.1真实COMMIT/锁等待/24路幂等/尺寸/安全测试，并新增24路不同请求唯一房间序。

聚焦自审已修正：ACK的错误/未关联回执不能确认；意外3xx/204保持unknown；关channel与ACK/广播enqueue共用锁；HTTP Shutdown完成后再关闭relay/库；relay SQLite语句也应用busy cap，慢批次逐条更新领取时间、退避从尝试结束起算；历史请求返回request_id消除lost ACK后的pending；不同request_id不能因相同temp碰撞；schema同名非唯一索引/缺版本历史拒绝。故障夹具日志buffer race已修复，全部原恢复断言保留并重跑。

环境与支持边界：Docker检查仍为WSL未启用Desktop集成。跨模块实验真实HTTP/WS/JWT/临时SQLite/业务事务/relay/fanout dispatcher/source；Kafka传输与Redis registry替换，另有本机Sarama协议模拟器，均不计作真实Kafka/Redis/浏览器成功。schema升级/回退说明见[迁移](message-persistence.md)，真实库未操作；旧v1快照可发布但不改字节，旧writer不可直接切回写入。Kafka旧consumer仅CHORDS_LEGACY_COMMANDS=1启用，没有自动到期窗口；升级需显式停旧writer/关闭兼容。

支持单SQLite writer/单relay，业务事实幂等与至少一次发布；fanout仍best-effort/失败推进，依靠具体的权威历史恢复触发补齐。pending只在当前页面JS内存，最多128条/5次自动发送，刷新/关标签不继续未接纳请求；取消不撤销已提交事务。后台标签节流/离线可延迟对账，无自动数据清理/保留期，不承诺delivered/exactly-once。S1b未决账户/邀请、S3多网关presence/完整draining、S4部署容量保持原范围，不把这些限制误列成未实现的S2核心功能。
