# 共享事件协议

遵循 [根指南](../AGENTS.md)。唯一 schema 源是 [event.proto](kafka/event.proto)。

产物：backend、connection、fanout 各自的 proto/kafka/event.pb.go，以及 frontend/src/proto/kafka/event.ts。不要手工编辑。

- 不改变/复用字段号，删除字段时 reserved。
- go_package 是历史路径，生成使用显式 import mapping 与 source_relative 保持独立模块布局。
- 保留 JSON 别名，特别是 created_at 的历史 CreateAt；HTTP fanout 使用 encoding/json，不能直接套用 protojson 格式。
- TS id/createdAt 必须保持字符串（forceLong=string），bytes 是 Uint8Array。

根目录 `make proto` 重建四份产物，需要 protoc、protoc-gen-go、锁文件安装的 ts-proto。工具安装与版本见 [协议](../docs/contracts.md)。生成会写文件，普通测试不自动运行。

变更后检查 schema 与四份 diff，运行三个 Go 模块测试及前端 build；生成器升级与字段变更分开说明。
