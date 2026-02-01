import React from "react";

const getMessageClass = (msg) => {
  if (msg.status === "failed") return "chat-message-self failed";
  if (msg.status === "pending") return "chat-message-self pending";
  return msg.fromself ? "chat-message-self" : "chat-message-others";
};

const MessageList = ({ messages, loading }) => (
  <div className="chat-messages">
    {loading && <p className="chat-loading">Loading messages…</p>}

    {!loading && messages.length === 0 && <p>No messages yet. Say hello!</p>}

    {messages.map((msg) => (
      <article key={msg.ID || msg._id || msg.timestamp} className={getMessageClass(msg)}>
        <div className="chat-message-header">
          <span className="chat-message-author">{msg.UserID || "Anonymous"}</span>
          <span className="chat-message-timestamp">
            {msg.CreatedAt ? new Date(msg.createdAt).toLocaleTimeString() : ""}
          </span>
        </div>
        <p className="chat-message-body">{msg.Content}</p>
      </article>
    ))}
  </div>
);

export default MessageList;
