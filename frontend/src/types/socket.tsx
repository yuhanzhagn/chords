export interface SocketPayload {
  MsgType: string;
  RoomID: number;
  UserID: string | number;
  Message?: string;
  Content?: string;
  TempID: string;
  CreatedAt: number;
  [key: string]: unknown;
}
