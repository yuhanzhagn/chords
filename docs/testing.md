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
