import { useEffect, useRef } from 'react';
import { cn } from '../../lib/utils';
import { ScrollArea } from '../ui/scroll-area';
import { ChatMessage } from '../../types/chat';

const getMessageClass = (msg: ChatMessage): string => {
  if (msg.status === "failed") return "failed";
  if (msg.status === "pending") return "self";
  return msg.fromself ? "self" : "other";
};

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
}

const MessageList = ({ messages, loading, loadingMore, hasMore, onLoadMore }: MessageListProps) => {
  console.log("Rendering MessageList with messages:", messages);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const viewportRef = useRef<HTMLDivElement | null>(null);
  const prevCountRef = useRef(0);
  const pendingAdjustRef = useRef(false);
  const prevScrollHeightRef = useRef(0);
  const stickToBottomRef = useRef(true);

  const handleScroll = () => {
    const viewport = viewportRef.current;
    if (!viewport) return;

    const distanceToBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight;
    stickToBottomRef.current = distanceToBottom < 40;

    if (viewport.scrollTop <= 40 && hasMore && !loadingMore) {
      prevScrollHeightRef.current = viewport.scrollHeight;
      pendingAdjustRef.current = true;
      onLoadMore();
    }
  };

  useEffect(() => {
    if (!messages.length) return;
    const viewport = viewportRef.current;
    if (!viewport) return;

    if (pendingAdjustRef.current) {
      const newHeight = viewport.scrollHeight;
      const delta = newHeight - prevScrollHeightRef.current;
      viewport.scrollTop = viewport.scrollTop + delta;
      pendingAdjustRef.current = false;
    } else {
      const isInitialLoad = prevCountRef.current === 0 && messages.length > 0;
      if (isInitialLoad || stickToBottomRef.current) {
        bottomRef.current?.scrollIntoView({ block: "end", behavior: "auto" });
      }
    }

    prevCountRef.current = messages.length;
  }, [messages.length]);
  return (
    <ScrollArea
      className="flex-1 min-h-0"
      viewportRef={viewportRef}
      onViewportScroll={handleScroll}
    >
      <div className="flex min-h-full flex-col gap-3 bg-gradient-to-b from-secondary/20 to-transparent px-6 py-5">
        {loadingMore && (
          <p className="text-xs text-muted-foreground">Loading older messages…</p>
        )}
        {loading && <p className="text-sm text-muted-foreground">Loading messages…</p>}

        {!loading && messages.length === 0 && (
          <p className="text-sm text-muted-foreground">No messages yet. Say hello!</p>
        )}

        {messages.map((msg) => {
          const style = getMessageClass(msg);
          const isPending = msg.status === "pending";
          const isFailed = msg.status === "failed";
          const isSelf = style === "self" || style === "failed";

          return (
            <article
              key={msg.ID || msg.CreatedAt || msg.TempID}
              className={cn(
                "w-fit max-w-[70%] rounded-2xl border px-4 py-3 text-sm shadow-sm",
                isSelf
                  ? "ml-auto border-primary/40 bg-primary text-primary-foreground"
                  : "border-border/70 bg-secondary/60 text-foreground",
                isFailed && "border-destructive/50 bg-destructive/15 text-destructive",
                isPending && "border-dashed border-primary/40 bg-primary/25 text-primary-foreground/90 opacity-80 shadow-none"
              )}
            >
              <div className={cn("mb-1 flex items-center justify-between text-[11px]", isSelf ? "text-primary-foreground/80" : "text-muted-foreground")}>
                <span className="font-semibold">{msg.UserID || "Anonymous"}</span>
                <span>
                  {isPending && "Sending… "}
                  {msg.CreatedAt ? new Date(msg.CreatedAt).toLocaleTimeString() : ''}
                </span>
              </div>
              <p className={cn("text-sm", isSelf ? "text-primary-foreground" : "text-foreground")}>
                {msg.Content || ''}
              </p>
            </article>
          );
        })}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  );
};

export default MessageList;
