# Agent 接手、执行与交接

## 最小上下文

先读 [根 AGENTS](../AGENTS.md)，然后按任务读对应目录指南、入口、邻近测试。需要跨模块理解时再读 [架构](architecture.md)、[协议](contracts.md)、[配置](configuration.md)；已知问题先查 [状态](status.md)。不需要每次把整个仓库载入上下文。

推荐检索：

```bash
git status --short
rg --files backend/internal connection/internal fanout/internal frontend/src
rg -n '目标符号' backend connection fanout frontend/src
make help
```

确认运行装配，而不仅是找到相同名字的辅助实现。例如 WS JWT 工具存在却未接入 main；ChatRoomBackup 不是当前页面。没有根 Go module，测试应进入相应目录或通过 Make。

## 执行与完成条件

先写清触发条件与预期可观察行为，选择最小影响模块。一般任务直接完成本地可审查改动，不因例行实现选择停下来征求许可。业务修复优先复现并添加有意义的回归测试；文档修改使用链接检查和源码核对。

跨服务变更先列影响：请求/响应与时间单位、schema 兼容、Redis key/TTL、session/身份、数据迁移、部署拓扑。不能只改其中一端。大任务可建立临时任务计划，但不要把计划当成已完成结果。

完成时检查 diff、执行 [相关验证](testing.md)、同步文档；不得用跳过断言、编辑生成代码或吞掉错误让检查变绿。已有失败记录命令、原因、是否本次引入。

## 可复用交接模板

```text
目标：具体触发条件与完成后的行为
范围：模块/文件入口；需要保留的既有行为
现状：已完成修改、未完成步骤、关键设计取舍
证据：复现步骤；验证命令、结果和环境限制
下一步：按依赖排序的具体动作
验收：能实际运行/观察到的条件
工作区：已有用户修改、本次文件、是否提交
```

保持路径准确、区分事实与假设；不要把运行时数据、token 或完整私有日志写入交接文档。

## 文档维护

| 改动类型 | 应同步文档 |
| --- | --- |
| 服务边界、主链路 | architecture、相关目录 AGENTS |
| API/schema/Redis 语义 | contracts、schema 与生成产物 |
| 配置、启动、端口、Docker/CI | configuration、development、testing、部署 README |
| 修复已知问题 | status 移除/更新条目，保留测试证据 |
| 跨模块设计取舍 | decisions 中的 ADR |

根 AGENTS 保持精简，避免把完整 API 表和历史日志重复塞入每个目录指南。以源码为事实来源，遇到过时说明就地修正。
