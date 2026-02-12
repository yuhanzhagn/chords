import "./SearchPage.css"; // optional CSS for results

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
    <ul className="results-list">
      {results.map(room => (
        <li
          key={room.ID}
          className="result-item"
          onClick={() => onRoomClick(room)}
        >
          {room.Name}
        </li>
      ))}
    </ul>
  );
}
