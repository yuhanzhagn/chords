import React, { useEffect, useState, useRef } from "react";

const ChatRoom = ({ roomId, userId }) => {
  const [messages, setMessages] = useState([]);
  const [input, setInput] = useState("");
  const ws = useRef(null);

  useEffect(() => {
    // Connect to your Go WebSocket backend
    const socket = new WebSocket(`ws://${process.env.REACT_APP_URL}/ws/${roomId}/${userId}`);
    ws.current = socket;

    socket.onopen = () => {
      console.log(`? Connected to chatroom ${roomId}`);
    };

    socket.onmessage = (event) => {
      setMessages((prev) => [...prev, event.data]);
    };

    socket.onclose = () => {
      console.log("? Disconnected");
    };

    return () => {
      socket.close();
    };
  }, [roomId, userId]);

  const sendMessage = () => {
    if (ws.current && input.trim() !== "") {
      ws.current.send(input);
      setInput("");
    }
  };

  const handleKeyPress = (e) => {
    if (e.key === "Enter") {
      sendMessage();
    }
  };

  return (
    <div style={styles.container}>
      <h2>Chat Room #{roomId}</h2>
      <div style={styles.chatBox}>
        {messages.map((msg, index) => (
          <div key={index} style={styles.message}>
            {msg}
          </div>
        ))}
      </div>
      <div style={styles.inputBox}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyPress}
          placeholder="Type a message..."
          style={styles.input}
        />
        <button onClick={sendMessage} style={styles.button}>
          Send
        </button>
      </div>
    </div>
  );
};

const styles = {
  container: {
    width: "400px",
    margin: "0 auto",
    fontFamily: "Arial, sans-serif",
  },
  chatBox: {
    border: "1px solid #ccc",
    height: "300px",
    overflowY: "auto",
    padding: "10px",
    marginBottom: "10px",
  },
  message: {
    padding: "5px 0",
    borderBottom: "1px solid #eee",
  },
  inputBox: {
    display: "flex",
  },
  input: {
    flex: 1,
    padding: "8px",
  },
  button: {
    padding: "8px 12px",
    marginLeft: "5px",
    cursor: "pointer",
  },
};

export default ChatRoom;

