export interface ChatMessage {
  ID?: number;
  _id?: string;
  timestamp?: string;
  UserID?: string | number;
  RoomID?: string | number;
  Content?: string;
  status?: 'sent' | 'pending' | 'failed' | string;
  fromself?: boolean;
  CreatedAt?: string;
  createdAt?: string;
  TempID?: string;
}

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
      <article key={msg.ID || msg._id || msg.timestamp || msg.TempID} className={getMessageClass(msg)}>
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
