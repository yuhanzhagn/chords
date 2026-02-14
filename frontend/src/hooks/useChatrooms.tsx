import { useCallback, useEffect, useState } from "react";
import { apiFetch } from "../services/api";
import { Chatroom, ChatroomResponse } from "../types/chat";

export function useChatrooms(username?: string, token?: string) {
  const [chatrooms, setChatrooms] = useState<Chatroom[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);

  useEffect(() => {
    if (!username || !token) return;

    setLoading(true);
    setError(null);

    apiFetch<ChatroomResponse>(`/memberships/${username}/chatrooms`, token)
      .then((data) => setChatrooms(data.data ?? []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [username, token, refreshKey]);

  return { chatrooms, loading, error, refresh };
}