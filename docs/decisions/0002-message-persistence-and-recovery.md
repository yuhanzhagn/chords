# 0002 · 消息持久化、确认与恢复

日期：2026-10-28；更新：2026-10-30。状态：**S2.1–S2.4 本地实现与隔离验证完成**。分轮历史见 [整改记录](../remediation-plan.md)，实际命令见 [测试记录](../testing.md)。没有真实数据库升级、部署或真实 Kafka/Redis/浏览器端到端证据。

## 不变量与标识

保留 SQLite、三个独立 Go module、React/CRA 和 [既有身份边界](0001-identity-and-room-authorization.md)。目标是业务事实幂等、至少一次发布、客户端去重与权威历史恢复；persisted 与 delivered 不同，不承诺网络端到端 exactly-once。

| 字段 | 事实与兼容规则 |
| --- | --- |
| request_id | 首次发送前生成的随机 128 位键；数据库主键 `(authenticated_user_id, request_id)`，不含 session。相同键重登可重试；旧合法 temp_id 可映射，两者缺失则服务端生成并返回 deduplication_supported=false |
| server_message_id | SQLite Message.ID；HTTP 精确十进制字符串、wire uint64、TS string。旧数字 ID/客户端 id 保留旧含义 |
| event_id | backend 随机 128 位 ID，outbox 主键；任何重发使用相同字节和 ID |
| persisted_at | 已提交 Message.CreatedAt 的 RFC3339Nano UTC；回执与通知一致，旧 created_at 仍为 Unix 秒 |
| temp_id | 首次快照保留；只可匹配本人 pending，不作为跨用户唯一键 |
| room_seq | 同事务分配的每房间递增序号；SQLite 正 int64 范围，HTTP/TS 字符串；删除允许有洞，不重排 |

[SendMessage](../../backend/internal/service/send_message.go) 在同一固定 SQLite 连接事务内验证有效 session、永久成员与发送身份，然后分配 room_seq、写 messages/send_requests/message_outbox。数据库约束裁决并发，无进程全局发送锁。指纹版本 1 是 JSON `[1, room_id, 实际内容]` 的 SHA-256，不规范化内容，不包含 session/temp_id。相同键异内容/房间返回 409，同内容返回原 ID/event/time/seq，不再次分配序号。

授权与写入共享事务快照，SQLite 写冲突重试时重新授权。BUSY/LOCKED 最多 12 次尝试、2 秒预算，耗尽返回 503，客户端保留原键。当前驱动 COMMIT 失败可能留下活动事务，故保留 [同连接 BEGIN/COMMIT/ROLLBACK](../../backend/internal/repo/repo_container.go)，清理失败丢弃连接。事务临时将原连接 busy_timeout 封顶 50ms，结束还原，避免驱动忙等待超过取消预算。序号更新也随失败回滚。

删除保留永久请求墓碑，无 message FK 级联。重放仍校验当前权限，返回原回执及 deleted=true，不重建消息。v1 中已删除消息无序号的旧墓碑保留 room_seq=0；不伪造历史事实。

## S2.2 · 后台发布与生命周期

[Relay](../../backend/internal/service/outbox_relay.go) 启动即扫描，默认每秒最多 32 条到期 pending；持久字段记录 attempts、next_attempt、lease_token、lease_until、last_error。逐条按实际领取时间起算租约 30 秒，发布预算 5 秒；只有同一 token 能标记结果。失败从尝试结束后指数退避，最高 64 秒（无自动次数终止）；SQLite 领取/扫描/标记均有独立 1 秒上下文和 50ms busy cap。租约覆盖重启/重叠退出，首版支持单 backend writer/单 relay，不构成多节点 SQLite 方案。

发布固定 topic notification、十进制 room key 和原 protobuf 快照。broker 确认后才标记 published；标记失败保留领取状态，租约过期可重发相同事件。publish 超时/取消属于结果未知，可重复；broker 错误记 broker_unavailable，超时记 publish_outcome_unknown。不支持的快照版本、损坏/不匹配内容进入 quarantined，不占据后续 pending 批次；永久失败不伪装成功。

新快照 schema_version=2 含 room_seq，旧 schema_version=1 pending 原字节继续发布；旧通知缺 seq 时客户端请求权威历史。快照不含 access/command/service 凭据。生产请求不即时发布，全部交后台领取；保留的注入式 PublishPending 也使用相同租约路径，不能并行绕过协调。

发布前查消息是否仍存在；已删除则标记 suppressed/message_deleted，保留原快照及请求键。检查后删除与已在途通知存在竞态，客户端按权威历史周期对账清除，不能宣称删除与网络投递线性化。没有新增删除事件。

[Router](../../backend/internal/routes/routes.go) 构造不连接 Kafka；[runtime](../../backend/internal/routes/message_runtime.go) 独立启动 lazy producer/relay。broker 故障时 HTTP 可持久接纳，重连由后台继续尝试。Sarama 连接/读写/metadata/ACK 均有预算；退出先取消 HTTP/relay，再关闭 producer/可选 consumer 和数据库。正常 shutdown 不依赖成功清空 outbox，未完成事实由下次启动恢复。

## S2.3 · 默认同步接纳与 persisted ACK

