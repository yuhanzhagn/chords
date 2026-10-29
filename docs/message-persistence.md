# 消息 schema v2、升级与回退

本页描述 [ADR 0002](decisions/0002-message-persistence-and-recovery.md) 的实际代码；本轮仅对临时 SQLite 运行迁移，从未打开项目实际数据库，不是生产升级记录。

## 显式版本

[InitDB](../backend/internal/app/db.go) 在模型迁移前检查消息版本，拒绝未来版本或损坏的已版本化结构。[迁移](../backend/internal/app/message_schema.go) 在同一事务中逐步记录 schema_migrations：

| 版本 | 事实变化 |
| --- | --- |
| 0→1 | 建立 send_requests 的 `(user_id,request_id)` 主键、event_id 唯一索引，以及 message_outbox 的 event_id 主键/状态索引；没有伪造旧请求或历史广播 |
| 1→2 | messages/send_requests 增加 room_seq；建立 room_sequences 高水位、`(room_id,room_seq)` 正序号唯一索引；outbox 增加领取/租约/尝试/退避/失败分类及 due 索引 |

新库同样执行 0→1→2，重复初始化验证列/索引/复合主键/序号唯一约束，不用修改版本常量跳过 v1。未版本化 send/outbox 表、缺列/缺索引、未来版本拒绝启动；不自动认领或修复损坏事实。

旧消息按 `ROW_NUMBER() OVER (PARTITION BY room_id ORDER BY id)` 回填历史规范序，各房间高水位取最大值；这不是恢复真实旧发送时序。已有请求关联到仍存在的消息时获得同一 seq；已删除旧请求保留 seq=0 墓碑，不造消息。新发送分配高水位后与 messages/request/outbox 同事务提交，删除不回退高水位，也不释放请求键。

旧 ID/内容/时间、CreatorID（包括未知归属）、membership、指纹/事件/请求键均保留；不产生历史 outbox 或广播。message_outbox.schema_version 与数据库 schema 版本是不同字段：旧版本 1 快照原字节保留，relay 可继续处理；新通知快照版本 2 含 seq。v1 缺 seq 通知只触发前端权威补拉。pending/published 及已发布时间不因迁移更改，坏快照运行时隔离为 quarantined。

## 运行和诊断

单 SQLite writer / 单 relay；不支持多节点共享 SQLite 的运维承诺。HTTP router 不连接 broker，后台 relay lazy 连接，停机保留未完成 pending。默认配置、状态和失败语义见 [配置](configuration.md)、[协议](contracts.md)。messages 删除但 pending 未发时标记 suppressed，不删原快照；已在途通知由周期历史对账修正。

Outbox 没有自动清理和最大重试次数；可通过只读聚合观察 status、last_error、attempts、next_attempt、lease_until。quarantined 需要调查版本/存储损坏，不能手改 payload 后假装同一事件；published 只是 broker 确认，suppressed 是删除抑制，都不代表用户送达。

## 真实升级前与回退

真实升级必须另获授权：先停写/停旧 relay，备份整个数据库及 WAL/SHM 的一致副本，在副本执行相同初始化和回归断言；检查旧行/时间/归属/成员/请求墓碑、快照字节与状态、每房间规范序和高水位、唯一约束及版本标记。禁止用项目 backend/mydb.sqlite 作为测试库。

本版本无破坏性 down migration。回退先停止写入和 relay，保留 messages/send_requests/message_outbox/room_sequences/schema_migrations；不得删请求键、pending 或降低版本标记。旧二进制可读取原字段，但 S2.1 版本检查会拒绝 v2，旧 writer 又可能写入零 seq/绕过 outbox，因此不能直接切回旧写流量。需要回到备份时必须先处理备份之后的已接纳事实，另行验证恢复方案；不把“旧字段能读”当成可安全回退写入。

[迁移测试](../backend/internal/app/message_schema_test.go) 覆盖准确 v1 DDL→v2、全新/旧 schema、重复初始化、稳定旧 ID 排名、已删除墓碑、pending/published 字节不变、无历史重播、未来/损坏 schema 拒绝及迁移失败事务回滚。[发送与恢复测试](../backend/internal/service/recovery_test.go) 覆盖真实文件 SQLite、回滚不消费序号、并发/重试、租约恢复和原字节重发。这些不代替生产数据库升级/回退实验。
