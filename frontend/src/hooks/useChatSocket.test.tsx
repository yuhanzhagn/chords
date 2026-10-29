import { TextEncoder, TextDecoder } from "util";
import { webcrypto } from "crypto";
import { act, renderHook } from "@testing-library/react";
import { KafkaEvent } from "../proto/kafka/event";
import { createSocket } from "../services/socket";
import { useChatSocket, selectPendingOwner } from "./useChatSocket";

Object.assign(global, { TextEncoder, TextDecoder, crypto: webcrypto });

jest.mock("../services/socket", () => ({ createSocket: jest.fn() }));

function fakeSocket() {
  return { readyState: WebSocket.OPEN, send: jest.fn(), close: jest.fn(), onopen: null, onmessage: null, onclose: null, onerror: null } as any;
}
const user = { id: 42, username: "alice" };

beforeEach(() => {
 // New test acts like a different account; pending state deliberately survives
 // ordinary unmount/relogin inside one test.
 selectPendingOwner("isolated"); jest.clearAllMocks();
});

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

test("assigns a random request ID once and preserves it across retries", () => {
  const ws = fakeSocket();
  (createSocket as jest.Mock).mockReturnValue(ws);
  const { result } = renderHook(() => useChatSocket(user, "private-token"));
  act(() => ws.onmessage({ data: JSON.stringify({ type: "ready" }) }));
  const payload = KafkaEvent.fromPartial({ msgType: "message", roomId: 7, tempId: "pending", content: new TextEncoder().encode("hello") });
  act(() => { expect(result.current.send(payload)).toBe(true); expect(result.current.send(payload)).toBe(true); });
  const a = KafkaEvent.decode(ws.send.mock.calls[0][0]);
  const b = KafkaEvent.decode(ws.send.mock.calls[1][0]);
  expect(a.requestId).toMatch(/^[0-9a-f]{32}$/);
  expect(b.requestId).toBe(a.requestId);
  expect(b.tempId).toBe("pending");
});

test("rejects oversized UTF-8 content before sending", () => {
  const ws = fakeSocket();
  (createSocket as jest.Mock).mockReturnValue(ws);
  const { result } = renderHook(() => useChatSocket(user, "private-token"));
  act(() => ws.onmessage({ data: JSON.stringify({ type: "ready" }) }));
  const payload = KafkaEvent.fromPartial({ msgType: "message", roomId: 7, content: new TextEncoder().encode("界".repeat(Math.floor(256 * 1024 / 3) + 1)) });
  act(() => { expect(result.current.send(payload)).toBe(false); });
  expect(ws.send).not.toHaveBeenCalled();
  expect(result.current.error).toMatch(/256 KiB/);
});

test("preserves 64-bit server IDs as strings through wire and JSON", () => {
  const e = KafkaEvent.fromPartial({ id: "9007199254740993", serverMessageId: "18446744073709551615", roomSeq:"9223372036854775807", requestId: "r", eventId: "e" });
  const decoded = KafkaEvent.decode(KafkaEvent.encode(e).finish());
  expect(decoded.id).toBe(e.id);
  expect(decoded.serverMessageId).toBe(e.serverMessageId);
  expect(decoded.roomSeq).toBe("9223372036854775807");
  expect(KafkaEvent.fromJSON(KafkaEvent.toJSON(decoded)).serverMessageId).toBe(e.serverMessageId);
});

test("retries a lost ACK on relogin with the same key, stops at persisted and discards on account switch", () => {
 jest.useFakeTimers();const a=fakeSocket(),b=fakeSocket(),c=fakeSocket();
 (createSocket as jest.Mock).mockReturnValueOnce(a).mockReturnValueOnce(b).mockReturnValueOnce(c);
 const callback=jest.fn();const {result,rerender,unmount}=renderHook(({who,token})=>useChatSocket(who,token,callback),{initialProps:{who:user,token:"old"}});
 act(()=>a.onmessage({data:JSON.stringify({type:"ready"})}));
 const event=KafkaEvent.fromPartial({msgType:"message",roomId:7,userId:42,content:new TextEncoder().encode("hello")});
 act(()=>{expect(result.current.send(event)).toBe(true)});const request=event.requestId;
 rerender({who:user,token:"new"});act(()=>b.onmessage({data:JSON.stringify({type:"ready"})}));
 expect(KafkaEvent.decode(b.send.mock.calls[0][0]).requestId).toBe(request);
 act(()=>b.onmessage({data:KafkaEvent.encode(KafkaEvent.fromPartial({msgType:"persisted",requestId:request,userId:42,serverMessageId:"1",roomSeq:"1"})).finish().buffer}));
 b.send.mockClear();act(()=>jest.advanceTimersByTime(60_000));expect(b.send).not.toHaveBeenCalled();
 act(()=>{result.current.send(KafkaEvent.fromPartial({msgType:"message",roomId:7,userId:42}))});
 rerender({who:{id:99,username:"bob"},token:"bob-token"});act(()=>c.onmessage({data:JSON.stringify({type:"ready"})}));
 expect(c.send).not.toHaveBeenCalled();act(()=>a.onclose({code:1008}));expect(result.current.error).toBeNull();
 unmount();jest.useRealTimers();
});
test("bounded unknown-result retries preserve ID; terminal errors stop retries; cancel stops waiting",()=>{
 jest.useFakeTimers();const ws=fakeSocket();(createSocket as jest.Mock).mockReturnValue(ws);const callback=jest.fn();
 const {result,unmount}=renderHook(()=>useChatSocket(user,"token",callback));act(()=>ws.onmessage({data:JSON.stringify({type:"ready"})}));
 const p=KafkaEvent.fromPartial({msgType:"message",roomId:7,userId:42});act(()=>{result.current.send(p)});
 act(()=>jest.advanceTimersByTime(100_000));expect(ws.send).toHaveBeenCalledTimes(5);
 for(const call of ws.send.mock.calls)expect(KafkaEvent.decode(call[0]).requestId).toBe(p.requestId);
 expect(result.current.error).toMatch(/5 attempts/);expect(callback).toHaveBeenCalledWith(expect.objectContaining({msgType:"send_unknown"}));
 act(()=>result.current.retryPending());expect(ws.send).toHaveBeenCalledTimes(6);
 const error=KafkaEvent.fromPartial({msgType:"send_error",userId:42,roomId:7,requestId:p.requestId,content:new TextEncoder().encode(JSON.stringify({code:409,outcome:"rejected",retryable:false}))});
 act(()=>ws.onmessage({data:KafkaEvent.encode(error).finish().buffer}));act(()=>jest.advanceTimersByTime(100_000));expect(ws.send).toHaveBeenCalledTimes(6);
 act(()=>{result.current.send(KafkaEvent.fromPartial({msgType:"message",roomId:7,userId:42}));result.current.cancelPending()});
 const count=ws.send.mock.calls.length;act(()=>jest.advanceTimersByTime(100_000));expect(ws.send).toHaveBeenCalledTimes(count);
 unmount();jest.useRealTimers();
});
