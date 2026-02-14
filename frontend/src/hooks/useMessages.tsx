import { useReducer } from "react";
import {ChatMessage} from "../types/chat";

type Action =
  | { type: "LOAD"; payload: ChatMessage[] }
  | { type: "ADD"; payload: ChatMessage }
  | { type: "CONFIRM"; payload: ChatMessage }
  | { type: "CLEAR" };

function reducer(state: ChatMessage[], action: Action): ChatMessage[] {
  switch (action.type) {
    case "LOAD":
      return action.payload;
    case "ADD":
      return [...state, action.payload];
    case "CONFIRM":
      return state.map((m) =>
        m.TempID === action.payload.TempID ? action.payload : m,
      );
    case "CLEAR":
      return [];
    default:
      return state;
  }
}

export function useMessages() {
  const [messages, dispatch] = useReducer(reducer, []);

  return {
    messages,
    load: (m: ChatMessage[]) => dispatch({ type: "LOAD", payload: m }),
    add: (m: ChatMessage) => dispatch({ type: "ADD", payload: m }),
    confirm: (m: ChatMessage) => dispatch({ type: "CONFIRM", payload: m }),
    clear: () => dispatch({ type: "CLEAR" }),
  };
}
