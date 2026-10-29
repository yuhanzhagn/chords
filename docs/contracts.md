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
| POST /messages | content、user_id、chat_room_id、request_id/temp_id | 首次 201 / 重放 200 持久回执，通知经后台 relay |
| GET /chatrooms/:id/messages | limit；before_seq / after_seq / snapshot_seq；兼容 before_id | seq 查询 200 分页对象；旧查询数组（最多 100） |
| DELETE /messages/:id | 路径 id | 204 无 body |

除 register/login 外，logout 自行解析 Authorization；users/chatrooms/memberships/messages 由 group/route middleware 认证。middleware 将 User/Session 转为不可变 Principal；消息读取/发送/删除要求有效 session 和房间成员。消息 user_id 必须与 Principal 相同，删除仅允许作者。membership 添加和列表查询只允许本人 username；发现/搜索房间继续允许登录用户。房间删除仅创建者可用（不要求成员），创建/删除服务再次校验 session；旧房间 CreatorID 为 null 或 0 时拒绝普通删除，不猜测归属。见 [ADR](decisions/0001-identity-and-room-authorization.md)、[归属/升级与离线处置](room-ownership.md)。

UserResponse 只含 id、username、email、created_at，不返回密码/哈希/session。注册与 POST /users 均在服务层哈希一次；哈希失败不创建用户。成功登录事务性替换旧会话，密码/输入/签发/持久化失败保留旧会话。登出旧 token 不撤销后来登录生成的新 session。

Message/ChatRoom 使用 Go 导出字段 JSON（如 ID、RoomID、CreatedAt），User 有显式小写 tags。不要将现有响应统一假定为 camelCase。新 seq 分页见下文。旧 limit/before_id 返回数组，limit 默认 30、范围 1–100，before_id 非负 int64；无参数最多最近 100 条。旧页内按 created_at 升序（仅兼容旧 ID 游标，顺序恢复使用 seq）。错误通常为 {error: string}，状态码因 controller 而异，测试以实际 handler 为准。

## 消息发送回执与幂等

POST /messages 新增可选 request_id/temp_id；REST 与 Kafka 调用同一 SendMessage，session/成员在事务内复查。键为 `(已认证 user_id, request_id)`，不含 session；指纹绑定版本、room 与实际持久内容。同键同内容返回原结果，不同内容/房间 409；重复读取仍需当前权限。旧合法非空 temp_id 可作为请求键；两个标识均缺失时服务端分配，deduplication_supported=false，客户端无标识重发不能去重。

首次提交返回 201，重复返回 200，保留原 ID/Content/UserID/RoomID/CreatedAt，增加 request_id、event_id、server_message_id（十进制字符串）、persisted_at（RFC3339Nano UTC）、room_seq（十进制字符串）、replayed、deleted、deduplication_supported。删除后返回原 ID/时间及 deleted=true，不重建消息；历史新增 server_message_id/room_seq 字符串，seq 历史另带 request_id/event_id 关联，旧 ID 数字保留。新 TS 读者优先使用字符串，before_id 传字符串避免大整数舍入。

消息、send_requests、message_outbox 一起提交；成功只表示 persisted。HTTP 请求不等待 broker；后台 relay 领取 pending，退避和重启/租约过期后重试。发布后标记失败允许相同 event_id 重复通知。通知只从已持久快照产生，不含 auth/command/service 凭据。详细语义与迁移见 [ADR 0002](decisions/0002-message-persistence-and-recovery.md)、[迁移说明](message-persistence.md)。

content 最多 256 KiB UTF-8，request_id/temp_id 各最多 128 字节可见 ASCII（无空格）；超限 413，非法编码/标识 400。REST 完整 JSON body 上限 `6*256KiB+4096` 以允许最坏转义，解码后再验实际字节。SQLite BUSY/LOCKED 有界重试，耗尽 503，应保留 request_id 重试。响应丢失属于结果未知，不换键。默认 WS 同步接纳后返回 persisted ACK；仅显式启用的旧 Kafka 命令仍受 30 秒 backlog/失败推进限制。

## 序号分页与客户端恢复

`GET /chatrooms/:id/messages?after_seq=0&limit=30` 或 `before_seq=0&limit=30` 返回 `{messages, high_water, next_cursor, has_more}`。before_seq=0 取最新页，after_seq=0 从起点；页内均 seq 升序。after 后续页使用 next_cursor，并携带首次 high_water 为 snapshot_seq，冻结扫描上界；每页授权/高水位/行来自同一 SQLite 快照，跨页删除下次对账收敛。高水位独立于存活消息，删除尾部不倒退；after 扫描完成的 next_cursor=high_water，删除洞不用等待。

