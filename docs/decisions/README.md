# 架构决策记录（ADR）

当前记录：[0001 身份、会话与房间授权](0001-identity-and-room-authorization.md)，区分已实施、用户确认和未决规则。另有 [0002 消息持久化与恢复](0002-message-persistence-and-recovery.md)（proposed，仅设计）。本目录规定后续记录方式；[架构文档](../architecture.md) 描述现状，不代表所有实现都是已认可的长期选择。

涉及身份协议、消息投递语义、状态所有权、数据库/队列替换或跨服务部署拓扑时，增加 `NNNN-short-title.md`。普通 bug 修复或文案修改无需 ADR。

建议结构：

```text
标题：NNNN 具体决策
状态：proposed / accepted / superseded
背景：触发问题、限制、现有源码入口
选择：方案、影响模块、协议/数据变化
备选：考虑过的方案及取舍
结果：兼容、迁移、运行与维护成本
验证：验收条件、测试/指标、回滚方式
```

未批准的设计标注 proposed；不要把 agent 推测或待办自动标为 accepted。
