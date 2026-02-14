import { useEffect, useRef } from "react";
import { createSocket } from "../services/socket";
import { v4 as uuidv4 } from "uuid";
import { UserInfo } from "../types/user";
import { SocketPayload } from "../types/socket";

export function useChatSocket(user?: UserInfo, token?: string, onMessage?: (p: any) => void) {
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!user || !token) return;

    const socket = createSocket();
    socketRef.current = socket;

    socket.onopen = () => {
      const auth: SocketPayload = {
        MsgType: "AUTH",
        RoomID: 0,
        UserID: user.id,
        Message: token,
        TempID: uuidv4(),
        CreatedAt: Date.now(),
      };
      socket.send(JSON.stringify(auth));
    };

    socket.onmessage = (e) => {
      try {
        const payload = JSON.parse(e.data);
        onMessage?.(payload);
      } catch {}
    };

    return () => socket.close();
  }, [user, token, onMessage]);

  function send(payload: SocketPayload) {
    socketRef.current?.send(JSON.stringify(payload));
  }

  function join(roomID: number, userID: string | number) {
    send({
      MsgType: "join",
      RoomID: roomID,
      UserID: userID,
      TempID: uuidv4(),
      CreatedAt: Date.now(),
    });
  }

  function leave(roomID: number, userID: string | number) {
    send({
      MsgType: "leave",
      RoomID: roomID,
      UserID: userID,
      TempID: uuidv4(),
      CreatedAt: Date.now(),
    });
  }

  return { send, join, leave, socketRef };
}