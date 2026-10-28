# 架构整改实施计划

日期：2026-10-28。依据 [架构审计](architecture-audit.md)，代码基线为 `08c2f3a`。当前审计报告与文档索引尚未提交。用户已授权规划，并新建一个 chat 开始实施；该授权不包含新的 commit/push、部署、生产数据操作或依赖/框架迁移。

## 执行方式

保留 backend、connection、fanout 的职责及独立 Go module。按依赖顺序完成下面里程碑；每个工作项采用“复现 → 安全/正确行为回归 → 最小一致修复 → 相关检查 → 一次聚焦自审”。以实际代码和验证更新状态，不能仅因为写好计划或旧探针 PASS 就标记完成。

新 chat 使用本项目现有本地工作区，确保可读取未提交审计与本计划；本 chat 完成交接后不再并行编辑。先完成 S1a，再推进 S1b。其余阶段作为后续路线，不自动扩大本次 chat 为数据库迁移或基础设施部署任务。

## 阶段总览

| 阶段 | 审计项 | 工作与交付 | 前置/完成关卡 | 状态 |
| --- | --- | --- | --- | --- |
| S1a：账户与启动安全 | A02/A03/A04 | 修复错误登录撤销、DTO/密码处理、正常启动 FLUSHDB；回归与文档 | 错误密码不改变会话；用户响应无密码/哈希/秘密；所有创建入口哈希；启动不清共享状态 | 已完成本地代码与回归；见 2026-10-28 实施记录 |
| S1b：身份与业务授权 | A01 及 A02 相关凭据边界 | Principal、REST 资源授权、WS 身份/订阅授权、前端凭据兼容；短 ADR | 无用户冒用；无匿名业务订阅；权限决策和兼容性明确 | 部分完成；成员/创建者删除规则已实施，账户管理/邀请仍未决，真实链路未验证 |
| S2：持久化与消息协议 | A05/A06/A07/A11 | 统一发送用例、幂等请求、事务 outbox/relay、确认语义、房间顺序/游标、消息尺寸预算 | DB/broker 故障和重复请求下持久事实可恢复；已提交消息有稳定服务端 ID；边界尺寸入口处理 | 待实施；依赖身份边界 |
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
