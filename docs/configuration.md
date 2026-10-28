# 配置参考

## 配置从哪里来

| 组件 | 来源 | 当前关键行为 |
| --- | --- | --- |
| backend | [YAML](../backend/configs/config.yaml)、[LoadConfig](../backend/internal/app/config.go) | 相对工作目录读取；数据库使用 dialect/dsn |
| backend JWT | 环境变量 JWT_SECRET | 必填、至少 32 字节；缺失/过短启动失败；HS256，access/command audience 分离 |
| backend HTTP | [main](../backend/cmd/main.go) | 固定 :8080，YAML app.port/frontend.port 目前不控制监听地址 |
| backend Redis | [redis.go](../backend/internal/redisdb/redis.go) | 硬编码 redis:6379、空密码、DB 0；启动保留状态 |
| backend Kafka | [routes](../backend/internal/routes/routes.go)、[KafkaService](../backend/internal/service/kafka_service.go) | 硬编码 kafka:9092、user-request、notification、backend-ws-processor |
| connection | [YAML](../connection/configs/config.yaml)、[默认值](../connection/internal/app/config.go) | 从本模块目录读取 configs/config.yaml；没有通用 env override |
| fanout | [YAML](../fanout/configs/config.yaml)、[默认值](../fanout/internal/app/config.go) | 从本模块目录读取；FANOUT_TOKEN 仅从环境读取 |
| 内部 fanout 服务认证 | gateway 与 worker 的环境变量 FANOUT_TOKEN | 必填且两端相同，至少 32 字节、无空格的可见 ASCII；YAML token 不生效；无默认值，缺失/非法在 Kafka/Redis 前失败 |
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
| auth.backend_url | http://app:8080；CHORDS_AUTH_BACKEND_URL 可覆盖（kind 使用 http://backend:8080） |
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

JWT 仅由 backend 签发与验签，密钥从 `JWT_SECRET` 读取，至少 32 字节。根 [.env.example](../.env.example) 只给占位字段；用 `openssl rand -hex 32` 在本地生成，设置 shell 环境或未跟踪的 `.env`，不要提交真实密钥。Compose app 将该变量传给 backend；kind 引用 `chords-auth` Secret。gateway 无需持有签名密钥，通过 backend 校验 session。

发布此修改需前后端一起更新并重新登录：旧 token 缺少 `chords-access` audience，将被拒绝。修改密钥会使既有 token 和未消费命令授权失效。`auth.backend_url` 必须是可信 HTTP(S) 服务地址；它接收 Authorization header，生产跨主机传输需 TLS。backend 调用超时 3 秒，失败关闭连接/拒绝授权，不使用永久缓存。connection/fanout 加载器不再打印配置结构；不要在 URL 加入凭据。

## 内部服务凭据

`FANOUT_TOKEN` 仅供 worker→gateway，独立生成，不能复用 JWT_SECRET、access/command JWT。根 `.env.example` 只提供空字段；例如另行执行 `openssl rand -hex 32`，将结果设置在 shell 或未跟踪的 `.env`。Compose 只向 connection/fanout 注入，kind gateway 引用 `chords-fanout` Secret 的 FANOUT_TOKEN；kind 仍无 worker 和 8082 Service。模块测试自设隔离凭据，无需配置真实密钥。

worker 在 `X-Chords-Fanout-Token` 头中传递；gateway 用标准库 SHA-256 固定长度摘要与 constant-time 比较，在方法/body/解码/Hub 之前拒绝缺失/错误或重复凭据头。worker 禁止所有 HTTP redirect，仅 2xx 成功；HTTP 响应 body 不进入错误/日志。两端不一致时启动无法跨进程比较，实际推送返回 401。更换凭据需协调两端（当前只支持一个值，滚动切换可能暂时拒绝投递），现有 worker 失败仍标记消费，不能宣称切换无损。

本地 Compose 使用 HTTP，凭据不防窃听或重放。跨不可信主机必须提供可信证书的 HTTPS 与受控网络；保持 Go 默认 TLS 验证，不设置 insecure skip。Redis 中的 gateway 地址、broker、内部网络仍需可信访问控制；此凭据不是 Kafka ACL 或多租户隔离方案。
