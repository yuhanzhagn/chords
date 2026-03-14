import { Button } from '../ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card';

interface Room {
  ID: number;
  Name: string;
}

interface ResultListProps {
  results: Room[];
  onRoomClick: (room: Room) => void;
}

export default function ResultList({ results, onRoomClick }: ResultListProps) {
  return (
    <Card className="border-border/70 bg-card/90">
      <CardHeader>
        <CardTitle>Rooms</CardTitle>
        <CardDescription>
          {results.length === 0
            ? "No rooms yet. Try a different search."
            : `${results.length} room${results.length === 1 ? "" : "s"} available.`}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        {results.map(room => (
          <Button
            key={room.ID}
            variant="secondary"
            className="w-full justify-between"
            onClick={() => onRoomClick(room)}
          >
            <span>{room.Name}</span>
            <span className="text-xs text-muted-foreground">#{room.ID}</span>
          </Button>
        ))}
      </CardContent>
    </Card>
  );
}
