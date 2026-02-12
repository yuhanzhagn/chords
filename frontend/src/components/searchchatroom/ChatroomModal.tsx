import "./SearchPage.css"; // CSS for modal

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
  if (!show) return null;
  if (!chatroom) return null;

  return (
    <div className="modal-backdrop">
      <div className="modal-content">
        <h2>Join Chatroom</h2>
        <p>Do you want to join <strong>{chatroom.Name || chatroom.name}</strong>?</p>
        <button onClick={() => { onJoin(chatroom); onClose(); }}>Join</button>
        <button onClick={onClose} className="cancel-btn">Cancel</button>
      </div>
    </div>
  );
}
