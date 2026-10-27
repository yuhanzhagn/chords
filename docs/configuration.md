# 配置参考

## 配置从哪里来

| 组件 | 来源 | 当前关键行为 |
| --- | --- | --- |
| backend | [YAML](../backend/configs/config.yaml)、[LoadConfig](../backend/internal/app/config.go) | 相对工作目录读取；数据库使用 dialect/dsn |
| backend HTTP | [main](../backend/cmd/main.go) | 固定 :8080，YAML app.port/frontend.port 目前不控制监听地址 |
| backend Redis | [redis.go](../backend/internal/redisdb/redis.go) | 硬编码 redis:6379、空密码、DB 0；启动 FLUSHDB |
| backend Kafka | [routes](../backend/internal/routes/routes.go)、[KafkaService](../backend/internal/service/kafka_service.go) | 硬编码 kafka:9092、user-request、notification、backend-ws-processor |
| connection | [YAML](../connection/configs/config.yaml)、[默认值](../connection/internal/app/config.go) | 从本模块目录读取 configs/config.yaml；没有通用 env override |
| fanout | [YAML](../fanout/configs/config.yaml)、[默认值](../fanout/internal/app/config.go) | 同上 |
| frontend REST | [api](../frontend/src/services/api.tsx) 及页面直接 fetch | REACT_APP_URL 不带协议，拼接 http://；修改后重启 dev server/重新 build |
| frontend WS | [socket](../frontend/src/services/socket.tsx) | 硬编码 ws://localhost:8000/ws，与 REST env 独立 |
| Python | [config](../python_ai_service/app/config.py) | 源文件相对的 data/store.json，无 DATA_PATH env override |

不要为未实现的选项创建假配置；新增 env/flags 时应同时修改读取实现、示例、测试和本表。

## 端口与网络

| 服务 | 内部/本地监听 | Compose 宿主机入口 |
| --- | --- | --- |
| Traefik | :80 | :8000 |
| backend | :8080 /api | 通过 Traefik /api，无直接端口映射 |
| connection WS | :8081 /ws | 通过 Traefik /ws |
| connection fanout | :8082 /fanout | 无映射，仅内部 |
| frontend CRA | Compose :8000；本地默认 :3000 | Traefik / |
| Redis | :6379 | :6379 |
| Kafka | :9092；controller :9093 | :9092，广播地址 kafka:9092 |
| Python（独立） | :8010 | 不在 Compose 中 |

Docker 服务名在 Docker 网络内解析；宿主机有端口映射不代表 Kafka metadata 中的 kafka:9092 能解析。容器内 localhost 也不指向其他服务。advertise_addr 是 worker 的目标地址，不能填浏览器的 localhost 或普通随机负载均衡地址。

## Gateway 配置

| YAML key | 默认值 |
| --- | --- |
| server.address | :8081 |
| fanout.address / advertise_addr | :8082 / connection:8082 |
| event.codec | protobuf；json 仅用于配套客户端，不适配当前浏览器主路径 |
| kafka.brokers / inbound_topic | [kafka:9092] / user-request |
| redis.addr / password / db | redis:6379 / 空 / 0 |
| redis.room_users_prefix / suffix | room: / :users |
| redis.user_gateway_prefix / suffix | user: / :gateway |
| redis.room_users_ttl / user_gateway_ttl | 2m / 2m |
| redis.presence_refresh_interval | 30s |

0 值 TTL/刷新间隔会被 withDefaults 替换，不能通过填 0 禁用。前缀/后缀与 fanout 必须一致。切换 gateway codec 为 json 时，Kafka sink 也会采用该 codec，而 backend/fanout 固定 protobuf，整条链路会不兼容。

## Fanout 配置

| YAML key | 默认值 |
| --- | --- |
| kafka.brokers / consumer_group / topics | [kafka:9092] / fanout-workers / [notification] |
| redis.* | 地址、密码、DB、key 前后缀与 gateway 相同；无 TTL 设置 |
| fanout.gateway_path / request_timeout | /fanout / 3s |

## 前端示例与开发凭据

[frontend/.env.example](../frontend/.env.example) 提供 `REACT_APP_URL=localhost:8000/api`；复制至 .env.local 可用于本地 CRA 配合 Compose 服务。本地直连 backend 用 localhost:8080/api，但 WS 仍指向 8000。

JWT 使用 backend/utils/jwt.go 和 connection/cmd/server/main.go 的开发固定密钥。修改其来源需要同时设计 claims、算法、REST session 与 WS token 传递；单改一个字符串会制造兼容问题。配置加载器还会打印整个结构，新增真实密码时应同时处理日志脱敏。
