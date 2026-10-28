# HTTP、WS、Kafka 与 Redis 协议

## REST

来源：[路由](../backend/internal/routes/routes.go)、[controllers](../backend/internal/controller)。API 前缀 /api；受保护路由使用 `Authorization: Bearer <token>` 并校验数据库 session。

| 方法/路径（省略 /api） | 请求或查询 | 成功响应 |
| --- | --- | --- |
| POST /auth/register | username、email、password | 200 message |
| POST /auth/login | username、password | 200 id、token、jti、sessionID |
| POST /auth/logout | Authorization header；撤销 token 对应 session | 200 message |
| POST /auth/ws-authorize | Authorization；room_id/kind/digest（见下） | 200 user_id/username/session_id，可含 command_token |
| POST /users | username/email/password DTO；忽略 id、sessions 等内部字段 | 201 UserResponse |
| GET /users | 无 | 200 UserResponse 数组 |
| POST /chatrooms | name；创建者由 Principal 派生，忽略额外归属/ID 字段 | 201 {data: ChatRoom}，新增 CreatorID |
| GET /chatrooms | 无 | 200 {data: ChatRoom[]} |
| GET /chatrooms/search | q | 200 {data: ChatRoom[]} |
| GET /chatrooms/:id | 路径 id | 200 {data: ChatRoom} |
| DELETE /chatrooms/:id | 正整数路径 id；有效 session 且仅创建者 | 200 message；非创建者/未知归属 403，不存在 404 |
| POST /memberships/add-user | username、chatroomid（此拼写） | 200 message、username、chatroom_id |
| GET /memberships/:username/chatrooms | 路径 username | 200 {data: ChatRoom[]} |
| POST /messages | content、user_id、chat_room_id | 201 Message；不发布实时事件 |
| GET /chatrooms/:id/messages | 可选 limit、before_id | 200 Message 数组 |
| DELETE /messages/:id | 路径 id | 204 无 body |

除 register/login 外，logout 自行解析 Authorization；users/chatrooms/memberships/messages 由 group/route middleware 认证。middleware 将 User/Session 转为不可变 Principal；消息读取/发送/删除要求有效 session 和房间成员。消息 user_id 必须与 Principal 相同，删除仅允许作者。membership 添加和列表查询只允许本人 username；发现/搜索房间继续允许登录用户。房间删除仅创建者可用（不要求成员），创建/删除服务再次校验 session；旧房间 CreatorID 为 null 或 0 时拒绝普通删除，不猜测归属。见 [ADR](decisions/0001-identity-and-room-authorization.md)、[归属/升级与离线处置](room-ownership.md)。

UserResponse 只含 id、username、email、created_at，不返回密码/哈希/session。注册与 POST /users 均在服务层哈希一次；哈希失败不创建用户。成功登录事务性替换旧会话，密码/输入/签发/持久化失败保留旧会话。登出旧 token 不撤销后来登录生成的新 session。

Message/ChatRoom 使用 Go 导出字段 JSON（如 ID、RoomID、CreatedAt），User 有显式小写 tags。不要将现有响应统一假定为 camelCase。分页只在提供 limit 或 before_id 时触发；limit 缺省 30、必须正数；before_id 非负。带游标筛选 id < before_id，页内按 created_at 升序返回；无分页参数返回完整历史。错误通常为 {error: string}，状态码因 controller 而异，测试以实际 handler 为准。

## WS 与 KafkaEvent

唯一 schema：[event.proto](../proto/kafka/event.proto)。浏览器连接 `/ws` 后必须在 5 秒内发送至多 4096 字节的 JSON 文本首帧 `{"type":"auth","token":"<access token>"}`。gateway 请求 backend 校验数据库 session，成功后返回 `{"type":"ready"}`，此后使用 protobuf 二进制帧。认证前不注册 client、不加入房间。无效凭据/权限返回 WS close 1008；客户端停止重连并提示重新登录/房间权限。认证服务暂时不可达/5xx/429 返回 1013，客户端退避重连。token 不放 URL/query/subprotocol 或日志。业务帧仍受 1 MiB 上限约束。

连接 user/session 固定；事件 user_id 可以是 0（服务端补身份）或本人，冒用其他 ID 关闭连接。join/message/leave 由 backend 逐帧检查 session 与永久房间成员。出站 room 广播发送前再次检查；空闲连接每 5 秒复查 session，校验失败关闭。授权检查之后已在途/排队的数据可能完成；不承诺与撤销提交的线性化边界。HTTP 校验超时 3 秒，服务不可用时拒绝。

| 字段号 / schema | Go | TS | 注意 |
| --- | --- | --- | --- |
| 1 id uint64 | Id uint64 | id string | 客户端事件 ID，当前不回填数据库 ID |
| 2 user_id uint32 | UserId uint32 | userId number | 客户端值不得改变连接身份；0 或已认证本人 |
| 3 room_id uint32 | RoomId uint32 | roomId number | 业务房间 ID |
| 4 msg_type string | MsgType string | msgType string | join / leave / message |
| 5 content bytes | Content []byte | content Uint8Array | 文本用 UTF-8 编解码 |
| 6 temp_id string | TempId string | tempId string | optimistic 匹配键 |
| 7 created_at int64 | CreatedAt int64 | createdAt string | 客户端发毫秒；backend 出站覆盖为 Unix 秒 |

