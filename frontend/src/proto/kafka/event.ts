/* eslint-disable */
// Code generated from proto/kafka/event.proto. DO NOT EDIT.

export interface KafkaEvent {
  ID: number;
  UserID: number;
  RoomID: number;
  MsgType: string;
  Content: Uint8Array;
  TempID: string;
  CreateAt: number;
}

export const KafkaEventSchema = {
  fromJSON(object: any): KafkaEvent {
    return {
      ID: isSet(object.ID) ? Number(object.ID) : 0,
      UserID: isSet(object.UserID) ? Number(object.UserID) : 0,
      RoomID: isSet(object.RoomID) ? Number(object.RoomID) : 0,
      MsgType: isSet(object.MsgType) ? String(object.MsgType) : "",
      Content: isSet(object.Content) ? bytesFromBase64(String(object.Content)) : new Uint8Array(0),
      TempID: isSet(object.TempID) ? String(object.TempID) : "",
      CreateAt: isSet(object.CreateAt) ? Number(object.CreateAt) : 0,
    };
  },

  toJSON(message: KafkaEvent): unknown {
    return {
      ID: Math.round(message.ID),
      UserID: Math.round(message.UserID),
      RoomID: Math.round(message.RoomID),
      MsgType: message.MsgType,
      Content: base64FromBytes(message.Content),
      TempID: message.TempID,
      CreateAt: Math.round(message.CreateAt),
    };
  },

  create(base?: Partial<KafkaEvent>): KafkaEvent {
    return {
      ID: base?.ID ?? 0,
      UserID: base?.UserID ?? 0,
      RoomID: base?.RoomID ?? 0,
      MsgType: base?.MsgType ?? "",
      Content: base?.Content ?? new Uint8Array(0),
      TempID: base?.TempID ?? "",
      CreateAt: base?.CreateAt ?? 0,
    };
  },
};

const isSet = (value: unknown): boolean => value !== null && value !== undefined;

function bytesFromBase64(b64: string): Uint8Array {
  if (typeof atob === "function") {
    const bin = atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i += 1) arr[i] = bin.charCodeAt(i);
    return arr;
  }

  const buf = Buffer.from(b64, "base64");
  return new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength);
}

function base64FromBytes(arr: Uint8Array): string {
  if (typeof btoa === "function") {
    let str = "";
    arr.forEach((byte) => {
      str += String.fromCharCode(byte);
    });
    return btoa(str);
  }

  return Buffer.from(arr).toString("base64");
}
