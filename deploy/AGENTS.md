# 部署工作

遵循 [根指南](../AGENTS.md)。默认准备并检查本地文件；任务包含部署/发布时再执行外部操作。

- 根 docker-compose.yml 覆盖 backend、connection、fanout、frontend、Redis、Kafka、Traefik。
- 根 Dockerfile 是 Compose 运行镜像，Go 二进制由宿主机预构建后挂载。
- Dockerfile.backend / Dockerfile.connection 为 CD、kind 多阶段构建。
- k8s/base 只有 namespace、backend、connection、Redis、Kafka、Ingress，缺少 fanout、frontend 和 Python。
- .github/workflows/ci.yml：Go 测试、前端 build、文档检查；cd.yml：main/tag 推送 backend、connection 镜像至 GHCR。

kind 尚非完整聊天部署：8082 未在 connection Service 暴露，fanout 不存在；也缺少生产持久卷与运维配置。backend 启动会清 Redis。TCP probes 不代表依赖或业务就绪。

多副本前先设计具体网关地址、多网关 presence、session/数据库一致性和 WS draining，单改 replicas 不足以完成扩容。

本地有工具时执行 `docker compose config --quiet`、`kubectl kustomize deploy/k8s/base`；记录验证限制。步骤见 [kind README](k8s/README.md)。
