import { useCallback, useEffect, useRef, useState } from "react";
import { apiFetch } from "../services/api";
import { ChatMessage } from "../types/chat";
import { mergeMessages, reconcileMessages, seq } from "./messageState";

interface MessageApiModel { ID: number; server_message_id: string; room_seq: string; request_id?: string; event_id?: string; UserID: number; RoomID: number; Content: string; CreatedAt: string }
interface Page { messages: MessageApiModel[]; high_water: string; next_cursor: string; has_more: boolean }
function convert(m: MessageApiModel, user?: number): ChatMessage {
  return { ID: m.server_message_id, RoomSeq: m.room_seq, RequestID:m.request_id || "", EventID:m.event_id || "", UserID: m.UserID, RoomID: m.RoomID, Content: m.Content, CreatedAt: m.CreatedAt, TempID: "", status: "sent", fromself: m.UserID === user };
}

export function useMessages(roomID?: number, currentUserID?: number, token?: string) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const state = useRef<ChatMessage[]>([]);
  const generation = useRef(0);
  const scope = `${currentUserID || ""}/${roomID || ""}/${token || ""}`;
  const stateScope = useRef(scope);
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  const reconcileRef = useRef<() => void>(() => {});
  const olderRef = useRef<() => void>(() => {});
  const update = useCallback((fn: (m: ChatMessage[]) => ChatMessage[]) => { state.current = fn(state.current); setMessages(state.current); }, []);

  useEffect(() => {
    const version = ++generation.current;
    const abort = new AbortController();
    let stopped = false, scanning = false, moreLoading = false;
    let scanAgain = false;
    let continuation: ReturnType<typeof setTimeout> | undefined;
    let oldest = "0", floor = "0", reconciled = "0";
    const valid = () => !stopped && version === generation.current && scopeRef.current === scope;
    stateScope.current = scope;
    update(() => []); setError(null); setHasMore(true); setLoading(false); setLoadingMore(false);
    if (!roomID || !token) return () => { stopped = true; abort.abort(); };
    const fetchPage = (query: string) => apiFetch<Page>(`/chatrooms/${roomID}/messages?limit=30&${query}`, token, abort.signal);
    const fail = (e: Error) => { if (valid() && e.name !== "AbortError") setError(e.message); };
    const apply = (page: Page) => page.messages.map(m => convert(m, currentUserID));
    // A scan freezes high water, pages ascending and only reconciles once every
    // page has arrived. At most ten requests per turn; continuation yields 1s.
    const scan = () => {
      if (!valid()) return;
      if (scanning) { scanAgain = true; return; }
      scanning = true;
      const low = floor;
      let cursor = low, high: string | undefined, rows: ChatMessage[] = [];
      const chunk = async () => {
        try {
          for (let i = 0; i < 10; i++) {
            const page = await fetchPage(`after_seq=${cursor}${high === undefined ? "" : `&snapshot_seq=${high}`}`);
            if (!valid()) return;
            high = page.high_water;
            rows = mergeMessages(rows, apply(page));
            if (!page.has_more) {
              reconciled = high;
              const completedRows = rows, completedHigh = high;
              update(m => reconcileMessages(m, completedRows, low, completedHigh));
              scanning = false; setError(null);
              if (scanAgain) { scanAgain = false; continuation = setTimeout(scan, 1000); }
              return;
            }
            if (seq(page.next_cursor) <= seq(cursor)) throw new Error("History cursor did not advance");
            cursor = page.next_cursor;
          }
          continuation = setTimeout(chunk, 1000);
        } catch (e) { scanning = false; fail(e as Error); }
      };
      void chunk();
    };
    const initial = async () => {
      setLoading(true);
      try {
        const page = await fetchPage("before_seq=0");
        if (!valid()) return;
        const rows = apply(page); oldest = rows[0]?.RoomSeq || "0";
        floor = oldest === "0" ? "0" : (seq(oldest) - 1n).toString();
        if(seq(page.high_water)>seq(reconciled)) reconciled = page.high_water;
        // Merge keeps notifications/receipts that arrived during the request.
        update(m => mergeMessages(m, rows)); setHasMore(page.has_more);
        scan();
      } catch (e) { fail(e as Error); } finally { if (valid()) setLoading(false); }
    };
    olderRef.current = async () => {
      if (!valid() || moreLoading || oldest === "0") return;
      moreLoading = true; setLoadingMore(true);
      try {
        const page = await fetchPage(`before_seq=${oldest}`);
        if (!valid()) return;
        const rows = apply(page); oldest = rows[0]?.RoomSeq || oldest;
        floor = (seq(oldest) - 1n).toString();
        update(m => mergeMessages(m, rows)); setHasMore(page.has_more); scan();
      } catch (e) { fail(e as Error); } finally { moreLoading = false; if (valid()) setLoadingMore(false); }
    };
    reconcileRef.current = scan;
    // Reconcile stale notifications as well as gaps. The periodic trigger covers
    // a lost final notification even if the socket stays connected forever.
    const timer = setInterval(scan, 15_000);
    window.addEventListener("focus", scan);
    const visibility = () => { if (!document.hidden) scan(); };
    document.addEventListener("visibilitychange", visibility);
    // Saved watermark is deliberately local to this room/credential generation.
    acceptRef.current = m => {
      if (!valid() || m.RoomID !== roomID) return;
      if (!m.RoomSeq || m.RoomSeq === "0") { scan(); return; }
      if (m.status === "sent" && seq(m.RoomSeq) <= seq(reconciled) && !state.current.some(v => String(v.ID) === String(m.ID) && v.status === "sent")) { scan(); return; }
      update(rows => mergeMessages(rows, [m])); scan();
    };
    void initial();
    return () => {
      stopped = true; abort.abort(); clearInterval(timer); clearTimeout(continuation);
      window.removeEventListener("focus", scan); document.removeEventListener("visibilitychange", visibility);
      reconcileRef.current = () => {}; olderRef.current = () => {}; acceptRef.current = () => {};
    };
  }, [roomID, currentUserID, token, scope, update]);
  const acceptRef = useRef<(m: ChatMessage) => void>(() => {});
  const add = (m: ChatMessage) => { if (scopeRef.current === scope && m.RoomID === roomID) update(rows => mergeMessages(rows, [m])); };
  return { messages: stateScope.current === scope ? messages : [], loading, loadingMore, hasMore, error,
    refresh: () => reconcileRef.current(), loadMore: () => olderRef.current(),
    add, confirm: (m: ChatMessage) => { if(scopeRef.current === scope) acceptRef.current(m); },
    clear: () => update(() => []),
    load: (rows: ChatMessage[]) => update(() => rows), prepend: (rows: ChatMessage[]) => update(m => mergeMessages(m, rows)),
    remove: (request: string) => { if(scopeRef.current === scope) update(rows => rows.filter(m => m.RequestID !== request)); },
  };
}
