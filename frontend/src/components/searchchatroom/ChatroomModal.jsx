import React from "react";
import "./SearchPage.css"; // CSS for modal

export default function ChatroomModal({ show, onClose, chatroom, onJoin }) {
  if (!show) return null;

  return (
    <div className="modal-backdrop">
      <div className="modal-content">
        <h2>Join Chatroom</h2>
        <p>Do you want to join <strong>{chatroom.name}</strong>?</p>
        <button onClick={() => { onJoin(chatroom); onClose(); }}>Join</button>
        <button onClick={onClose} className="cancel-btn">Cancel</button>
      </div>
    </div>
  );
}

