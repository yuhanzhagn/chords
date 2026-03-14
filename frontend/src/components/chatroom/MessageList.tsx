import { cn } from '../../lib/utils';
import { ScrollArea } from '../ui/scroll-area';
import { ChatMessage } from '../../types/chat';

const getMessageClass = (msg: ChatMessage): string => {
  if (msg.status === "failed") return "failed";
  if (msg.status === "pending") return "pending";
  return msg.fromself ? "self" : "other";
};

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
}

const MessageList = ({ messages, loading }: MessageListProps) => {
  console.log("Rendering MessageList with messages:", messages);
  return (
    <ScrollArea className="flex-1">
      <div className="flex min-h-full flex-col gap-3 bg-gradient-to-b from-secondary/20 to-transparent px-6 py-5">
        {loading && <p className="text-sm text-muted-foreground">Loading messages…</p>}

        {!loading && messages.length === 0 && (
          <p className="text-sm text-muted-foreground">No messages yet. Say hello!</p>
        )}

        {messages.map((msg) => {
          const style = getMessageClass(msg);
          const isSelf = style === "self" || style === "pending" || style === "failed";

          return (
            <article
              key={msg.ID || msg.CreatedAt || msg.TempID}
              className={cn(
                "w-fit max-w-[70%] rounded-2xl border px-4 py-3 text-sm shadow-sm",
                isSelf
                  ? "ml-auto border-primary/40 bg-primary text-primary-foreground"
                  : "border-border/70 bg-secondary/60 text-foreground",
                style === "pending" && "opacity-70",
                style === "failed" && "border-destructive/50 bg-destructive/15 text-destructive"
              )}
            >
              <div className={cn("mb-1 flex items-center justify-between text-[11px]", isSelf ? "text-primary-foreground/80" : "text-muted-foreground")}>
                <span className="font-semibold">{msg.UserID || "Anonymous"}</span>
                <span>
                  {msg.CreatedAt ? new Date(msg.CreatedAt).toLocaleTimeString() : ''}
                </span>
              </div>
              <p className={cn("text-sm", isSelf ? "text-primary-foreground" : "text-foreground")}>
                {msg.Content || ''}
              </p>
            </article>
          );
        })}
      </div>
    </ScrollArea>
  );
};

export default MessageList;
