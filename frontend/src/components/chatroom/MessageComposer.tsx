import type { FormEvent } from 'react';
import './chat.css'

interface MessageComposerProps {
  draft: string;
  onDraftChange: (value: string) => void;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
}

const MessageComposer = ({ draft, onDraftChange, onSubmit }: MessageComposerProps) => (
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