proto JSON names 是 ID、UserID、RoomID、MsgType、Content、TempID、CreateAt（历史拼写）。TS fromJSON/toJSON 与二进制编解码不同；HTTP fanout 则使用下面的 Go JSON 规则。

只有 message 进入 Kafka。gateway 请求 `/api/auth/ws-authorize`，body 为 room_id、kind（join/leave/message/subscribe）、digest。无 room/kind 的 `{}` 用于会话校验；message 的 digest 是实际 protobuf 字节 SHA-256 十六进制。返回 30 秒有效、audience 为 chords-command 的 HS256 命令凭据，含 user/session、room 与内容摘要。

REST 的失效 session 返回 401，资源越权返回 403；认证数据库故障返回 503，不能误当成密码错误或撤销。

命令凭据仅放 `user-request` Kafka header `chords-command`，经 async/retry 保留，不转发到 notification 或前端。backend 消费时检查签名、过期、内容摘要、user/room 与数据库有效 session/成员；缺失或篡改凭据不写库、不发布。access token 的 audience 为 chords-access，两类凭据不能互换。入站未授权/过期命令当前被消费并丢弃，失败不提供客户端 ACK 或可靠补发，30 秒 backlog 以上可能丢消息；这是 S2 需要处理的明确限制。共享 event.proto 未改动。

join/leave 在 gateway 更新本地 group 和 Redis。连接 ID 单独生成，不能与事件中的业务 user_id 混用。当前没有端到端 ACK 协议；前端依赖回流事件的 temp_id 确认 pending。

| Topic | Producer → Consumer | 编码/默认 group |
| --- | --- | --- |
| user-request | connection → backend | protobuf / backend-ws-processor |
| notification | backend → fanout | protobuf / fanout-workers |

两个 consumer 默认新 group 从 newest 开始；不是完整历史回放。gateway 配置 json codec 会同时改变 Kafka sink，和固定 protobuf 的下游不兼容。

## 内部 HTTP fanout

来源：[worker payload](../fanout/internal/fanout/dispatcher.go)、[gateway handler](../connection/internal/source/http_source.go)。POST /fanout，必须带 `X-Chords-Fanout-Token: <FANOUT_TOKEN>`，成功 204。缺失/错误或重复凭据头在检查方法、读取 body、解码与调用 Hub 前返回 401；用户 Authorization/access/command JWT 不能替代该独立服务秘密。有效凭据后保留原有 405/400/payload/room 广播与 client ID fallback 语义，仍逐接收者校验 session/成员。worker 禁止所有 3xx 跟随，将非 2xx 作为错误；不将对端 body 写入错误/日志。共享秘密不防 HTTP 窃听/重放；跨不可信网络使用可信 TLS，见 [配置](configuration.md)。

```json
{
  "room_id": 1,
  "user_ids": [42],
  "event": {
    "id": 123,
    "user_id": 42,
    "room_id": 1,
    "msg_type": "message",
    "content": "aGk=",
    "temp_id": "client-123",
    "created_at": 1770000000
  }
}
```

嵌套 event 经 `encoding/json` 处理生成 Go struct 的 snake_case tags；bytes 为 base64，int64/uint64 输出为 JSON number，不是 protojson 的字符串约定。本示例不是 WS 文本帧。

非零顶层 room_id 触发本地 room 广播，user_ids 此时不做过滤。room_id 为 0 时要求 user_ids 非空，但当前 SendToClients 按连接 ID 查找，与 worker 传入业务用户 ID 不一致。event 必须存在，方法错误返回 405、非法 payload 返回 400；handler 读取上限 1 MiB。服务凭据鉴权在上述检查之前执行，仍保留用户/session/成员投递检查。

## Redis

| Key | 类型 | 所有者与用途 |
| --- | --- | --- |
| room:{roomID}:users | Set，成员为十进制业务 user ID | connection 写，fanout SMEMBERS；整集合默认 TTL 2m |
| user:{userID}:gateway | String，网关 host:port 或 URL | connection 写，fanout MGET；默认 TTL 2m |
| username | JSON 聊天室列表缓存 | backend membership service；30m |

gateway 默认每 30s 刷新在线映射；最后本地连接离线时清理。所有在线 key 的前后缀在两份 YAML 里必须匹配。Redis 在线成员不等于 SQLite UserChatRoom 永久成员。

## 代码生成

需要 protoc（现有产物头声明 3.21.12）、protoc-gen-go v1.36.11、前端锁文件安装的 ts-proto。命令从根目录执行：

```bash
# 一次性准备；按需执行：
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
make install-frontend
# Go plugin 需位于 PATH、GOPATH/bin，或指定 PROTOC_GEN_GO：
make proto
```

[生成脚本](../scripts/generate-proto.sh) 显式覆盖每个模块的 Go import mapping，使用 source_relative，TS 使用 forceLong=string。输出为三个 Go pb.go 与一个 TS 文件；不改 schema 内历史 go_package 来迁移目录。

可将产物写到临时目录检查生成器变化：

```bash
bash scripts/generate-proto.sh /tmp/chords-proto-preview
```

当前已提交 TS 头声明 2.11.2，锁文件实际解析 ts-proto 2.11.5，重生成可能出现版本头差异。先检查版本变化，再评估字段变化；不把生成器差异混当业务需求。变更字段后验证全部消费者，不复用字段号，保留 long 字符串语义。
