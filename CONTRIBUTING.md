# 开发与贡献

先读 [仓库指南](AGENTS.md)、目标目录的 AGENTS 及 [文档索引](docs/README.md)。

## 改动流程

1. 明确问题、影响模块和可观察的验收条件，检查已有工作区修改。
2. 从运行入口追踪实际调用链和邻近测试，先复现问题；历史注释或未接线的工具类不等于运行行为。
3. 保持改动集中，延续模块边界与接口注入方式，避免顺手升级依赖或批量格式化。
4. 执行 [相应验证](docs/testing.md)。Bug 修复补充有意义的回归测试；记录环境限制和已有失败。
5. 同步更新接口、配置和运行文档；schema 变更按 [proto 指南](proto/AGENTS.md) 生成四份代码。
6. 检查最终 diff 和 `git diff --check`，避免带入凭据、数据、二进制和缓存。

## 代码与评审

- Go 使用 `gofmt`；后端保持 controller → service → repo，接口变更同步 mocks。网关保持 transport、handler、sink、registry 边界。
- 前端使用 TypeScript，复用现有 UI/hooks；显式映射字段大小写、64 位字符串和时间单位。
- Python 保持 schema 与检索/存储分离，测试使用临时数据。
- 普通构建不运行 `go mod tidy`。前端按锁文件 `npm ci --legacy-peer-deps`；依赖变更更新对应锁文件。
- 评审说明写明触发条件、改变后的行为、验证命令与结果、兼容影响和剩余限制。
- 跨模块架构选择参考 [决策记录约定](docs/decisions/README.md)。只有任务包含部署、推送或发布时才执行相应外部操作。
