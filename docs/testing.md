# 测试与检查

## 按变更范围选择

| 变更 | 根目录命令 | 边界 |
| --- | --- | --- |
| 文档与指南 | make check-docs | 本地内联 Markdown 链接存在性，不验证外网/锚点/语义 |
| Python / 辅助脚本语法 | make check-python | ast.parse，不 import、不创建 pyc、不启动服务 |
| backend | make test-backend | 单元与分层集成，内存 SQLite/mocks，无需真实 Redis/Kafka |
| connection | make test-connection | handler、sink、gateway、HTTP source、装配函数等；不连接真实依赖 |
| fanout | make test-fanout | HTTP 请求/凭据/状态码/redirect、mock registry、配置/真实入口失败顺序；无真实 Kafka/Redis |
| 前端 | make test-frontend / make build-frontend | 单次 Jest 冒烟 / 生产构建；先 make install-frontend |
| 跨模块或 schema | make test-go，加前端 build/test | 检查四个消费者；无法代替真实消息链路 |
| 全仓基线 | make check | 不自动安装依赖；汇总以上关键检查 |
| 不写二进制的 Go 编译 | make build-go | 各模块 go build ./... |

Go 单模块命令也可直接在相应目录执行 go test ./...。精确选择包或用 `-run TestName` 定位；有新变更、失败或未解决疑点时才重复/扩大检查。

修改网关并发/连接生命周期时按需运行：

```bash
cd connection
go test -race ./cmd/server ./internal/gateway ./internal/sink
```

frontend 另有 `npm run lint` 和 `npx --no-install tsc --noEmit`。不默认运行 npm test 的 watch 模式；根 Make 已固定 CI=true、watchAll=false、runInBand。

## 测试环境说明

App 冒烟测试 mock 启动时的 token 检查 fetch，不依赖 backend。CRA 使用 Jest 27，无法按当前 protobuf 包 exports 直接解析 wire 子路径；package.json 的 Jest moduleNameMapper 指向已锁 runtime 的 CJS 文件，运行时构建仍使用正常 imports。升级 protobuf/CRA 时核对此映射。

Python 暂无 pytest 套件。新行为测试应覆盖空库、重建、重载、更新、删除与停用词边界，使用临时数据路径。ast.parse 成功不能说明依赖安装齐全或运行行为正确。

## 2026-10-27 实际验证

环境：Linux/WSL、Node 18.19.1 / npm 9.2.0、Python 3.12.3。宿主 go version 是 1.22.2，module 命令自动选择缓存中的 Go 1.24.6 / 1.24.12。

| 检查 | 结果 |
| --- | --- |
| make check / make build-go | 全部通过；汇总检查与不写二进制的构建入口均验证成功 |
| backend go test ./... | 通过，包括 service/controller 与两组分层集成 |
| connection go test ./... | 通过 |
| fanout go test ./... | 成功，无测试文件 |
| npm ci --legacy-peer-deps --no-audit --no-fund | 按锁文件安装成功，未修改依赖声明/锁文件 |
| npm run build | 通过；存在 Browserslist 数据过期提示 |
| npm run lint | 通过 |
| Jest 非交互 | 通过（1 个 App 冒烟测试）；本次修复 protobuf 子路径映射及离线 fetch mock；保留 React Router future flag 提示 |
| kubectl kustomize deploy/k8s/base | 离线渲染成功，不代表集群运行 |
| docker compose config --quiet | 未成功：本机 WSL 缺少 Docker Desktop 集成；没有启动服务 |
| 协议脚本生成到临时目录 | 成功；三份 Go 产物一致，TS 只有生成器头 2.11.2 → 2.11.5 差异；仓库产物未改 |
| Python 服务运行 | 未执行，本机缺 sklearn；未安装服务依赖 |

本次文档链接检查通过（21 个 Markdown 文件、152 条本地链接），Python/辅助脚本语法解析通过（7 个文件）；链接检查器也验证了缺失目标返回失败与忽略 fenced code 示例的行为。CI/Compose YAML 语法解析通过。完整 Compose、kind 集群、Python API、负载和端到端投递未验证；不使用本页推断运行环境已经配置完成。

