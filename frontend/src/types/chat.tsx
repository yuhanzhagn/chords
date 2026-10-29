export interface Chatroom {
  ID: number;
  Name: string;
}

export interface ChatMessage {
  ID: string | number;
  UserID: string | number;
  RoomID: number;
  Content: string;
  CreatedAt: string;
  TempID: string;
  RequestID?: string;
  EventID?: string;
  RoomSeq?: string;
  WaitingCanceled?: boolean;
  status: "pending" | "sent" | "failed" | "unknown";
  fromself: boolean;
}

export interface ChatroomResponse {
  data: Chatroom[];
}

export interface ChatroomProps {
  roomId?: string;
  userId?: string;
}

