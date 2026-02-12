interface ChatroomSummary {
  ID: number;
  Name: string;
}

interface ChatSidebarProps {
  chatrooms: ChatroomSummary[];
  loading: boolean;
  selectedChatroom: ChatroomSummary | null;
  onSelectChatroom: (room: ChatroomSummary) => void;
}
//import CreateChatroomButton from "./CreateChatroomButton";

const ChatSidebar = ({ chatrooms, loading, selectedChatroom, onSelectChatroom }: ChatSidebarProps) => (
  <aside className="chat-sidebar">
    <h2 className="chat-sidebar-title">Chat Rooms</h2>

    {loading && <p className="chat-loading">Loading…</p>}

    {!loading && chatrooms.length === 0 && <p>No chatrooms found.</p>}

    <ul className="chatroom-list">
      {chatrooms.map((room) => {
        const isActive = selectedChatroom?.ID === room.ID;

        return (
          <li
            key={room.ID}
            className={`chatroom-item ${isActive ? "chatroom-item--active" : ""}`}
            onClick={() => onSelectChatroom(room)}
          >
            {room.Name}
          </li>
        );
      })}
    </ul>
  </aside>
);

export default ChatSidebar;