## 手工端到端验收

使用 [隔离演示](development.md) 注册并登录两名用户，创建/加入房间，选择房间发送，检查发送者 pending、接收者新消息、历史分页、断线重入和其他连接存活。按 gateway → Kafka → backend → fanout → gateway → UI 查每跳证据。已知未完成项见 [现状](status.md)。

CI 当前运行三个 Go 模块测试、前端测试/构建，以及文档链接/Python 语法；Python 行为、完整部署链路仍无自动覆盖；fanout 已在下一轮增加 HTTP 分发与服务凭据行为覆盖。

## 2026-10-28 S1 实施验证

本节保留上一轮的实际结果；后续 N1/N2/N3 的新增认证/房间政策与验证见本页末尾。

| 检查 | 结果与实际边界 |
| --- | --- |
| make test-backend | 通过；全部 backend 单元/分层集成，包括新增内存 SQLite 账户与资源授权回归 |
| make test-connection | 通过；真实本机 WS + 模拟 backend 的认证、消息 pipeline、首帧超时、身份不可变、成员拒绝、撤销投递、暂时不可用场景 |
| cd connection && go test -race ./cmd/server ./internal/gateway ./internal/sink ./internal/adapter/kafka | 通过；没有真实 Kafka/Redis |
| make test-fanout | 成功；仍无行为测试，只证明编译 |
| make test-frontend | 通过，2 个 suite、5 个测试；App 冒烟和 WS auth/ready、未认证发送、1008 停止重试、凭据切换旧回调 |
| make build-frontend / npm --prefix frontend run lint | 通过；保留既有 Browserslist 数据过期与 React Router future flag 提示 |
| make check-docs / git diff --check | 通过；本地链接与 whitespace 检查，不验证文档内外部命令或网络 |
| kubectl kustomize deploy/k8s/base | 离线渲染通过，含外部 JWT Secret 引用和 gateway 认证地址；未 apply |
| JWT_SECRET=configuration-validation-placeholder-only docker compose config --quiet | 未执行成功：当前 WSL 仍无 Docker Desktop 集成；没有启动依赖或重置数据库 |

新增 [账户回归](../backend/test/controller_service_test/account_security_test.go) 首次在修复前失败：错误密码后原 token 的 GET /users 返回 401，预期 200。修复后通过，并验证未知用户名/非法输入、成功替换、两入口单次 bcrypt、73 字节密码拒绝、模型内部字段隔离与响应无秘密。[事务回归](../backend/internal/service/auth_atomic_test.go) 覆盖签发失败、SQLite trigger 模拟撤销/插入失败的回滚、并发登录只保留一个有效 session、过期与旧 token 登出不影响新 session。

[资源授权回归](../backend/test/controller_service_test/authorization_security_test.go) 用真实 token 和持久化行验证：非成员全部历史分支/WS 授权拒绝、加入自己成功、不能伪造他人 membership/发言/删除、作者可删除；命令缺凭据、access/command 混用、篡改、撤销及成员变化均拒绝。命令验证使用原始未被 notification 时间更新污染的副本，成员删除和 session 撤销分开测试；数据库故障返回 503。[Kafka adapter 回归](../connection/internal/adapter/kafka/event_sink_test.go) 验证授权 header。共享 protobuf 未改动，无生成操作。

正常启动不再调用 FLUSHDB 由 main/redis 初始化路径自审确认；[Redis 初始化失败测试](../backend/internal/redisdb/redis_test.go) 验证返回 nil/error 而非 panic。没有启动完整 main、真实共享 Redis/Kafka，也没有执行浏览器端到端、部署、迁移或压测。该轮尚未确定的房间删除政策已由后续 N3 落实；其他未决权限与短效命令 backlog/逐接收者 RPC 成本见 [ADR](decisions/0001-identity-and-room-authorization.md)。

## 2026-10-28 N1/N2 实施验证

