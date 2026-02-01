import React from "react";

const MessageComposer = ({ draft, onDraftChange, onSubmit }) => (
  <form className="chat-composer" onSubmit={onSubmit}>
    <input
      type="text"
      value={draft}
      placeholder="Type a message…"
      onChange={(e) => onDraftChange(e.target.value)}
      className="chat-input"
    />
    <button type="submit" className="chat-send-button">
      Send
    </button>
  </form>
);

export default MessageComposer;
