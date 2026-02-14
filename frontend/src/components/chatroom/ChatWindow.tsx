import { useEffect, useState } from 'react';
import type { FormEvent } from 'react';
import MessageList from "./MessageList";
import MessageComposer from "./MessageComposer";
import type { ChatMessage } from '../../types/chat';
import './chat.css'


interface ChatroomInfo {
  ID: number;
  Name: string;
}

interface ChatWindowProps {
  chatroom: ChatroomInfo | null;
  messages: ChatMessage[];
  loading: boolean;
  onSendMessage: (text: string) => void;
}

const ChatWindow = ({ chatroom, messages, loading, onSendMessage }: ChatWindowProps) => {
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
      <section className="chat-window chat-window--empty">
        <p>Select a chatroom to start chatting.</p>
      </section>
    );
  }

  return (
    <section className="chat-window">
      <header className="chat-header">
        <h2 className="chat-title">{chatroom.Name}</h2>
      </header>

      <MessageList messages={messages} loading={loading} />

      <MessageComposer draft={draft} onDraftChange={setDraft} onSubmit={handleSubmit} />
    </section>
  );
};

export default ChatWindow;
