import type { FormEvent } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface MessageComposerProps {
  draft: string;
  onDraftChange: (value: string) => void;
  onSubmit: (e: FormEvent<HTMLFormElement>) => void;
}

const MessageComposer = ({ draft, onDraftChange, onSubmit }: MessageComposerProps) => (
  <form
    className="flex items-center gap-3 border-t border-border/70 bg-background/40 px-6 py-4"
    onSubmit={onSubmit}
  >
    <Input
      type="text"
      value={draft}
      placeholder="Type a message…"
      onChange={(e) => onDraftChange(e.target.value)}
    />
    <Button type="submit" className="shrink-0">
      Send
    </Button>
  </form>
);

export default MessageComposer;
