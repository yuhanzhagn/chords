# 文档索引

| 文档 | 用途 |
| --- | --- |
| [仓库概览](../README.md) | 能力、目录、快速入口 |
| [根 AGENTS](../AGENTS.md) | Agent 最小上下文和约束 |
| [架构](architecture.md) | 模块、实际消息流、状态所有权 |
| [架构审计（2026-10-27）](architecture-audit.md) | 严重程度、源码证据、隔离复现、目标架构和整改顺序 |
| [整改实施计划](remediation-plan.md) | S1a/S1b 起步、后续阶段、授权边界和验收关卡 |
| [开发运行](development.md) | 安装、Compose、分服务运行、排障 |
| [配置参考](configuration.md) | 来源、默认值、端口与网络 |
| [接口与事件](contracts.md) | REST / WS / Kafka / Redis / 代码生成 |
| [测试](testing.md) | 按变更选检查、当前基线和覆盖缺口 |
| [现状与待办](status.md) | 已确认问题、源码证据和验收条件 |
| [Agent 流程](agent-workflow.md) | 接手、交付与交接模板 |
| [贡献约定](../CONTRIBUTING.md) | 开发和评审规则 |
| [决策记录](decisions/README.md) | 记录跨模块设计取舍；[身份与房间授权](decisions/0001-identity-and-room-authorization.md) |
| [kind 部署](../deploy/k8s/README.md) | 当前集群实验步骤与限制 |
| [Python 服务](../python_ai_service/README.md) | 独立检索 API 和运行 |

目录指南：[backend](../backend/AGENTS.md) · [connection](../connection/AGENTS.md) · [fanout](../fanout/AGENTS.md) · [frontend](../frontend/AGENTS.md) · [proto](../proto/AGENTS.md) · [Python](../python_ai_service/AGENTS.md) · [deploy](../deploy/AGENTS.md)。

文档以源码为依据；发现冲突时核实实际入口并修正。未来设计必须与已实现行为区分。`make check-docs` 检查本地内联 Markdown 链接的文件存在性，不验证外网、标题锚点或内容语义。

- [房间创建者与历史归属处置](room-ownership.md)：已实施规则、临时旧库兼容验证和人工指定/回退边界。
