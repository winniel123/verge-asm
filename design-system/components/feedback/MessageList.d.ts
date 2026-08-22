import * as React from "react";
export interface Message {
  id: string;
  /** Message class, e.g. "signals", "coverage" */
  cls: string;
  text: string;
  /** Terse relative time */
  time: string;
  unread?: boolean;
}
export interface MessageListProps {
  messages: Message[];
  onOpen?: (m: Message) => void;
  style?: React.CSSProperties;
}
export function MessageList(props: MessageListProps): JSX.Element;
