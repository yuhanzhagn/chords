export const SOCKET_URL = "ws://localhost:8000/ws";

export function createSocket() {
  const socket = new WebSocket(SOCKET_URL);

  socket.binaryType = "arraybuffer";

  return socket;
}
