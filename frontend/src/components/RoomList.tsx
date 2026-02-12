import { useState } from "react";
import ChatRoom from "./chatroom/ChatRoom";

function App() {
  const [userId, setUserId] = useState("");
  const [roomId, setRoomId] = useState("");
  const [joined, setJoined] = useState(false);

  const handleJoin = () => {
    if (userId && roomId) {
      setJoined(true);
    }
  };

  if (!joined) {
    return (
      <div style={{ padding: 20 }}>
        <h1>Join Chat Room</h1>
        <input
          placeholder="User ID"
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          style={{ marginRight: 10 }}
        />
        <input
          placeholder="Room ID"
          value={roomId}
          onChange={(e) => setRoomId(e.target.value)}
          style={{ marginRight: 10 }}
        />
        <button onClick={handleJoin}>Join</button>
      </div>
    );
  }

  return <ChatRoom roomId={roomId} userId={userId} />;
}

export default App;
