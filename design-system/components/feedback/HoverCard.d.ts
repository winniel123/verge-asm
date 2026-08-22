import * as React from "react";
export interface HoverCardProps {
  /** The peek panel contents */
  content: React.ReactNode;
  /** ms before showing. Default 350 */
  delay?: number;
  /** Preferred side; flips automatically. Default "bottom" */
  side?: "top" | "bottom";
  children?: React.ReactNode;
  style?: React.CSSProperties;
}
export function HoverCard(props: HoverCardProps): JSX.Element;
