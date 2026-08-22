import * as React from "react";
export interface BatchStatusProps {
  status?: "scheduled" | "running" | "complete" | "failed";
  /** Recorded scope, e.g. "203.0.113.0/24" */
  scope?: string;
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function BatchStatus(props: BatchStatusProps): JSX.Element;
