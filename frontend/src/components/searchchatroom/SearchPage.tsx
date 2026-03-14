import { useState, useEffect, useCallback } from 'react';
import type { FormEvent } from 'react';
import ResultList from "./ResultList";
import ChatroomModal from "./ChatroomModal";
import CreateChatroomButton from "./CreateChatroomButton";
import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';
import { Input } from '../ui/input';

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
    try {
      let res: Response;
      if (!query) {
        res = await fetch(`http://${ipaddr}/chatrooms`, {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${jwttoken}`,
          },
        });
      } else {
        res = await fetch(`http://${ipaddr}/chatrooms/search?q=${encodeURIComponent(query)}`, {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${jwttoken}`,
          },
        });
      }

      const data: SearchResponse = await res.json();
      setResults(data.data);
    } catch (err: unknown) {
      console.error('Search failed', err);
    }
  }, [ipaddr, jwttoken, query]);

  useEffect(() => {
    handleSearch();
    return () => {};
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
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${jwttoken}`,
        },
        body: JSON.stringify({ username: user.username, chatroomid: room.ID })
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
    <div className="space-y-6">
      <Card className="border-border/70 bg-card/90">
        <CardHeader>
          <CardTitle>Search Chatrooms</CardTitle>
          <CardDescription>Find active rooms or create a fresh one.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSearch} className="flex flex-wrap gap-3">
            <div className="min-w-[220px] flex-1">
              <Input
                type="text"
                placeholder="Search chatrooms..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
              />
            </div>
            <Button type="submit">Search</Button>
            <CreateChatroomButton refreshResults={handleSearch} />
          </form>
        </CardContent>
      </Card>

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