| 检查 | 结果与实际边界 |
| --- | --- |
| make test-connection | 通过；新增 body 读取前认证、独立服务头、错误用途/重复头拒绝、有效请求原有 405/400/204 行为、配置校验/保密及真实 main 子进程启动失败顺序 |
| make test-fanout | 通过；已有实质行为测试：真实 httptest HTTP、mock registry、准确头与 JSON payload、4xx/5xx、301/302/303/307/308 一律拒绝跟随、响应秘密不进错误、构造器/配置与 main 失败顺序 |
| make test-backend | 通过；先确认 S1 身份边界无回归，N3 追加后再次通过全模块测试 |
| cd connection && go test -race ./cmd/server ./internal/gateway ./internal/sink ./internal/source ./internal/adapter/kafka | 通过；含 HTTP fanout→真实本机 WS 的跨层场景，无真实 Redis/Kafka |
| kubectl kustomize deploy/k8s/base | 离线渲染成功，输出到 /tmp；gateway 引用独立 chords-fanout Secret；未 apply，仍无 fanout Deployment/8082 Service |
| JWT_SECRET=offline-validation-placeholder-32-bytes FANOUT_TOKEN=offline-fanout-validation-placeholder-32-bytes docker compose config --quiet | 未成功：当前 WSL 仍提示缺少 Docker Desktop 集成；未启动服务 |
| Python/PyYAML 离线断言 | 通过；Compose 只有 connection/fanout 接收 FANOUT_TOKEN，且必填；渲染清单 Secret 引用准确；YAML 无 token 值；不代替 Compose 插值或实际启动 |
| make check-docs / git diff --check | 通过；最终本地链接与 whitespace 检查，未验证外网或运行环境 |

[首个回归](../connection/internal/source/http_source_test.go) 在修复前失败：缺失/错误/其他用途凭据请求读取了 body，返回 400（预期读取前 401）。修复后通过。有效专用服务凭据之外，用户 Authorization header 或 access/command 字符串不能代替；两个模块从环境读取同一值，构造器也拒绝空/短/非法字符，不设匿名 fallback。配置日志不再整体打印，返回对端错误仅含数值状态码。

[本机跨层回归](../connection/cmd/server/fanout_integration_test.go) 使用生产 WS mux/pipeline、HTTP source、Hub、protobuf codec 和真实本机 socket，验证：合法服务请求可向认证成员投递；匿名/用户 token push 返回 401；非成员无法订阅；成员移除或撤销 session 后合法服务 push 也不会泄漏消息。backend 授权为模拟服务；worker 端另用真实 HTTP 接收端与内存 registry 验证，不含真实 Kafka/Redis/浏览器 UI 完整链路。

本轮 N1/N2/N3 无前端源码变更，未重跑前端检查，保留前轮结果；没有修改 event.proto、生成产物、依赖声明或历史二进制。短效命令 backlog 丢弃、fanout 失败仍标记消费、广播路由和用户/client ID fallback 限制仍在，见 [S2 proposed 设计](decisions/0002-message-persistence-and-recovery.md)。

## 2026-10-28 N3 创建者删除追加验证

用户在原规划 chat 明确选择仅创建者删除后追加实施，不再保留为待选择。真实路由/服务/SQLite [回归](../backend/test/controller_service_test/chatroom_ownership_test.go) 初次在旧实现失败：创建响应没有可信 CreatorID。实现后验证新建归属不受 body 的 CreatorID/creator_id/ID 影响、创建者可删（不要求成员）、其他成员/非成员和未知旧归属拒删、匿名/撤销拒绝、缺失/非法路径状态码正确。

[服务回归](../backend/internal/service/chatroom_ownership_test.go) 验证绕过 middleware 调用时仍检查缺失/撤销/过期 session、NULL/0 归属拒删、认证 DB 错误不删房间，以及读后归属变动时条件 DELETE 不误删。追加的归属变动测试通过 `cd backend && go test ./internal/service -run 'TestChatRoom(Mutations|DeletionChecks)' -count=1` 聚焦运行。

