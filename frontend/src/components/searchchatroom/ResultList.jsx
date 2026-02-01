import React from "react";
import "./SearchPage.css"; // optional CSS for results

export default function ResultList({ results, onRoomClick }) {
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

