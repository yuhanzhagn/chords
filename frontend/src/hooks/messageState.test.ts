import { ChatMessage } from "../types/chat";
import { mergeMessages, reconcileMessages } from "./messageState";
const message = (id: string, roomSeq: string, extra: Partial<ChatMessage> = {}): ChatMessage => ({ ID:id,RoomSeq:roomSeq,UserID:1,RoomID:7,Content:id,CreatedAt:"2026-01-01",TempID:"",status:"sent",fromself:true,...extra });
test("ACK, notification and history upsert in every arrival order without rounding", () => {
 const pending=message("temp","0",{status:"pending",RequestID:"r",TempID:"t"});
 const ack=message("9007199254740993","9007199254740993",{RequestID:"r",TempID:"t",EventID:"e"});
 const history=message(ack.ID as string,ack.RoomSeq!,{RequestID:"r",EventID:"e"});
 for(const events of [[ack,history,ack],[history,ack,history]]) {
  let rows=[pending];for(const e of events)rows=mergeMessages(rows,[e]);
  expect(rows).toHaveLength(1);expect(rows[0]!.ID).toBe("9007199254740993");expect(rows[0]!.status).toBe("sent");
 }
 const rows=mergeMessages([ack],[message("remote","9007199254740992",{UserID:2,fromself:false,TempID:"t"})]);
 expect(rows.map(m=>m.ID)).toEqual(["remote",ack.ID]);
});
test("temp IDs only match the same user's pending and durable messages resist status downgrades",()=>{
 const own=message("own","1",{TempID:"same",RequestID:"r"});
 const remote=message("remote","2",{UserID:2,fromself:false,TempID:"same"});
 expect(mergeMessages([own],[remote])).toHaveLength(2);
 const late=message("temp","0",{RequestID:"r",status:"unknown"});
 expect(mergeMessages([own],[late])[0]!.status).toBe("sent");
});
test("completed authority range removes deletion holes but preserves a realtime event beyond high water",()=>{
 const rows=[message("1","1"),message("2","2"),message("3","3"),message("4","4")];
 expect(reconcileMessages(rows,[rows[0]!],"0","3").map(m=>m.ID)).toEqual(["1","4"]);
});

test("different logical requests never collide on a reused temp ID",()=>{
 const a=message("temp-a","0",{RequestID:"a",TempID:"same",status:"pending"});
 const b=message("temp-b","0",{RequestID:"b",TempID:"same",status:"pending"});
 expect(mergeMessages([a],[b])).toHaveLength(2);
 const ack=message("server-a","1",{RequestID:"a",TempID:"same"});
 expect(mergeMessages([a,b],[ack]).map(m=>m.ID)).toEqual(["server-a","temp-b"]);
});
