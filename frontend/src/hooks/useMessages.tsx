import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { apiFetch } from "../services/api";
import { ChatMessage } from "../types/chat";

type Action =
  | { type: "LOAD"; payload: ChatMessage[] }
  | { type: "PREPEND"; payload: ChatMessage[] }
  | { type: "ADD"; payload: ChatMessage }
  | { type: "CONFIRM"; payload: ChatMessage }
  | { type: "CLEAR" };

function reducer(state: ChatMessage[], action: Action): ChatMessage[] {
  switch (action.type) {
    case "LOAD":
      return action.payload;
    case "PREPEND":
      return [...action.payload, ...state];
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
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);
  const messagesRef = useRef<ChatMessage[]>([]);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);
  const pageSize = 30;

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  const getOldestServerId = useCallback(() => {
    for (const msg of messagesRef.current) {
      const id = Number(msg.ID);
      if (Number.isFinite(id) && id > 0) {
        return id;
      }
    }
    return undefined;
  }, []);

  useEffect(() => {
    if (!roomID || !token) {
      dispatch({ type: "CLEAR" });
      setHasMore(true);
      return;
    }

    setLoading(true);
    setError(null);
    setHasMore(true);

    apiFetch<MessageApiModel[]>(`/chatrooms/${roomID}/messages?limit=${pageSize}`, token)
      .then((data) => {
        setHasMore(data.length >= pageSize);
        dispatch({
          type: "LOAD",
          payload: data.map((m) => toChatMessage(m, currentUserID)),
        });
      })
      .catch((e) => {
        setError(e.message);
        dispatch({ type: "CLEAR" });
        setHasMore(false);
      })
      .finally(() => setLoading(false));
  }, [roomID, token, currentUserID, refreshKey]);

  const loadMore = useCallback(() => {
    if (!roomID || !token || loadingMore || !hasMore) return;
    const beforeId = getOldestServerId();
    if (!beforeId) {
      setHasMore(false);
      return;
    }

    setLoadingMore(true);
    setError(null);

    apiFetch<MessageApiModel[]>(
      `/chatrooms/${roomID}/messages?limit=${pageSize}&before_id=${beforeId}`,
      token
    )
      .then((data) => {
        if (data.length === 0) {
          setHasMore(false);
          return;
        }
        if (data.length < pageSize) {
          setHasMore(false);
        }
        dispatch({
          type: "PREPEND",
          payload: data.map((m) => toChatMessage(m, currentUserID)),
        });
      })
      .catch((e) => {
        setError(e.message);
      })
      .finally(() => setLoadingMore(false));
  }, [roomID, token, loadingMore, hasMore, getOldestServerId, currentUserID]);

  return {
    messages,
    loading,
    loadingMore,
    hasMore,
    error,
    refresh,
    load: (m: ChatMessage[]) => dispatch({ type: "LOAD", payload: m }),
    prepend: (m: ChatMessage[]) => dispatch({ type: "PREPEND", payload: m }),
    add: (m: ChatMessage) => dispatch({ type: "ADD", payload: m }),
    confirm: (m: ChatMessage) => dispatch({ type: "CONFIRM", payload: m }),
    clear: () => dispatch({ type: "CLEAR" }),
    loadMore,
  };
}
