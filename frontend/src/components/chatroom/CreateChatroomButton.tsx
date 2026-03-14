import { useState, useContext } from 'react';
import { RefreshContext } from './RefreshContext';
import { Button } from '../ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '../ui/dialog';
import { Input } from '../ui/input';

function CreateChatroomButton() {
  const [open, setOpen] = useState(false);
  const [chatroomName, setChatroomName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const triggerParentRefresh = useContext(RefreshContext);
  const jwttoken = localStorage.getItem("jwt");
  const ipaddr = `${process.env.REACT_APP_URL}`;

  const openPopup = () => {
    setChatroomName("");
    setOpen(true);
  };

  const closePopup = () => {
    setOpen(false);
    setError(null);
  };

  const handleCreate = async () => {
    if (!chatroomName.trim()) {
      setError("Chatroom name cannot be empty.");
      return;
    }
    triggerParentRefresh();
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

      triggerParentRefresh();
      closePopup();
    } catch (err: unknown) {
      console.error(err);
      setError(err instanceof Error ? err.message : 'Failed to create chatroom');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? openPopup() : closePopup())}>
      <DialogTrigger asChild>
        <Button type="button" variant="secondary">Create Chatroom</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create New Chatroom</DialogTitle>
          <DialogDescription>Spin up a new space for your team.</DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Input
            type="text"
            placeholder="Enter chatroom name"
            value={chatroomName}
            onChange={(e) => setChatroomName(e.target.value)}
          />
          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={closePopup} disabled={loading}>
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
