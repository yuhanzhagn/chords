import { useState } from 'react';
import { Button } from '../ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog';
import { Input } from '../ui/input';

interface CreateChatroomButtonProps {
  refreshResults: () => void;
}

function CreateChatroomButton({ refreshResults }: CreateChatroomButtonProps) {
  const [open, setOpen] = useState(false);
  const [chatroomName, setChatroomName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const jwttoken = localStorage.getItem("jwt");
  const ipaddr = `${process.env.REACT_APP_URL}`;

  const handleCreate = async () => {
    if (!chatroomName.trim()) {
      setError("Chatroom name cannot be empty.");
      return;
    }
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
      setOpen(false);
      setChatroomName("");
    } catch (err: unknown) {
      console.error(err);
      setError(err instanceof Error ? err.message : 'Failed to create chatroom');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button type="button" variant="outline">Create Chatroom</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Chatroom</DialogTitle>
          <DialogDescription>Give your new space a friendly name.</DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Input
            type="text"
            placeholder="Enter chatroom name"
            value={chatroomName}
            onChange={(e) => setChatroomName(e.target.value)}
          />
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={loading}>
            {loading ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default CreateChatroomButton;
