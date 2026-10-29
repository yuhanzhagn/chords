import { act, renderHook } from "@testing-library/react";
import { apiFetch } from "../services/api";
import { useMessages } from "./useMessages";
import { ChatMessage } from "../types/chat";
jest.mock("../services/api",()=>({apiFetch:jest.fn()}));
const row=(n:number,room=7)=>({ID:n,server_message_id:String(n),room_seq:String(n),UserID:2,RoomID:room,Content:String(n),CreatedAt:"2026-01-01"});
const notice=(n:number):ChatMessage=>({ID:String(n),RoomSeq:String(n),UserID:2,RoomID:7,Content:String(n),CreatedAt:"2026-01-01",TempID:"",status:"sent",fromself:false});
const flush=async()=>{await act(async()=>{for(let i=0;i<30;i++)await Promise.resolve()})};
beforeEach(()=>{jest.useFakeTimers();jest.clearAllMocks()});afterEach(()=>jest.useRealTimers());
test("catches up multiple offline pages and a lost final notification while still connected; reconciles deletes",async()=>{
 let high=75;let server=Array.from({length:75},(_,i)=>row(i+1));
 (apiFetch as jest.Mock).mockImplementation(async(path:string)=>{
  const query=new URLSearchParams(path.split("?")[1]);const snapshot=query.get("snapshot_seq");const ceiling=snapshot?Number(snapshot):high;
  let rows=server.filter(m=>Number(m.room_seq)<=ceiling);const before=query.get("before_seq");const after=query.get("after_seq");
  if(before!==null){if(before!=="0")rows=rows.filter(m=>Number(m.room_seq)<Number(before));rows=rows.slice().reverse()}
  else rows=rows.filter(m=>Number(m.room_seq)>Number(after));
  const more=rows.length>30;rows=rows.slice(0,30);const cursor=before!==null?(rows[rows.length-1]?.room_seq||"0"):more?rows[rows.length-1]!.room_seq:String(ceiling);
  return {messages:before!==null?rows.reverse():rows,high_water:String(ceiling),next_cursor:cursor,has_more:more};
 });
 const {result,unmount}=renderHook(()=>useMessages(7,1,"token"));await flush();expect(result.current.messages).toHaveLength(30);
 server=Array.from({length:145},(_,i)=>row(i+1));high=145;
 act(()=>jest.advanceTimersByTime(15_000));await flush();expect(result.current.messages).toHaveLength(100);expect(result.current.messages[99]!.RoomSeq).toBe("145");
 const calls=(apiFetch as jest.Mock).mock.calls.map(c=>c[0] as string);expect(calls.some(c=>c.includes("after_seq=75&snapshot_seq=145"))).toBe(true);
 // A last event is lost with no disconnect and no following event to reveal a gap.
 high=146;server.push(row(146));act(()=>jest.advanceTimersByTime(15_000));await flush();expect(result.current.messages[100]!.RoomSeq).toBe("146");
 server=server.filter(m=>![80,146].includes(m.ID));act(()=>jest.advanceTimersByTime(15_000));await flush();expect(result.current.messages.some(m=>["80","146"].includes(String(m.ID)))).toBe(false);
 // A delayed notification cannot permanently revive a deleted tail.
 act(()=>result.current.confirm(notice(146)));await flush();expect(result.current.messages.some(m=>m.ID==="146")).toBe(false);
 act(()=>{result.current.loadMore()});await flush();expect(result.current.messages[0]!.RoomSeq).toBe("16");
 unmount();
});
test("ignores late old-room and old-account requests and callbacks; live rows survive an initial response",async()=>{
 let resolveOld:(v:any)=>void=()=>{};
 (apiFetch as jest.Mock).mockImplementation((path:string)=>path.includes("/7/")?new Promise(resolve=>{resolveOld=resolve}):Promise.resolve({messages:[row(1,8)],high_water:"1",next_cursor:"1",has_more:false}));
 const {result,rerender,unmount}=renderHook(({room,user,token})=>useMessages(room,user,token),{initialProps:{room:7,user:1,token:"a"}});
 const oldConfirm=result.current.confirm;
 rerender({room:8,user:2,token:"b"});await flush();
 act(()=>resolveOld({messages:[row(99)],high_water:"99",next_cursor:"99",has_more:false}));await flush();
 act(()=>oldConfirm(notice(100)));expect(result.current.messages.every(m=>m.RoomID===8)).toBe(true);
 unmount();
});
test("history arriving after a live receipt merges pending by request and never replaces newer realtime",async()=>{
 let resolveInitial:(v:any)=>void=()=>{};
 (apiFetch as jest.Mock).mockImplementationOnce(()=>new Promise(resolve=>{resolveInitial=resolve})).mockResolvedValue({messages:[row(1),{...row(2),UserID:1,request_id:"r"}],high_water:"2",next_cursor:"2",has_more:false});
 const {result,unmount}=renderHook(()=>useMessages(7,1,"token"));
 act(()=>result.current.add({...notice(0),ID:"temp",UserID:1,RequestID:"r",TempID:"t",status:"pending",fromself:true}));
 act(()=>result.current.confirm({...notice(2),UserID:1,RequestID:"r",TempID:"t",fromself:true}));
 act(()=>resolveInitial({messages:[row(1)],high_water:"1",next_cursor:"0",has_more:false}));await flush();
 expect(result.current.messages.map(m=>m.ID)).toEqual(["1","2"]);expect(result.current.messages.every(m=>m.status==="sent")).toBe(true);
 unmount();
});

test("the first render after switching rooms or accounts never exposes the previous history",async()=>{
 (apiFetch as jest.Mock).mockImplementation(async(path:string)=>{
  const room=path.includes("/7/")?7:8;
  return {messages:[row(1,room)],high_water:"1",next_cursor:"1",has_more:false};
 });
 const snapshots:{room:number;messageRooms:number[]}[]=[];
 const {result,rerender,unmount}=renderHook(({room,user,token})=>{
  const store=useMessages(room,user,token);snapshots.push({room,messageRooms:store.messages.map(m=>m.RoomID)});return store;
 },{initialProps:{room:7,user:1,token:"alice"}});
 await flush();expect(result.current.messages).toHaveLength(1);
 rerender({room:8,user:2,token:"bob"});await flush();
 expect(snapshots.filter(s=>s.room===8).every(s=>s.messageRooms.every(room=>room===8))).toBe(true);
 unmount();
});