[迁移回归](../backend/internal/app/db_ownership_test.go) 在临时文件 SQLite 上运行真实 InitDB：全新 schema 和旧 chat_rooms schema 均通过，可空列/索引存在；旧房间 ID/name/时间与 membership 保留、creator_id 仍 NULL；新建归属正确，重复 InitDB 不改变已知或未知归属，新创建者可删。`make test-backend` 全模块通过。仅临时数据库执行了迁移验证，没有操作实际项目/生产库；真实升级与可审查离线归属处置见 [说明](room-ownership.md)。房间级联清理、真实部署升级/回退和完整聊天链路未验证。

## 2026-10-29 S2.1 实施验证

| 命令 | 实际结果与边界 |
| --- | --- |
| make test-backend | 通过；含全部既有账户/房间/授权回归、新发送事务与临时新/旧 schema 测试 |
| make test-connection | 通过；真实本机 WS 准入/1009、签名新字段、HTTP body 明确 413，原认证/内部 fanout 边界保持 |
| make test-fanout | 通过；十万路由用户、两个网关，每网关一次且 body 不含 user_ids；fallback 和最终 body 超限也覆盖 |
| make test-frontend | 通过，2 suites / 8 tests；随机 request_id 及重发复用、UTF-8 超限、64 位 wire/JSON 与原 auth/ready 回归 |
| make build-frontend | 通过；保留既有 Browserslist 数据过期提示，未升级依赖；Jest 保留 React Router future flag 提示 |
| cd backend && go test -race ./internal/service -run 'TestSend\|TestKafkaAndREST' -count=1 | 通过；真实临时文件 SQLite/WAL、多连接，不是每连接独立 :memory: 或 mock 唯一键 |
| cd connection && go test -race ./cmd/server ./internal/gateway ./internal/sink ./internal/source ./internal/adapter/kafka | 通过；本机 WS/HTTP 与模拟 backend，未连接真实 broker/Redis |
| bash scripts/generate-proto.sh /tmp/chords-s21-proto-verify | 四份 protobuf 与工作区逐字节一致；三份 Go 文件相同；三份准入副本也与源/脚本输出相同 |
| make check-docs / git diff --check | 通过；只验证本地链接/空白，不验证外网或实际部署 |

[首批 REST 回归](../backend/test/controller_service_test/send_idempotency_test.go) 在旧实现中重试返回 201（期望 200）；[HTTP source 回归](../connection/internal/source/http_source_test.go) 在旧实现中超限返回 400（期望 413）。修复后验证原 request/server_message/event/time、异内容 409、删除后终态不重建、UTF-8/最坏 JSON 转义和完整 body 上限。

[事务回归](../backend/internal/service/send_message_test.go) 用真实 SQLite 24 路并发，最终各一条 message/request/outbox；测试直接重复 INSERT 证实数据库复合主键有效。分别以 trigger 注入三处 INSERT 失败，并以 DEFERRABLE INITIALLY DEFERRED FK + trigger 注入只在实际 COMMIT 才失败的约束，全部回滚。提交失败测试先暴露驱动/sql.Tx 留下活动事务；同连接显式 ROLLBACK 修正后，失败行不可见且能继续用同库发送。producer 失败/发布成功但 mark 失败仍有 pending，重试字节/event_id 完全相同；已 published 的普通重试不再发布。并发或 publish/mark 之间崩溃仍允许相同事件重复。

同套回归覆盖响应丢失、重登同键、当前授权拒绝、不同用户同键、旧 temp_id/无标识兼容、删除墓碑、SQLite WAL 写锁及 rollback-journal COMMIT 读锁下有界 ErrSendBusy（driver busy_timeout=5000ms；后者最初实测 5.35 秒超出预算，修正为事务连接临时封顶 50ms 并还原后通过）、REST/Kafka 共用请求和摘要覆盖新增字段，以及超过 JS 安全整数的回执/历史/通知一致。Outbox 只生成白名单事件，不携带命令凭据；没有第二套 notification 构造器。

