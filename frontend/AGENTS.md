# Frontend

遵循 [根指南](../AGENTS.md)。React 19 / TypeScript / CRA，Tailwind/Radix UI。

## 定位与约束

- `src/App.tsx`：路由/登录状态；components/login、register 为认证页。
- `components/chatroom/ChatRoom.tsx` 为当前聊天入口；ChatRoomBackup 是保留版本，先查 import。
- `hooks/useChatrooms.tsx`、`useMessages.tsx`：REST/分页/消息状态；`useChatSocket.tsx`：protobuf/重连/join/leave/send。
- `services/api.tsx` 拼接 `http://${REACT_APP_URL}`；`services/socket.tsx` 硬编码 ws://localhost:8000/ws。
- `.env.example` 不带协议；改 URL 约定时还需搜索页面里的直接 fetch。
- token 当前只控制是否建连，没有发送给 WS 服务端；认证接入需跨模块设计。
- REST 消息用 ID/UserID/RoomID/Content/CreatedAt；TS protobuf 用 camelCase，id/createdAt 是字符串、content 是 Uint8Array。避免丢失 64 位精度。
- 以 TempID 匹配 optimistic pending；CONFIRM 当前只替换已有消息，不能添加其他用户新消息；重连未自动重入房间。见 [待办](../docs/status.md)。
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

Agent 使用非交互 Jest 参数；目前只有 App 冒烟测试，build 不验证消息流。node_modules 与 lock 不一致先按锁文件重装，不要修改生成代码绕过缺包。
