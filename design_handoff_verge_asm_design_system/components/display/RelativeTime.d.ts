import * as React from "react";
export interface RelativeTimeProps {
  /** Terse relative time: "4m", "1h", "3d" */
  value: string;
  /** Absolute ISO 8601 shown on hover */
  iso: string;
  side?: "top" | "bottom" | "left" | "right";
  style?: React.CSSProperties;
}
export function RelativeTime(props: RelativeTimeProps): JSX.Element;
