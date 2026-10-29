# 共享事件协议

遵循 [根指南](../AGENTS.md)。唯一 schema 源是 [event.proto](kafka/event.proto)。

产物：backend、connection、fanout 各自的 proto/kafka/event.pb.go，以及 frontend/src/proto/kafka/event.ts。不要手工编辑。

- 不改变/复用字段号，删除字段时 reserved。
- go_package 是历史路径，生成使用显式 import mapping 与 source_relative 保持独立模块布局。
- 保留 JSON 别名，特别是 created_at 的历史 CreateAt；HTTP fanout 使用 encoding/json，不能直接套用 protojson 格式。
- TS id/createdAt/serverMessageId 必须保持字符串（forceLong=string），bytes 是 Uint8Array。

根目录 `make proto` 重建四份产物，需要 protoc、protoc-gen-go、锁文件安装的 ts-proto。工具安装与版本见 [协议](../docs/contracts.md)。生成会写文件，普通测试不自动运行。

变更后检查 schema 与四份 diff，运行三个 Go 模块测试及前端 build；生成器升级与字段变更分开说明。

S2.1 新增字段号 8 request_id、9 event_id、10 server_message_id、11 persisted_at；旧 Id/TempId 不复用。命令摘要覆盖新字段。admission.go 是三个 Go 消费者唯一的准入策略源，生成脚本除四份 protobuf 产物外也复制该文件，不能单改模块副本。HTTP 新回执和历史补 server_message_id 十进制字符串；内部 Go fanout 仍 encoding/json，不等于 protojson。

S2.4 字段12 room_seq 为uint64；SQLite支持0..MaxInt64，新发送为正数，HTTP/TS十进制字符串。v1不可变快照无seq继续发布并触发权威补拉，不改旧字节；新snapshot schema_version=2。ACK/error复用msg_type，persisted/deleted携带持久元数据，send_error.content为UTF-8结构化JSON，不能把其内容当用户消息。
