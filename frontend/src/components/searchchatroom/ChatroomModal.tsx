import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';

interface Room {
  ID: number;
  Name: string;
  name?: string;
}

interface ChatroomModalProps {
  show: boolean;
  onClose: () => void;
  chatroom: Room | null;
  onJoin: (room: Room) => void;
}

export default function ChatroomModal({ show, onClose, chatroom, onJoin }: ChatroomModalProps) {
  if (!chatroom) return null;

  return (
    <Dialog open={show} onOpenChange={(open) => (!open ? onClose() : undefined)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Join Chatroom</DialogTitle>
          <DialogDescription>
            Do you want to join <strong>{chatroom.Name || chatroom.name}</strong>?
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={() => { onJoin(chatroom); onClose(); }}>Join</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
