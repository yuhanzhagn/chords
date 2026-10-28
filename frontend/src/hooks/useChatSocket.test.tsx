import { TextEncoder, TextDecoder } from "util";
import { act, renderHook } from "@testing-library/react";
import { KafkaEvent } from "../proto/kafka/event";
import { createSocket } from "../services/socket";
import { useChatSocket } from "./useChatSocket";

Object.assign(global, { TextEncoder, TextDecoder });

jest.mock("../services/socket", () => ({ createSocket: jest.fn() }));

function fakeSocket() {
  return { readyState: WebSocket.OPEN, send: jest.fn(), close: jest.fn(), onopen: null, onmessage: null, onclose: null, onerror: null } as any;
}
const user = { id: 42, username: "alice" };

beforeEach(() => jest.clearAllMocks());

test("authenticates in the first frame and joins only after the ready response", () => {
  const ws = fakeSocket();
  (createSocket as jest.Mock).mockReturnValue(ws);
  const { result, unmount } = renderHook(() => useChatSocket(user, "private-token"));
  act(() => result.current.join(7, user.id));
  expect(ws.send).not.toHaveBeenCalled();
  act(() => ws.onopen());
  expect(JSON.parse(ws.send.mock.calls[0][0])).toEqual({ type: "auth", token: "private-token" });
  act(() => ws.onmessage({ data: JSON.stringify({ type: "ready" }) }));
  const event = KafkaEvent.decode(ws.send.mock.calls[1][0]);
  expect(event).toMatchObject({ msgType: "join", roomId: 7, userId: 42 });
  expect(result.current.error).toBeNull();
  unmount();
  expect(ws.close).toHaveBeenCalled();
});

test("does not silently enqueue a message before authentication", () => {
  const ws = fakeSocket();
  (createSocket as jest.Mock).mockReturnValue(ws);
  const { result } = renderHook(() => useChatSocket(user, "private-token"));
  let sent = true;
  act(() => { sent = result.current.send(KafkaEvent.fromPartial({ msgType: "message", roomId: 7 })); });
  expect(sent).toBe(false);
  expect(ws.send).not.toHaveBeenCalled();
  expect(result.current.error).toMatch(/not ready/);
});

test("shows access rejection and stops retrying invalid credentials", () => {
  jest.useFakeTimers();
  const ws = fakeSocket();
  (createSocket as jest.Mock).mockReturnValue(ws);
  const { result, unmount } = renderHook(() => useChatSocket(user, "private-token"));
  act(() => ws.onclose({ code: 1008 }));
  expect(result.current.error).toMatch(/Session expired or room access denied/);
  act(() => jest.advanceTimersByTime(60_000));
  expect(createSocket).toHaveBeenCalledTimes(1);
  unmount();
  jest.useRealTimers();
});

test("ignores stale connection callbacks after credentials change", () => {
  const oldSocket = fakeSocket();
  const newSocket = fakeSocket();
  (createSocket as jest.Mock).mockReturnValueOnce(oldSocket).mockReturnValueOnce(newSocket);
  const { result, rerender } = renderHook(({ token }) => useChatSocket(user, token), { initialProps: { token: "old" } });
  rerender({ token: "new" });
  act(() => newSocket.onmessage({ data: JSON.stringify({ type: "ready" }) }));
  act(() => oldSocket.onclose({ code: 1008 }));
  let sent = false;
  act(() => { sent = result.current.send(KafkaEvent.fromPartial({ msgType: "message", roomId: 7, userId: 42 })); });
  expect(sent).toBe(true);
  expect(result.current.error).toBeNull();
});
