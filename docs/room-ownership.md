# 房间创建者与历史数据处置

2026-10-28 用户明确选择：仅创建者可删除房间。该规则已在本地实现；未操作真实或生产数据库。

## 运行规则与兼容

新房间的 `CreatorID` 从已校验的 Principal.UserID 派生，服务层再次检查有效 session；POST body 仍只接受 name，CreatorID/creator_id/ID 等额外输入不决定归属。创建不会自动加入房间；创建者删除不要求成员关系，其他成员/非成员均不能删除。读取、发现、本人 membership、成员历史/发送/订阅等原规则保留。

`chat_rooms.creator_id` 为可空整数并有索引，无默认用户值。现有 [InitDB](../backend/internal/app/db.go) 的 GORM AutoMigrate 对旧表增加可空列，不回填归属；NULL 和 0 都按未知处理。房间 ID/name/created_at 与 membership 不因该升级改变。无 API 自动认领、转让或管理员旁路；不按第一位成员、最早消息或当前用户猜测归属。未知归属房间继续可发现、加入与按成员规则聊天，只是普通 DELETE 返回 403。

响应新增 `CreatorID: number | null`，沿用 ChatRoom 的 Go 导出字段风格；现有前端只提交 name、消费 ID/Name，没有删除控件，额外响应字段不影响使用，本轮无前端改动。DELETE 成功仍为 200；有效 session 的非创建者/未知归属为 403，失效/缺失认证为 401，不存在为 404，非法/零/负数 ID 为 400。服务层先验证 session/归属，repo 的 DELETE 同时带 id/creator_id 条件，防止读后归属变化造成误删。授权检查与撤销提交仍不承诺线性化；沿用 [ADR 0001](decisions/0001-identity-and-room-authorization.md) 的在途边界。

当前删除保持既有“删除房间行”语义，没有新增消息/membership/cache/presence 的级联清理或即时断开协议；不存在的房间在后续成员授权检查中被拒绝。不要把 CreatorID 改动当作删除生命周期/历史保留策略的完整设计。

## 升级与回退

启动新版 backend 会通过现有初始化路径执行该加法 schema 变更。实际环境升级前需备份 SQLite，并在副本上执行初始化和权限验收；本轮仅使用 t.TempDir 下 SQLite 测试。不要混跑旧版写入/删除接口：旧版本不检查归属，可能绕过新政策；旧版创建的 NULL 归属房间也无法普通删除。

回退应保留 creator_id 列和已确认值，禁止为了兼容旧程序删除数据列或重写为默认用户。若应用回退，先关闭房间创建/删除入口或回退到同样执行创建者校验的版本；不能恢复任意登录用户可删除的旧策略。再次升级不得覆盖已确认归属或自动认领未知房间。

## 可审查的离线指定归属流程（仅操作说明）

默认不处理未知归属。如果维护者以后取得可核实的外部创建记录并获得数据操作授权，可按单房间执行下列流程；当前没有运行这些步骤，也没有实现线上管理员角色：

1. 停止 backend 写入，备份数据库并保存哈希/回退副本；先在副本验证，记录房间 ID、现值、目标用户 ID、证据、审批人和原因。
2. 读取并核对精确房间和仍存在的目标用户；不能从 membership、最早消息或登录身份推导创建者，也不能批量赋给同一个账号。
3. 用绑定参数在 `BEGIN IMMEDIATE` 事务内仅更新一个未知归属房间，且要求目标用户仍存在：

   ```sql
   UPDATE chat_rooms
   SET creator_id = :approved_user_id
   WHERE id = :room_id
     AND (creator_id IS NULL OR creator_id = 0)
     AND EXISTS (SELECT 1 FROM users WHERE id = :approved_user_id);
   ```

4. 检查影响行数必须为 1，并重新读取确认目标字段；不满足则 rollback，不覆盖已知归属。该 SQL 是审查模板，需要操作者提供并绑定经过审批的数值，不是自动迁移脚本。
5. 提交后验证目标创建者可删、其他人拒绝、成员历史/订阅规则不变；留存审计记录。没有可信证据的房间继续保持未知。
6. 如需回撤此次人工指定，在同等停写/备份/审批下，按记录且仅在当前归属仍等于本次指定值时恢复原值；不做全库清零。

自动回归见 [schema 兼容](../backend/internal/app/db_ownership_test.go)、[真实授权链路](../backend/test/controller_service_test/chatroom_ownership_test.go)、[服务层会话与条件删除](../backend/internal/service/chatroom_ownership_test.go)。更多验证边界见 [测试记录](testing.md)。
