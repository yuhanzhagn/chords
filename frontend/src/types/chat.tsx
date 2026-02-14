export interface Chatroom {
  ID: number;
  Name: string;
}

export interface ChatMessage {
  ID: number;
  UserID: string | number;
  RoomID: number;
  Content: string;
  CreatedAt: string;
  TempID: string;
  status: "pending" | "sent";
  fromself: boolean;
}

export interface ChatroomResponse {
  data: Chatroom[];
}

export interface ChatroomProps {
  roomId?: string;
  userId?: string;
}