所有 seq/snapshot/cursor 都是 0..9223372036854775807 的十进制字符串；before 与 after 恰选一个，禁止混 before_id。非法、空值、重复参数、溢出或 limit 非 1–100 返回 400；after/snapshot 超过当前高水位或 before 超过 high+1 返回 409 reset required。没有自动保留期/游标过期清理，不静默改查 newest。旧 ID 数字/游标仅兼容旧读者，不能承担新排序。

浏览器入房、重连、通知（含 v1 无 seq）、聚焦/可见以及每 15 秒对已加载区间权威对账；每批最多 10 页，更多页 1 秒后继续。整次扫描完成才清除该范围已删除消息，保留高水位之后回流及未决请求；迟到已删通知先补拉，不直接复活。最后一条通知丢失且无下一条也由周期触发恢复。标签后台节流/离线会延迟恢复，刷新后仅恢复服务端历史，不保留内存 pending。

## WS 与 KafkaEvent

唯一 schema：[event.proto](../proto/kafka/event.proto)。浏览器连接 `/ws` 后必须在 5 秒内发送至多 4096 字节的 JSON 文本首帧 `{"type":"auth","token":"<access token>"}`。gateway 请求 backend 校验数据库 session，成功后返回 `{"type":"ready"}`，此后使用 protobuf 二进制帧。认证前不注册 client、不加入房间。无效凭据/权限返回 WS close 1008；客户端停止重连并提示重新登录/房间权限。认证服务暂时不可达/5xx/429 返回 1013，客户端退避重连。token 不放 URL/query/subprotocol 或日志。业务帧仍受 1 MiB 上限约束；message 入队前执行 256 KiB UTF-8 content 和 128 字节可见 ASCII request_id/temp_id 准入，超限 close 1009，非法编码/标识 close 1008。

连接 user/session 固定；事件 user_id 可以是 0（服务端补身份）或本人，冒用其他 ID 关闭连接。join/message/leave 由 backend 逐帧检查 session 与永久房间成员。出站 room 广播发送前再次检查；空闲连接每 5 秒复查 session，校验失败关闭。授权检查之后已在途/排队的数据可能完成；不承诺与撤销提交的线性化边界。HTTP 校验超时 3 秒，服务不可用时拒绝。

| 字段号 / schema | Go | TS | 注意 |
| --- | --- | --- | --- |
| 1 id uint64 | Id uint64 | id string | 旧客户端事件 ID；保留，不作为服务端消息 ID |
| 2 user_id uint32 | UserId uint32 | userId number | 客户端值不得改变连接身份；0 或已认证本人 |
| 3 room_id uint32 | RoomId uint32 | roomId number | 业务房间 ID |
| 4 msg_type string | MsgType string | msgType string | 入站 join/leave/message；出站另有 persisted/deleted/send_error |
| 5 content bytes | Content []byte | content Uint8Array | 文本用 UTF-8 编解码 |
| 6 temp_id string | TempId string | tempId string | optimistic 匹配键 |
| 7 created_at int64 | CreatedAt int64 | createdAt string | 客户端发毫秒；通知为持久时间的 Unix 秒 |
| 8 request_id string | RequestId string | requestId string | 用户范围内的逻辑请求键，重试复用 |
| 9 event_id string | EventId string | eventId string | backend 随机生成的稳定通知 ID |
| 10 server_message_id uint64 | ServerMessageId uint64 | serverMessageId string | Message.ID，TS 保持 64 位字符串 |
| 11 persisted_at string | PersistedAt string | persistedAt string | RFC3339Nano UTC，和回执相同 |
| 12 room_seq uint64 | RoomSeq uint64 | roomSeq string | 房间事务序，当前 SQLite 最大 int64；旧 v1 为 0 |

proto JSON names 是 ID、UserID、RoomID、MsgType、Content、TempID、CreateAt（历史拼写）。TS fromJSON/toJSON 与二进制编解码不同；HTTP fanout 则使用下面的 Go JSON 规则。

