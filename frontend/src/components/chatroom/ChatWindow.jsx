import React, { useEffect, useState } from "react";
import MessageList from "./MessageList";
import MessageComposer from "./MessageComposer";

const ChatWindow = ({ chatroom, messages, loading, onSendMessage }) => {
  const [draft, setDraft] = useState("");

  useEffect(() => {
    setDraft("");
  }, [chatroom?.id]);
  const handleSubmit = (e) => {
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
        <h2 className="chat-title">{chatroom.name}</h2>
      </header>

      <MessageList messages={messages} loading={loading} />

      <MessageComposer draft={draft} onDraftChange={setDraft} onSubmit={handleSubmit} />
    </section>
  );
};

export default ChatWindow;
