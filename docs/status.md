# 当前状态与后续工作

基于 2026-10-27 审计，更新至 2026-10-28 的 S1 / N1–N2 实施。下面区分已修复与剩余缺口。运行检查结果见 [测试记录](testing.md)。优先级表示后续修复次序，不替代实际任务范围。

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
| 数据库 ID 未回填、发布失败无恢复，重试可能重复落库 | [KafkaService](../backend/internal/service/kafka_service.go)、[MessageService](../backend/internal/service/message_service.go) | 定义持久化与投递关系、幂等键及服务端 ID；失败/重复消息有可重复测试 |
| fanout 失败仍 MarkMessage，无持久补发 | [consumer](../fanout/internal/kafka/consumer.go)、[dispatcher](../fanout/internal/fanout/dispatcher.go) | 明确重试、提交、超时、死信/丢弃策略；错误和部分成功有覆盖，不无限阻塞分区 |
| 顶层 room_id 优先，user_ids fallback 实际按 client ID | [applyFanout](../connection/internal/source/http_source.go)、[Hub](../connection/internal/gateway/hub.go) | 明确房间广播与用户定向协议；业务用户与连接 ID 分开，跨用户/多连接测试不误投 |
| 单用户只有一个 gateway 映射；房间 TTL 是整个 Set | [registry](../connection/internal/registry/redis_registry.go)、[presence 刷新/清理](../connection/cmd/server/main.go) | 多网关在线/迁移不会互相覆盖删除；活跃房间不会一直保留故障用户；多实例定时/故障测试 |
| 其他用户实时新消息未加入 UI | [ChatRoom](../frontend/src/components/chatroom/ChatRoom.tsx)、[CONFIRM reducer](../frontend/src/hooks/useMessages.tsx) | 自己的 pending 被确认，别人的新消息可追加，同一事件不重复，非当前房间事件不污染列表 |
| WS 重连已恢复所选房间订阅，但未补拉遗漏历史 | [useChatSocket](../frontend/src/hooks/useChatSocket.tsx)、[selectRoom](../frontend/src/components/chatroom/ChatRoom.tsx) | 断网重连后恢复未收到的历史；清理/重连无重复 socket |
| gateway/backend 尚无完整信号驱动 draining | [gateway main](../connection/cmd/server/main.go)、[backend main](../backend/cmd/main.go) | SIGTERM 停新连接、取消后台任务、清理 presence、限时 flush/关闭队列和 WS，退出可验证 |
| kind 缺 fanout、8082 Service 和前端，CD 仅两种镜像 | [kustomization](../deploy/k8s/base/kustomization.yaml)、[connection Service](../deploy/k8s/base/connection.yaml)、[CD](../.github/workflows/cd.yml) | 完整拓扑可端到端收发；具体网关寻址、持久化、探针及 rollout 行为明确 |

## P2：工程和独立检索服务

| 问题 | 依据 | 建议验收 |
| --- | --- | --- |
| backend 配置分散、端口配置未生效，前端 WS URL 硬编码 | [配置参考](configuration.md) | 统一且可验证的配置入口；host/container 地址与 HTTPS/WS 协议适配，不靠改源码启动 |
| Python 没有自动行为测试；真实基础设施端到端覆盖缺失 | [测试说明](testing.md) | fanout 已有请求/鉴权/redirect/配置与本机 HTTP→WS 回归；持久重试、检索持久化及真实聊天链路仍需覆盖 |
| Python 本地单进程 JSON、导入即加载、空词表可能报错 | [storage](../python_ai_service/app/storage.py) | 空库/纯停用词/坏文件/重载有明确定义；写失败不破坏索引；并发和部署限制明确 |
| Python Docker COPY data 要求目录存在 | [Dockerfile](../python_ai_service/Dockerfile) | 全新 clone 无运行数据也可构建；数据不烘焙进镜像或提交仓库 |
| TS 生成器头与锁文件版本不同；protobuf runtime 是间接依赖 | [生成说明](contracts.md)、[package](../frontend/package.json) | 单独审查生成器同步；直接使用的 runtime 依赖有明确声明与锁定策略 |
| fanout/fanout 历史二进制被跟踪 | [fanout 指南](../fanout/AGENTS.md) | 单独清理版本控制产物；构建/演示路径无需修改已跟踪二进制 |

现有 Go 测试通过只证明覆盖的单元/分层行为，前端 build 只证明编译；不代表上述能力已补齐。修复条目后更新本页及相关协议/架构，并附具体回归证据。

S2 的标识、事务幂等/outbox、接纳/ACK、命令 backlog、尺寸预算与分页/重放已形成 [proposed ADR 0002](decisions/0002-message-persistence-and-recovery.md)。仅设计，尚未实施；下一片为统一发送用例与事务持久事实，不代表上述可靠性问题已修复。