生产 gateway 不创建 Kafka 入站 sink；message 在尺寸/固定身份检查后，同步带当前 access token 调用可信 backend `POST /api/messages`，backend 再认证并在事务内授权，提交后返回回执。此时才通过单 WS writer 队列返回 protobuf msg_type=persisted（墓碑为 deleted）。该默认请求不同时写 user-request。join/leave 与接收者授权仍走原 backend 检查，服务凭据不替代用户授权。

回执携带 request_id、server_message_id、event_id、persisted_at、room_seq，内容来自本次逻辑请求。send_error 的 content 为 UTF-8 JSON `{code, outcome, retryable}`。401/403、409、413、400 是明确拒绝；5xx/429、网络失败、响应丢失和坏回执是 unknown/retryable。unknown 可能已经提交，必须复用 request_id。gateway 排队、Kafka publish、fanout 204 或 WS write 都不是用户已读/送达回执。

旧 Kafka 命令消费者默认关闭，仅 `CHORDS_LEGACY_COMMANDS=1` 显式启用；保留 user-request、30 秒 chords-command audience、摘要与 session/成员复查，不延长凭据、不传长效 access token 到 broker。原命令过期/失败推进仍可能丢弃，兼容消费者没有 persisted ACK。不提供自动窗口到期，运维升级时明确关闭旧 writer 后移除兼容开关；默认新客户端不使用它。

前端按持久 ID/event_id 合并，request_id 可关联本人 pending；历史也返回请求关联信息。通知/ACK/历史任意顺序不重复，远端新消息追加，按 room_seq 的 BigInt 排序；64 位值不经过 Number。账号/房间/凭据 generation 隔离旧回调和迟到 HTTP。

pending 仅在当前页面 JS 内存保留（最多 128 条），支持同账号重连/重登/页面内导航；不存凭据，不承诺刷新/关闭标签后继续重试。每逻辑请求最多自动发送 5 次，2/4/8/16/30 秒等待后保留 unknown；手动重试复用键，取消只停止本地等待，不撤销可能已提交的事务。不同账号清空旧队列。恢复的 pending 仍由同账号历史/回执合并。

## S2.4 · 房间序与权威恢复

[历史用例](../../backend/internal/service/message_history.go) 支持 before_seq / after_seq，二者恰选一个；0 分别表示 newest / 从起点。limit 默认 30、最大 100。返回 messages、high_water、next_cursor、has_more；序号均十进制字符串。授权、序号高水位和该页消息读取共享同一个 SQLite 快照，后续页携带 snapshot_seq 固定上界；删除可在跨请求期间变化，下次对账修正。高水位来自 room_sequences，尾消息被删也不回退。after 页结束时 next_cursor=high_water，删除洞不会无限等待。

旧 before_id 和数组响应暂兼容，禁止与 seq 混用；旧无参历史最多最近 100 条。非法/溢出/重复/空游标 400，游标或 snapshot 超过当前房间高水位 409 reset required。没有自动清理/保留期限，也没有静默回到 newest；人工恢复数据库导致游标超前时显示错误，用户明确重新入房重置本地视图。

[客户端恢复](../../frontend/src/hooks/useMessages.tsx) 入房先取最新页，随后从已加载区间下界向固定高水位逐页对账；重连、任何新通知/缺 seq、focus、visibility 和每 15 秒触发。没有下一条通知的尾丢失也能被周期触发恢复。每轮至多 10 个请求，未完成扫描 1 秒后续跑，直到该冻结高水位；断网失败保留现有视图，后续触发重试。离线超过一页不会只抓 newest 而漏中间消息。

只有整次扫描完成才对扫描区间移除权威不存在的持久消息，同时保留高水位之后实时回流和未决请求。迟到通知低于已对账高水位、且当前无该持久 ID 时，先拉权威而不直接复活；已显示消息后续删除也由下一次对账移除。当前只对已加载范围同步删除，旧未加载消息在加载时读取权威状态；后台标签可能受浏览器计时器节流，恢复网络/聚焦后继续。不扩展为编辑/已读产品。

## 迁移、尺寸与验证

[消息 schema v2](../message-persistence.md) 显式按 0→1→2 迁移；新表/room_seq 回填/唯一索引/版本标记同一事务。旧消息按每房间 ID 稳定排名形成历史规范序，不声称是真实旧发送顺序。不造旧请求、不新增历史 outbox，不改已存在 v1 快照、pending/published、membership/CreatorID/墓碑。只在临时数据库验证，旧 writer 不可直接回退使用。

唯一 [准入源](../../proto/kafka/admission.go) 及生成四份产物保留 256 KiB UTF-8、128 字节可见 ASCII 标识、1 MiB WS/fanout/record 预算，新增 seq 计入最坏 envelope。REST 完整 JSON body 为 `6*256KiB+4096`。非零 room 广播每选中网关一次，不携带无效 user_ids。

fanout 仍为 best-effort，失败后仍 MarkMessage；历史是恢复权威，具体触发见上文。新 group 默认 newest、presence 单 gateway/整集合 TTL、房间级联清理和 gateway 整体 draining 仍待 S3/S4。测试覆盖真实临时 SQLite、HTTP/WS 与生产 fanout dispatcher，集成实验仅替换 Kafka 传输和 Redis registry；真实 broker/Redis/浏览器、生产升级和部署均未验证。
