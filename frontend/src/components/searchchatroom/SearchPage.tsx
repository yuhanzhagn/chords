import { useState, useEffect, useCallback } from 'react';
import type { FormEvent } from 'react';
import ResultList from "./ResultList";
import ChatroomModal from "./ChatroomModal";
import CreateChatroomButton from "./CreateChatroomButton";
import "./SearchPage.css";

interface UserInfo {
  username: string;
}

interface Room {
  ID: number;
  Name: string;
}

interface SearchResponse {
  data: Room[];
}

export default function SearchPage() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Room[]>([]);
  const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const ipaddr = `${process.env.REACT_APP_URL}`;
  const jwttoken = localStorage.getItem('jwt');

  const storedUser = localStorage.getItem('user');
  const user: UserInfo | null = storedUser ? JSON.parse(storedUser) : null;


  const handleSearch = useCallback(async (e?: FormEvent<HTMLFormElement>) => {
    if (e) e.preventDefault();
//    if (!query) return;
    try {
      let res: Response;
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

      const data: SearchResponse = await res.json();
      setResults(data.data);
        //console.log(data.data);
    } catch (err: unknown) {
      console.error('Search failed', err);
    }
  }, [ipaddr, jwttoken, query]);

useEffect(() => {
    handleSearch();
    return ()=>{};
    }, [handleSearch]);

  const handleRoomClick = (room: Room) => {
    setSelectedRoom(room);
    setModalVisible(true);
  };

  const handleJoin = async (room: Room) => {
    if (!user) return;
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
    } catch (err: unknown) {
      console.error('Join failed', err);
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
