# Frontend

遵循 [根指南](../AGENTS.md)。React 19 / TypeScript / CRA，Tailwind/Radix UI。

## 定位与约束

- `src/App.tsx`：路由/登录状态；components/login、register 为认证页。
- `components/chatroom/ChatRoom.tsx` 为当前聊天入口；ChatRoomBackup 是保留版本，先查 import。
- `hooks/useChatrooms.tsx`、`useMessages.tsx`：REST/分页/消息状态；`useChatSocket.tsx`：protobuf/重连/join/leave/send。
- `services/api.tsx` 拼接 `http://${REACT_APP_URL}`；`services/socket.tsx` 硬编码 ws://localhost:8000/ws。
- `.env.example` 不带协议；改 URL 约定时还需搜索页面里的直接 fetch。
- WS 连接后首帧发送 JSON auth/token，收到 JSON ready 后才能发 protobuf；拒绝认证（1008）停止重连并显示错误，禁止打印 token。
- REST 消息用 ID/UserID/RoomID/Content/CreatedAt；TS protobuf 用 camelCase，id/createdAt/serverMessageId 是字符串、content 是 Uint8Array。避免丢失 64 位精度。
- 按持久ID/event/request upsert，TempID仅匹配本人pending；ACK/通知/历史任意顺序合并并按room_seq BigInt排序。认证ready重入房和逐页补拉，入房/通知/聚焦/每15秒对账，覆盖尾通知丢失和删除洞。见 [待办](../docs/status.md)。
- 复用 components/ui 和现有 hooks，不手改 src/proto/kafka/event.ts。

```bash
# 在根目录：
make install-frontend
make test-frontend
make build-frontend
# 在 frontend 内按需：
npm run lint
npx --no-install tsc --noEmit
```

Agent 使用非交互 Jest 参数；已有 App 冒烟及 WS auth/ready、请求复用/尺寸/精度测试，build 不验证完整消息流。node_modules 与 lock 不一致先按锁文件重装，不要修改生成代码绕过缺包。

S2.1：send 首次为逻辑 payload 分配随机 request_id，重复发送同一 payload 保留；TempID 仍匹配 pending，通知优先读取 serverMessageId/persistedAt，历史优先读取 server_message_id 字符串，before_id 不经 Number 舍入。pending仅JS内存，重连/重登/页面内导航保留原键，5次自动发送后unknown，可手动重试/取消等待；切账号清空，刷新不保留未接纳请求。历史用before_seq/after_seq/snapshot_seq，不混before_id。前端发送前检查 UTF-8 content 256 KiB；gateway 仍是完整准入边界，1009 显示尺寸错误。新 proto 字段用 fromPartial 构造，不手改生成文件。

useMessages的generation/AbortController与scope检查用于隔离旧房间/用户/token的迟到响应；完成整个冻结高水位扫描才权威删除已加载区间，保留实时新行/未决请求，不用LOAD覆盖。messageState.ts的纯合并器及useMessages/useChatSocket回归覆盖多页离线、尾丢失、删除、重试/账号切换。新ID/seq不能经Number；测试必须同步等待act异步更新。
