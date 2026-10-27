# 当前状态与后续工作

基于 2026-10-27 的源码检查。下面是已定位的缺口与建议验收条件，不是本次文档整理已经实现的功能。运行检查结果见 [测试记录](testing.md)。优先级表示后续修复次序，不替代实际任务范围。

## P0：运行数据与身份边界

| 问题 | 源码证据 | 建议验收 |
| --- | --- | --- |
| backend 每次启动清空 Redis DB | [main](../backend/cmd/main.go)、[ClearRedis](../backend/internal/redisdb/redis.go) | 正常启动保留在线路由和缓存；开发重置变为显式操作；连接失败不会在错误检查前使用空 client |
| WS JWT 尚未接线，事件自报 user_id | [setupHandlerChain / newMux](../connection/cmd/server/main.go)、[客户端](../frontend/src/hooks/useChatSocket.tsx) | 无 token/过期 token 拒绝；业务 user ID 由验证身份派生；前后端 claims/token 传递一致，有跨模块回归 |
| 开发密钥硬编码、内部 fanout 无认证 | [JWT](../backend/utils/jwt.go)、[fanout handler](../connection/internal/source/http_source.go) | 明确配置/密钥来源与内部信任边界，非法 push 被拒；配置日志不泄露凭据 |

## P1：消息正确性、恢复与部署完整性

| 问题 | 源码证据 | 建议验收 |
| --- | --- | --- |
| 写库失败仍发布；数据库 ID 未回填，重试可能重复落库 | [KafkaService](../backend/internal/service/kafka_service.go)、[MessageService](../backend/internal/service/message_service.go) | 定义持久化与投递关系、幂等键及服务端 ID；失败/重复消息有可重复测试 |
| fanout 失败仍 MarkMessage，无持久补发 | [consumer](../fanout/internal/kafka/consumer.go)、[dispatcher](../fanout/internal/fanout/dispatcher.go) | 明确重试、提交、超时、死信/丢弃策略；错误和部分成功有覆盖，不无限阻塞分区 |
| 顶层 room_id 优先，user_ids fallback 实际按 client ID | [applyFanout](../connection/internal/source/http_source.go)、[Hub](../connection/internal/gateway/hub.go) | 明确房间广播与用户定向协议；业务用户与连接 ID 分开，跨用户/多连接测试不误投 |
| 单用户只有一个 gateway 映射；房间 TTL 是整个 Set | [registry](../connection/internal/registry/redis_registry.go)、[presence 刷新/清理](../connection/cmd/server/main.go) | 多网关在线/迁移不会互相覆盖删除；活跃房间不会一直保留故障用户；多实例定时/故障测试 |
| 其他用户实时新消息未加入 UI | [ChatRoom](../frontend/src/components/chatroom/ChatRoom.tsx)、[CONFIRM reducer](../frontend/src/hooks/useMessages.tsx) | 自己的 pending 被确认，别人的新消息可追加，同一事件不重复，非当前房间事件不污染列表 |
| WS 重连不自动恢复房间订阅 | [useChatSocket](../frontend/src/hooks/useChatSocket.tsx)、[selectRoom](../frontend/src/components/chatroom/ChatRoom.tsx) | 断网重连后自动重入当前房间并恢复未收到的历史；清理/重连无重复 socket |
| gateway/backend 尚无完整信号驱动 draining | [gateway main](../connection/cmd/server/main.go)、[backend main](../backend/cmd/main.go) | SIGTERM 停新连接、取消后台任务、清理 presence、限时 flush/关闭队列和 WS，退出可验证 |
| kind 缺 fanout、8082 Service 和前端，CD 仅两种镜像 | [kustomization](../deploy/k8s/base/kustomization.yaml)、[connection Service](../deploy/k8s/base/connection.yaml)、[CD](../.github/workflows/cd.yml) | 完整拓扑可端到端收发；具体网关寻址、持久化、探针及 rollout 行为明确 |

## P2：工程和独立检索服务

| 问题 | 依据 | 建议验收 |
| --- | --- | --- |
| backend 配置分散、端口配置未生效，前端 WS URL 硬编码 | [配置参考](configuration.md) | 统一且可验证的配置入口；host/container 地址与 HTTPS/WS 协议适配，不靠改源码启动 |
| fanout 和 Python 没有自动行为测试；端到端覆盖缺失 | [测试说明](testing.md) | 分发失败/重试、检索持久化和最小聊天链路有隔离可重复测试 |
| Python 本地单进程 JSON、导入即加载、空词表可能报错 | [storage](../python_ai_service/app/storage.py) | 空库/纯停用词/坏文件/重载有明确定义；写失败不破坏索引；并发和部署限制明确 |
| Python Docker COPY data 要求目录存在 | [Dockerfile](../python_ai_service/Dockerfile) | 全新 clone 无运行数据也可构建；数据不烘焙进镜像或提交仓库 |
| TS 生成器头与锁文件版本不同；protobuf runtime 是间接依赖 | [生成说明](contracts.md)、[package](../frontend/package.json) | 单独审查生成器同步；直接使用的 runtime 依赖有明确声明与锁定策略 |
| fanout/fanout 历史二进制被跟踪 | [fanout 指南](../fanout/AGENTS.md) | 单独清理版本控制产物；构建/演示路径无需修改已跟踪二进制 |

现有 Go 测试通过只证明覆盖的单元/分层行为，前端 build 只证明编译；不代表上述能力已补齐。修复条目后更新本页及相关协议/架构，并附具体回归证据。
