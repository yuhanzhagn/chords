import { useEffect, useState, useRef } from 'react';
import type { KeyboardEvent } from 'react';
import { Button } from '../ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card';
import { Input } from '../ui/input';
import { ScrollArea } from '../ui/scroll-area';

interface ChatRoomBackupProps {
  roomId: string;
  userId: string;
}

const ChatRoom = ({ roomId, userId }: ChatRoomBackupProps) => {
  const [messages, setMessages] = useState<string[]>([]);
  const [input, setInput] = useState('');
  const ws = useRef<WebSocket | null>(null);

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

  const handleKeyPress = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      sendMessage();
    }
  };

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader>
        <CardTitle>Chat Room #{roomId}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <ScrollArea className="h-72 rounded-xl border border-border/70 bg-background/40">
          <div className="space-y-2 p-3 text-sm">
            {messages.map((msg, index) => (
              <div key={index} className="rounded-lg border border-border/60 bg-secondary/40 px-3 py-2">
                {msg}
              </div>
            ))}
          </div>
        </ScrollArea>
        <div className="flex gap-2">
          <Input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyPress}
            placeholder="Type a message..."
          />
          <Button onClick={sendMessage}>Send</Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default ChatRoom;
