# HTTP、WS、Kafka 与 Redis 协议

## REST

来源：[路由](../backend/internal/routes/routes.go)、[controllers](../backend/internal/controller)。API 前缀 /api；受保护路由使用 `Authorization: Bearer <token>` 并校验数据库 session。

| 方法/路径（省略 /api） | 请求或查询 | 成功响应 |
| --- | --- | --- |
| POST /auth/register | username、email、password | 200 message |
| POST /auth/login | username、password | 200 id、token、jti、sessionID |
| POST /auth/logout | Authorization header；handler 内解析 | 200 message |
| POST /users | User JSON（含 username/email/password） | 201 User |
| GET /users | 无 | 200 User 数组 |
| POST /chatrooms | name | 201 {data: ChatRoom} |
| GET /chatrooms | 无 | 200 {data: ChatRoom[]} |
| GET /chatrooms/search | q | 200 {data: ChatRoom[]} |
| GET /chatrooms/:id | 路径 id | 200 {data: ChatRoom} |
| DELETE /chatrooms/:id | 路径 id | 200 message |
| POST /memberships/add-user | username、chatroomid（此拼写） | 200 message、username、chatroom_id |
| GET /memberships/:username/chatrooms | 路径 username | 200 {data: ChatRoom[]} |
| POST /messages | content、user_id、chat_room_id | 201 Message；不发布实时事件 |
| GET /chatrooms/:id/messages | 可选 limit、before_id | 200 Message 数组 |
| DELETE /messages/:id | 路径 id | 204 无 body |

除 register/login 外，logout 自行解析 Authorization；users/chatrooms/memberships/messages 由 group/route middleware 认证。认证存在不意味着已实现所有资源授权检查。

Message/ChatRoom 使用 Go 导出字段 JSON（如 ID、RoomID、CreatedAt），User 有显式小写 tags。不要将现有响应统一假定为 camelCase。分页只在提供 limit 或 before_id 时触发；limit 缺省 30、必须正数；before_id 非负。带游标筛选 id < before_id，页内按 created_at 升序返回；无分页参数返回完整历史。错误通常为 {error: string}，状态码因 controller 而异，测试以实际 handler 为准。

## WS 与 KafkaEvent

唯一 schema：[event.proto](../proto/kafka/event.proto)。默认浏览器通过 `/ws` 发送 protobuf 二进制帧。

| 字段号 / schema | Go | TS | 注意 |
| --- | --- | --- | --- |
| 1 id uint64 | Id uint64 | id string | 客户端事件 ID，当前不回填数据库 ID |
| 2 user_id uint32 | UserId uint32 | userId number | 当前来自客户端，尚未绑定认证身份 |
| 3 room_id uint32 | RoomId uint32 | roomId number | 业务房间 ID |
| 4 msg_type string | MsgType string | msgType string | join / leave / message |
| 5 content bytes | Content []byte | content Uint8Array | 文本用 UTF-8 编解码 |
| 6 temp_id string | TempId string | tempId string | optimistic 匹配键 |
| 7 created_at int64 | CreatedAt int64 | createdAt string | 客户端发毫秒；backend 出站覆盖为 Unix 秒 |

proto JSON names 是 ID、UserID、RoomID、MsgType、Content、TempID、CreateAt（历史拼写）。TS fromJSON/toJSON 与二进制编解码不同；HTTP fanout 则使用下面的 Go JSON 规则。

只有 message 进入 Kafka。join/leave 在 gateway 更新本地 group 和 Redis。连接 ID 单独生成，不能与事件中的业务 user_id 混用。当前没有端到端 ACK 协议；前端依赖回流事件的 temp_id 确认 pending。

| Topic | Producer → Consumer | 编码/默认 group |
| --- | --- | --- |
| user-request | connection → backend | protobuf / backend-ws-processor |
| notification | backend → fanout | protobuf / fanout-workers |

两个 consumer 默认新 group 从 newest 开始；不是完整历史回放。gateway 配置 json codec 会同时改变 Kafka sink，和固定 protobuf 的下游不兼容。

## 内部 HTTP fanout

来源：[worker payload](../fanout/internal/fanout/dispatcher.go)、[gateway handler](../connection/internal/source/http_source.go)。POST /fanout，成功 204。

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

非零顶层 room_id 触发本地 room 广播，user_ids 此时不做过滤。room_id 为 0 时要求 user_ids 非空，但当前 SendToClients 按连接 ID 查找，与 worker 传入业务用户 ID 不一致。event 必须存在，方法错误返回 405、非法 payload 返回 400；handler 读取上限 1 MiB。当前没有内部 HTTP 鉴权。

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
