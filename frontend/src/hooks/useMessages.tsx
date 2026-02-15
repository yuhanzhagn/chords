import { useCallback, useEffect, useReducer, useState } from "react";
import { apiFetch } from "../services/api";
import { ChatMessage } from "../types/chat";

type Action =
  | { type: "LOAD"; payload: ChatMessage[] }
  | { type: "ADD"; payload: ChatMessage }
  | { type: "CONFIRM"; payload: ChatMessage }
  | { type: "CLEAR" };

function reducer(state: ChatMessage[], action: Action): ChatMessage[] {
  switch (action.type) {
    case "LOAD":
      return action.payload;
    case "ADD":
      return [...state, action.payload];
    case "CONFIRM":
      return state.map((m) =>
        m.TempID === action.payload.TempID ? action.payload : m,
      );
    case "CLEAR":
      return [];
    default:
      return state;
  }
}

interface MessageApiModel {
  ID: number;
  UserID: number;
  RoomID: number;
  Content: string;
  CreatedAt: string;
}

function toChatMessage(message: MessageApiModel, currentUserID?: number): ChatMessage {
  return {
    ID: message.ID,
    UserID: message.UserID,
    RoomID: message.RoomID,
    Content: message.Content,
    CreatedAt: message.CreatedAt,
    TempID: "",
    status: "sent",
    fromself: currentUserID !== undefined && Number(message.UserID) === Number(currentUserID),
  };
}

export function useMessages(roomID?: number, currentUserID?: number, token?: string) {
  const [messages, dispatch] = useReducer(reducer, []);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  useEffect(() => {
    if (!roomID || !token) {
      dispatch({ type: "CLEAR" });
      return;
    }

    setLoading(true);
    setError(null);

    apiFetch<MessageApiModel[]>(`/chatrooms/${roomID}/messages`, token)
      .then((data) => {
        dispatch({
          type: "LOAD",
          payload: data.map((m) => toChatMessage(m, currentUserID)),
        });
      })
      .catch((e) => {
        setError(e.message);
        dispatch({ type: "CLEAR" });
      })
      .finally(() => setLoading(false));
  }, [roomID, token, currentUserID, refreshKey]);

  return {
    messages,
    loading,
    error,
    refresh,
    load: (m: ChatMessage[]) => dispatch({ type: "LOAD", payload: m }),
    add: (m: ChatMessage) => dispatch({ type: "ADD", payload: m }),
    confirm: (m: ChatMessage) => dispatch({ type: "CONFIRM", payload: m }),
    clear: () => dispatch({ type: "CLEAR" }),
  };
}
