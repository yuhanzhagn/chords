import './chat.css'
import { ChatMessage } from '../../types/chat';

const getMessageClass = (msg: ChatMessage): string => {
  if (msg.status === "failed") return "chat-message-self failed";
  if (msg.status === "pending") return "chat-message-self pending";
  return msg.fromself ? "chat-message-self" : "chat-message-others";
};

interface MessageListProps {
  messages: ChatMessage[];
  loading: boolean;
}

const MessageList = ({ messages, loading }: MessageListProps) => {
  console.log("Rendering MessageList with messages:", messages);
  return (
  <div className="chat-messages">
    {loading && <p className="chat-loading">Loading messages…</p>}

    {!loading && messages.length === 0 && <p>No messages yet. Say hello!</p>}

    {messages.map((msg) => (
      <article key={msg.ID || msg.CreatedAt || msg.TempID} className={getMessageClass(msg)}>
        <div className="chat-message-header">
          <span className="chat-message-author">{msg.UserID || "Anonymous"}</span>
          <span className="chat-message-timestamp">
            {msg.CreatedAt ? new Date(msg.CreatedAt).toLocaleTimeString() : ''}
          </span>
        </div>
        <p className="chat-message-body">{msg.Content || ''}</p>
      </article>
    ))}
  </div>
)};

export default MessageList;
