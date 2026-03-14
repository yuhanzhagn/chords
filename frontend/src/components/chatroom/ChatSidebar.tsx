import { cn } from "../../lib/utils";

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

const ChatSidebar = ({ chatrooms, loading, selectedChatroom, onSelectChatroom }: ChatSidebarProps) => (
  <aside className="flex flex-col border-b border-border/70 bg-secondary/30 p-4 lg:border-b-0 lg:border-r">
    <div className="flex items-center justify-between">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        Chat Rooms
      </h2>
      <span className="text-xs text-muted-foreground">{chatrooms.length}</span>
    </div>

    {loading && <p className="mt-3 text-sm text-muted-foreground">Loading…</p>}

    {!loading && chatrooms.length === 0 && (
      <p className="mt-3 text-sm text-muted-foreground">No chatrooms found.</p>
    )}

    <div className="mt-4 flex flex-1 flex-col gap-2 overflow-y-auto">
      {chatrooms.map((room) => {
        const isActive = selectedChatroom?.ID === room.ID;

        return (
          <button
            key={room.ID}
            type="button"
            className={cn(
              "rounded-xl border px-3 py-2 text-left text-sm font-medium transition",
              isActive
                ? "border-primary/60 bg-primary text-primary-foreground shadow"
                : "border-border/70 bg-background/40 text-foreground hover:bg-secondary/60"
            )}
            onClick={() => onSelectChatroom(room)}
          >
            {room.Name}
          </button>
        );
      })}
    </div>
  </aside>
);

export default ChatSidebar;
