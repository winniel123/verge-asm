import * as React from "react";
export interface StatusDotProps {
  /** running = the live scan pulse */
  status?: "ok" | "warn" | "danger" | "neutral" | "running";
  label?: string;
  /** Render the label as a mono micro-label */
  micro?: boolean;
  style?: React.CSSProperties;
}
export function StatusDot(props: StatusDotProps): JSX.Element;
