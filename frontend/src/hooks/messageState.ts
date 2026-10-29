import { ChatMessage } from "../types/chat";

export const seq = (value?: string) => BigInt(value || "0");
const durable = (m: ChatMessage) => m.status === "sent" && !!m.RoomSeq && m.RoomSeq !== "0";
export function mergeMessages(state: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  const out = [...state];
  for (const item of incoming) {
    const index = out.findIndex(m =>
      (m.status === "sent" && item.status === "sent" && String(m.ID) === String(item.ID)) ||
      (!!m.EventID && m.EventID === item.EventID) ||
      (String(m.UserID) === String(item.UserID) && m.fromself && item.fromself &&
        ((!!m.RequestID && m.RequestID === item.RequestID) ||
         (m.status !== "sent" && (!m.RequestID || !item.RequestID) && !!m.TempID && m.TempID === item.TempID))));
    if (index < 0) out.push(item);
    else { const previous = out[index]!; if (previous.status !== "sent" || item.status === "sent") out[index] = { ...previous, ...item, Content: item.status === "sent" ? item.Content : item.Content || previous.Content, CreatedAt: item.CreatedAt || previous.CreatedAt }; }
  }
  return out.sort((a, b) => durable(a) && durable(b) ? (seq(a.RoomSeq) < seq(b.RoomSeq) ? -1 : seq(a.RoomSeq) > seq(b.RoomSeq) ? 1 : 0) : durable(a) ? -1 : durable(b) ? 1 : 0);
}

// The server is authoritative only inside the completed scan's range. Preserve
// real-time events beyond its high water and pending requests until a receipt.
export function reconcileMessages(state: ChatMessage[], rows: ChatMessage[], low: string, high: string): ChatMessage[] {
  const ids = new Set(rows.map(m => String(m.ID)));
  return mergeMessages(state.filter(m => !durable(m) || seq(m.RoomSeq) <= seq(low) || seq(m.RoomSeq) > seq(high) || ids.has(String(m.ID))), rows);
}
