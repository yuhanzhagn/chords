import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import MessageList from "./MessageList";
import MessageComposer from "./MessageComposer";
import type { ChatMessage } from '../../types/chat';

interface ChatroomInfo {
  ID: number;
  Name: string;
}

interface ChatWindowProps {
  chatroom: ChatroomInfo | null;
  messages: ChatMessage[];
  loading: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  onSendMessage: (text: string) => void;
  onLoadMore: () => void;
}

const ChatWindow = ({ chatroom, messages, loading, loadingMore, hasMore, onSendMessage, onLoadMore }: ChatWindowProps) => {
  const [draft, setDraft] = useState("");
  console.log("messages in ChatWindow:", messages);

  useEffect(() => {
    setDraft("");
  }, [chatroom?.ID]);
  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!draft.trim()) return;

    onSendMessage(draft.trim());
    setDraft("");
  };

  if (!chatroom) {
    return (
      <section className="flex h-full min-h-0 flex-1 items-center justify-center bg-card text-muted-foreground">
        <p>Select a chatroom to start chatting.</p>
      </section>
    );
  }

  return (
    <section className="flex h-full min-h-0 flex-1 flex-col bg-card">
      <header className="flex items-center justify-between border-b border-border/70 bg-background/40 px-6 py-4">
        <div>
          <h2 className="text-lg font-semibold">{chatroom.Name}</h2>
          <p className="text-xs text-muted-foreground">Room ID #{chatroom.ID}</p>
        </div>
      </header>

      <MessageList
        messages={messages}
        loading={loading}
        loadingMore={loadingMore}
        hasMore={hasMore}
        onLoadMore={onLoadMore}
      />

      <MessageComposer draft={draft} onDraftChange={setDraft} onSubmit={handleSubmit} />
    </section>
  );
};

export default ChatWindow;