默认 message 带连接当前 access token 同步调用可信 backend POST /api/messages；事务提交后才返回 persisted（墓碑 deleted），含稳定 request/message/event/time/seq。失败 send_error 的 content 是 UTF-8 JSON `{code,outcome,retryable}`；401/403/409/413/400 为 rejected，5xx/429/网络/坏响应为 unknown、可重试。同一个默认请求不进入 user-request。最多 5 次自动重试后显示 unknown，手动重试保留 ID；取消仅停止本地等待，不撤销事务。不同账号清空队列，当前页面内存最多 128 条，不存凭据，不支持刷新后重发。

旧兼容路径仅 backend CHORDS_LEGACY_COMMANDS=1 消费；生产 gateway 不装配旧 sink。兼容 adapter 请求 `/api/auth/ws-authorize`，body 为 room_id、kind（join/leave/message/subscribe）、digest。无 room/kind 的 `{}` 用于会话校验；message 的 digest 是实际 protobuf 字节 SHA-256 十六进制。返回 30 秒有效、audience 为 chords-command 的 HS256 命令凭据，含 user/session、room 与内容摘要。

REST 的失效 session 返回 401，资源越权返回 403；认证数据库故障返回 503，不能误当成密码错误或撤销。

命令凭据仅放 `user-request` Kafka header `chords-command`，经 async/retry 保留，不转发到 notification 或前端。backend 消费时检查签名、过期、内容摘要、user/room 与数据库有效 session/成员；缺失或篡改凭据不写库、不发布。access token 的 audience 为 chords-access，两类凭据不能互换。入站未授权/过期命令当前被消费并丢弃，失败不提供客户端 ACK 或可靠补发，30 秒 backlog 以上可能丢消息；这是显式旧兼容路径的限制，默认同步接纳不走该路径。新增字段参与实际 protobuf 摘要，不改变命令的 30 秒过期规则。

join/leave 在 gateway 更新本地 group 和 Redis。连接 ID 单独生成，不能与事件中的业务 user_id 混用。persisted ACK 只确认后端事务；前端按持久 ID/event_id upsert，temp_id 只匹配本人 pending，ACK/通知/历史任意顺序合并。

| Topic | Producer → Consumer | 编码/默认 group |
| --- | --- | --- |
| user-request | 显式旧 adapter → backend（默认 consumer 关闭） | protobuf / backend-ws-processor；30 秒命令授权 |
| notification | backend → fanout | protobuf / fanout-workers |

两个 consumer 默认新 group 从 newest 开始；不是完整历史回放。生产 gateway 使用 protobuf；json codec 仅供配套客户端，旧 Kafka sink 以 json 编码仍不兼容固定 protobuf 下游。fanout 保留 best-effort/失败仍 MarkMessage，丢通知由上述权威恢复触发补齐。

## 内部 HTTP fanout

来源：[worker payload](../fanout/internal/fanout/dispatcher.go)、[gateway handler](../connection/internal/source/http_source.go)。POST /fanout，必须带 `X-Chords-Fanout-Token: <FANOUT_TOKEN>`，成功 204。缺失/错误或重复凭据头在检查方法、读取 body、解码与调用 Hub 前返回 401；用户 Authorization/access/command JWT 不能替代该独立服务秘密。有效凭据后保留原有 405/400/payload/room 广播与 client ID fallback 语义，仍逐接收者校验 session/成员。worker 禁止所有 3xx 跟随，将非 2xx 作为错误；不将对端 body 写入错误/日志。共享秘密不防 HTTP 窃听/重放；跨不可信网络使用可信 TLS，见 [配置](configuration.md)。

```json
{
  "room_id": 1,
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

非零顶层 room_id 触发本地 room 广播；worker 由 registry 选择 gateway，每个 gateway 每事件一次，并省略不做过滤的 user_ids，接收人数不扩大 body。room_id 为 0 时要求 user_ids 非空，但当前 SendToClients 按连接 ID 查找，与 worker 传入业务用户 ID 不一致。event 必须存在，方法错误返回 405、非法 payload 返回 400；handler max+1 读取，上限 1 MiB，超限明确返回 413。worker 也在发送前检查最终 body 上限。服务凭据鉴权在上述检查之前执行，仍保留用户/session/成员投递检查。

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

S2.1 按现有锁文件 ts-proto 2.11.5 生成，头从 2.11.2 同步为 2.11.5；没有升级依赖或生成器。三份 Go 产物一致。脚本还将唯一准入源 proto/kafka/admission.go 同步到三个 Go 消费模块；修改策略后一起生成/对比。变更字段后验证全部消费者，不复用字段号，保留 long 字符串语义。