[预算回归](../backend/proto/kafka/admission_test.go) 在 content 262142/262143/262144 字节、最大 uint32/uint64/int64、各 128 个 HTML 转义标识字符、4096 字节 command header 下，实测 protobuf 262527/262528/262529，Sarama record 266693/266694/266695，HTTP JSON/base64 envelope 351391/351391/351395 字节。测试同时断言当前锁定 Sarama 默认 record 上限为 1 MiB，检查 UTF-8 超限/非法字节、ASCII 标识、base64 余数；同策略供 REST/WS。worker 十万接收者不扩大 body，也不按用户分批重复房间广播。

[迁移回归](../backend/internal/app/message_schema_test.go) 只运行新/旧 schema 临时库：旧消息的 ID/内容/时间保留，新表无历史请求/outbox，重复初始化/旧字段读取兼容，未来版本/未版本化表/缺索引拒绝。真实数据库未打开，迁移/回退尚无实际环境证据；流程见 [说明](message-persistence.md)。

本轮没有真实 Kafka/Redis/浏览器端到端、部署或数据迁移验证。S2.2 后台 relay/重启补发、S2.3 persisted ACK/完整前端 upsert、S2.4 room_seq/补拉均未实施；pending 持久存在不代表可自动恢复。保留旧命令 30 秒过期、失败消费推进与 fanout 丢弃限制，测试通过不代表可靠 delivered。

## 2026-10-30 S2.2–S2.4 连续实施验证

保留前轮 S2.1 全部回归；本轮不启动默认 main，不读取项目实际 mydb.sqlite。测试库全部来自 t.TempDir/os.MkdirTemp，凭据为隔离测试值。没有 commit/push、部署或真实数据迁移。

| 实际命令 | 结果与覆盖边界 |
| --- | --- |
| make test-go | 三个独立 Go module 全部通过；包括原账户/授权、S2.1 24路幂等/三处INSERT/真实COMMIT故障/长busy_timeout和新relay/游标/迁移回归 |
| cd backend && go test -race ./internal/service ./internal/app ./internal/routes ./internal/kafka | 通过；恢复/分页与事务并发真实临时SQLite；完整service约170秒。补充迁移损坏/生产Sarama协议测试后另跑下一条 |
| cd backend && go test -race ./internal/service -run 'TestRelay\|TestSendOutbox\|TestKafkaAndREST' -count=1 | 通过；最终慢批次计时修正及原发布/标记/旧命令回归 |
| cd backend && go test -race ./internal/kafka ./internal/app -count=1 | 通过；lazy producer对本机Sarama MockBroker的拒绝/恢复/ACK/关闭，v2损坏索引/缺迁移历史拒绝。MockBroker是协议模拟器，不是真实Kafka集群 |
| cd connection && CHORDS_RUN_RECOVERY_PROBE=1 go test -race ./cmd/server ./internal/gateway ./internal/sink ./internal/source ./internal/adapter/kafka | 通过；含真实HTTP+临时SQLite+WS+relay+fanout的多进程故障实验、结构化ACK/错误和全部安全兼容回归 |
| cd frontend && CI=true npm test -- --watch=false --runInBand | 4 suites / 18 tests通过；upsert排列/大整数/临时ID碰撞、lost ACK重登、账号切换/旧回调、5次有界重试/拒绝/取消、多页补拉/尾丢失/删除/迟到历史 |
| cd frontend && npm run build && npm run lint && npx --no-install tsc --noEmit | 全部通过；无新增lint告警，保留既有Browserslist数据过期提示和Jest React Router future flag提示 |
| bash scripts/generate-proto.sh /tmp/chords-s2-final-generated | 产物与工作区四份逐字节一致，三份Go相同；三份admission副本与唯一源/暂存输出一致 |
| cd backend && go test ./proto/kafka -run 'TestSendAdmission\|TestAdditive' -v | 通过；新增field12最大int64 seq计入实编码，最大protobuf 262539 / Kafka record 266705 / HTTP envelope 351426字节，均低于1MiB |
| make check-docs / git diff --check | 通过；本地链接和空白检查，不代替运行环境验证 |
| docker version | 失败：WSL2提示未启用Docker Desktop集成；未启动Kafka/Redis，未安装基础设施 |

