import React, { useState, useEffect } from "react";
import ResultList from "./ResultList";
import ChatroomModal from "./ChatroomModal";
import CreateChatroomButton from "./CreateChatroomButton";
import "./SearchPage.css";

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState([]);
  const [selectedRoom, setSelectedRoom] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);
  const ipaddr = `${process.env.REACT_APP_URL}`
  const jwttoken = localStorage.getItem("jwt");

  const storedUser = localStorage.getItem('user');
  const user = storedUser ? JSON.parse(storedUser) : null;


  const handleSearch = async (e) => {
   if (e) e.preventDefault();
//    if (!query) return;
    try {
     let res;
      if (!query){
            res = await fetch(`http://${ipaddr}/chatrooms`, {
            method: "GET",
        headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${jwttoken}`, // <-- JWT here
            },
        });

        //console.log("query nothing")
        }else{
      res = await fetch(`http://${ipaddr}/chatrooms/search?q=${encodeURIComponent(query)}`, {
        method: "GET",
        headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${jwttoken}`, // <-- JWT here
    },
    }); }

      const data = await res.json();
      setResults(data.data);
        //console.log(data.data);
    } catch (err) {
      console.error("Search failed", err);
    }
  };

useEffect(() => { 
    handleSearch();
    return ()=>{};
    } , []);

  const handleRoomClick = (room) => {
    setSelectedRoom(room);
    setModalVisible(true);
  };

  const handleJoin = async (room) => {
    try {
      const res = await fetch(`http://${ipaddr}/memberships/add-user`, {
        method: "POST",
        headers: { "Content-Type": "application/json",
                "Authorization": `Bearer ${jwttoken}`, 
                 },
        body: JSON.stringify({ username: user.username, chatroomid : room.ID })
      });

      if (res.ok) {
        alert(`You have joined chatroom: ${room.Name}`);
      } else {
        const errMsg = await res.text();
        alert(`Failed to join: ${errMsg}`);
      }
    } catch (err) {
      console.error("Join failed", err);
    }
  };

  return (
    <div className="app-container">
      <h1>Search Chatrooms</h1>

      <form onSubmit={handleSearch} className="search-form">
        <input
          type="text"
          placeholder="Search chatrooms..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button type="submit">Search</button>
        <CreateChatroomButton refreshResults={handleSearch}/>
      </form>

      <ResultList results={results} onRoomClick={handleRoomClick} />

      <ChatroomModal
        show={modalVisible}
        onClose={() => setModalVisible(false)}
        chatroom={selectedRoom}
        onJoin={handleJoin}
      />
    </div>
  );
}

