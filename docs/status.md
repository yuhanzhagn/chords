# 当前状态与后续工作

基于 2026-10-27 审计，更新至 2026-10-30 的剩余 S2 连续实施。下面区分已修复与剩余缺口。运行检查结果见 [测试记录](testing.md)。优先级表示后续修复次序，不替代实际任务范围。

详细架构审计及新增隔离复现见 [2026-10-27 审计报告](architecture-audit.md)，包含 REST 越权、密码哈希暴露、失败登录撤销会话、presence race、准入与消息尺寸缺陷。以下列表是项目待办索引，审计报告提供完整证据与整改关卡。

## P0：运行数据与身份边界

| 问题 | 源码证据 | 建议验收 |
| --- | --- | --- |
| 账户创建权限、跨用户邀请规则仍待产品确认 | [ADR](decisions/0001-identity-and-room-authorization.md) | POST /users 仍允许任意登录用户创建账户；未新增 admin/invite 模型。房间删除已按用户明确选择限制为创建者，未知旧归属拒绝删除 |
| broker ACL、可信传输与密钥运维未验证 | [fanout handler](../connection/internal/source/http_source.go) | 内部 fanout 专用服务认证已完成，非法 push 在 body 前拒绝；broker ACL、可信 TLS 和密钥运维仍需部署验证 |

S1a 已修复：失败登录不撤销原会话，成功登录事务替换；创建入口共享哈希路径，DTO/模型禁止凭据泄漏；正常启动不清 Redis，初始化错误先返回。S1b 已落实 Principal、本人 membership、成员历史/发送/订阅、作者删除、WS 首帧认证与逐次会话检查、Kafka 内容绑定短效授权，JWT_SECRET 由外部提供。N1/N2 补齐内部 fanout 服务凭据和本机跨层回归；N3 按用户选择实现房间 CreatorID，仅创建者删除；旧归属 NULL/0 拒删，临时旧 schema 升级回归通过，见 [归属策略](room-ownership.md)。详见 [验收与限制](remediation-plan.md)、[ADR](decisions/0001-identity-and-room-authorization.md)。S1b 保持部分完成，尚无真实 broker/浏览器端到端证据。

## P1：消息正确性、恢复与部署完整性

| 问题 | 源码证据 | 建议验收 |
| --- | --- | --- |
| fanout 失败仍 MarkMessage，无持久补发 | [consumer](../fanout/internal/kafka/consumer.go)、[dispatcher](../fanout/internal/fanout/dispatcher.go) | 当前 best-effort；丢通知由客户端入房/重连/通知/聚焦/每15秒高水位分页对账恢复；worker 持久补发仍未实施 |
| 顶层 room_id 优先，user_ids fallback 实际按 client ID | [applyFanout](../connection/internal/source/http_source.go)、[Hub](../connection/internal/gateway/hub.go) | 明确房间广播与用户定向协议；业务用户与连接 ID 分开，跨用户/多连接测试不误投 |
| 单用户只有一个 gateway 映射；房间 TTL 是整个 Set | [registry](../connection/internal/registry/redis_registry.go)、[presence 刷新/清理](../connection/cmd/server/main.go) | 多网关在线/迁移不会互相覆盖删除；活跃房间不会一直保留故障用户；多实例定时/故障测试 |
| gateway 尚无完整信号驱动 draining，backend relay 已可取消/关闭 | [gateway main](../connection/cmd/server/main.go)、[backend main](../backend/cmd/main.go) | SIGTERM 停新连接、取消后台任务、清理 presence、限时 flush/关闭队列和 WS，退出可验证 |
| kind 缺 fanout、8082 Service 和前端，CD 仅两种镜像 | [kustomization](../deploy/k8s/base/kustomization.yaml)、[connection Service](../deploy/k8s/base/connection.yaml)、[CD](../.github/workflows/cd.yml) | 完整拓扑可端到端收发；具体网关寻址、持久化、探针及 rollout 行为明确 |

## P2：工程和独立检索服务

| 问题 | 依据 | 建议验收 |
| --- | --- | --- |
| backend 配置分散、端口配置未生效，前端 WS URL 硬编码 | [配置参考](configuration.md) | 统一且可验证的配置入口；host/container 地址与 HTTPS/WS 协议适配，不靠改源码启动 |
| Python 没有自动行为测试；真实基础设施端到端覆盖缺失 | [测试说明](testing.md) | fanout 已有请求/鉴权/redirect/配置与本机 HTTP→WS 回归；持久重试、检索持久化及真实聊天链路仍需覆盖 |
| Python 本地单进程 JSON、导入即加载、空词表可能报错 | [storage](../python_ai_service/app/storage.py) | 空库/纯停用词/坏文件/重载有明确定义；写失败不破坏索引；并发和部署限制明确 |
| Python Docker COPY data 要求目录存在 | [Dockerfile](../python_ai_service/Dockerfile) | 全新 clone 无运行数据也可构建；数据不烘焙进镜像或提交仓库 |
| protobuf runtime 是间接依赖 | [生成说明](contracts.md)、[package](../frontend/package.json) | S2.1 已按现有锁文件同步生成头；后续明确直接使用的 runtime 依赖声明与锁定策略 |
| fanout/fanout 历史二进制被跟踪 | [fanout 指南](../fanout/AGENTS.md) | 单独清理版本控制产物；构建/演示路径无需修改已跟踪二进制 |

现有 Go 测试通过只证明覆盖的单元/分层行为，前端 build 只证明编译；不代表上述能力已补齐。修复条目后更新本页及相关协议/架构，并附具体回归证据。

S2.1–S2.4 本地代码与隔离验证已完成：统一幂等事务、永久删除墓碑、持久 room_seq、不可变 outbox、租约/退避/隔离 relay、broker 故障独立接纳、默认 WS 同步 persisted ACK、客户端 upsert/有界重试与权威游标补拉。新 schema v2 与准确 v1→v2 升级/失败回滚只在临时库验证。历史记录/证据见 [ADR 0002](decisions/0002-message-persistence-and-recovery.md)、[迁移](message-persistence.md)、[测试](testing.md)、[里程碑](remediation-plan.md)。

剩余边界：单 SQLite writer/单 relay；fanout 仍 best-effort/失败推进，依靠已实现的权威恢复触发补齐；旧 Kafka consumer 默认关闭，显式兼容仍会过期/丢命令。pending 仅页面内存，刷新/关标签不能继续未接纳发送，切账号清空；后台计时器节流和断网会延迟对账。没有自动数据保留期/游标清理、生产迁移/回退、真实 Kafka/Redis/浏览器完整链路、集群或容量证据；不承诺 delivered/exactly-once。S1b 账户管理/邀请产品规则与 S3/S4 缺口保留。