[Relay/序号回归](../backend/internal/service/recovery_test.go) 验证提交后新建relay即可续发、broker失败持久退避/恢复、publish成功但mark失败后租约到期原字节重发、旧owner不能标记新owner、慢批次按实际领取时间建立新租约/按失败结束时间退避、取消退出、坏版本/快照隔离不阻塞健康v1快照、删除抑制且请求键不释放；两组24路并发分别覆盖同键只一条与不同键房间序1..24唯一。跨房间独立、大于JS安全整数的seq、回滚不消耗序号、固定高水位多页、删除洞/尾删除、成员/session撤销均覆盖。

[迁移回归](../backend/internal/app/message_schema_test.go) 使用准确v1原DDL与旧行/墓碑/pending/published数据，验证0→1→2和1→2、重复运行、同时间戳按ID形成规范序、原内容/时间/快照字节与状态保留、不造历史请求/广播。注入版本2标记INSERT失败证实整个消息迁移（字段/表/索引/标记）回滚；未来版本、缺索引、同名非唯一序号索引、缺v1迁移历史拒绝。只验证读取旧字段和停止旧writer的回退边界，没有生产down migration证据。

[HTTP装配/停机](../backend/internal/routes/message_runtime_test.go) 在关闭的loopback broker端口下构造真实router、启动relay、提交消息并取消退出；[producer回归](../backend/internal/kafka/producer_test.go) 同时覆盖预取消与连接拒绝、协议ACK、关闭后禁止发送；relay数据库语句沿用同连接busy cap。生产main已等HTTP Shutdown完成后才关闭relay/数据库，不启动main触碰实际库。

[跨模块实验](../connection/cmd/server/recovery_integration_test.go) 用生产authenticator/codec/Hub/WS writer和[backend fixture](../backend/test/recoveryfixture/main.go)、[fanout fixture](../fanout/test/recoveryfixture/main.go)：

- 真实：本机HTTP与WebSocket、backend JWT/session/member校验、SendMessage/临时文件SQLite/room_seq/outbox、后台relay、fanout Dispatcher及专用服务头、gateway fanout source与逐接收者权限、真实REST游标历史。
- 替换：Kafka传输是可控HTTP测试broker，Redis路由是固定内存registry；另用Sarama MockBroker验证实际producer协议。没有真实Kafka/Redis或浏览器自动化。
- broker不可用期间仍得到已持久回执；事务提交后丢HTTP响应产生unknown，同键重试ACK seq=1，恢复broker后经真实dispatcher→source→WS收到相同event/message ID。
- 发送前超时没有提交，同键重试获得下一序号；409/413/503不被误当ACK。撤销/退成员得到401/403，不写库。模拟丢通知后连续65条，通过至少3页权威历史取回；前端另测试socket不断/无后续通知时15秒触发恢复。
- 首轮race暴露测试夹具stderr缓冲并发读写，改用锁保护后完整race通过。新增测试文件一度被补测脚本误写，已恢复全部原relay/历史断言并通过完整backend race；未删除S2.1断言或改成mock事务。

重复实验只需上表的CHORDS_RUN_RECOVERY_PROBE=1命令；fixture只绑定loopback随机端口，强制创建/删除临时库，stdin关闭退出，二进制写t.TempDir。不需要安装基础设施或配置真实凭据。默认go test跳过该显式多进程实验，但单模块业务/安全测试照常运行。

本轮结论是本地剩余S2功能已完成；真实Kafka/Redis/浏览器、生产迁移/回退与部署仍未验证。fanout失败仍MarkMessage、presence单网关/整Set TTL及gateway整体draining仍待后续；恢复依靠已实现的入房/重连/通知/聚焦/15秒权威分页对账，不承诺网络exactly-once或delivered。
