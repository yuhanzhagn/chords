import React, { useState, useContext} from "react";
//import { RefreshContext } from "./RefreshContext"

function CreateChatroomButton({refreshResults}) {
  const [showPopup, setShowPopup] = useState(false);
  const [chatroomName, setChatroomName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  //const triggerParentRefresh = useContext(RefreshContext);
  const jwttoken = localStorage.getItem("jwt");
  const ipaddr = `${process.env.REACT_APP_URL}`

  const openPopup = () => {
    setChatroomName("");
    setShowPopup(true);
  };

  const closePopup = () => {
    setShowPopup(false);
    setError(null);
  };

  const handleCreate = async () => {
    if (!chatroomName.trim()) {
      setError("Chatroom name cannot be empty.");
      return;
    }
    //triggerParentRefresh();
    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`http://${ipaddr}/chatrooms`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
           "Authorization": `Bearer ${jwttoken}`,
        },
        body: JSON.stringify({ name: chatroomName }),
      });

      if (!res.ok) {
        throw new Error(`Server error: ${res.status}`);
      }

      const data = await res.json();
      console.log("Chatroom created:", data);
      
      refreshResults();
      //triggerParentRefresh();
      // Close popup after successful creation
      closePopup();
    } catch (err) {
      console.error(err);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <button onClick={openPopup}>Create Chatroom</button>

      {showPopup && (
        <div style={styles.overlay}>
          <div style={styles.popup}>
            <h3>Create New Chatroom</h3>

            <input
              type="text"
              placeholder="Enter chatroom name"
              value={chatroomName}
              onChange={(e) => setChatroomName(e.target.value)}
              style={styles.input}
            />

            {error && <p style={{ color: "red" }}>{error}</p>}

            <div style={styles.actions}>
              <button onClick={handleCreate} disabled={loading}>
                {loading ? "Creating..." : "Create"}
              </button>
              <button onClick={closePopup}>Cancel</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

const styles = {
  overlay: {
    position: "fixed",
    top: 0,
    left: 0,
    width: "100vw",
    height: "100vh",
    background: "rgba(0,0,0,0.5)",
    display: "flex",
    justifyContent: "center",
    alignItems: "center",
  },
  popup: {
    background: "white",
    padding: "20px",
    borderRadius: "8px",
    width: "300px",
    display: "flex",
    flexDirection: "column",
    gap: "10px",
  },
  input: {
    padding: "8px",
    fontSize: "14px",
  },
  actions: {
    display: "flex",
    justifyContent: "space-between",
  },
};

export default CreateChatroomButton;

