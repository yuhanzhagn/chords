export const SOCKET_URL = "ws://localhost:8000/ws";

export function createSocket() {
  return new WebSocket(SOCKET_URL);
}
