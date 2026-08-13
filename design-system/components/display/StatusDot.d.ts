import * as React from "react";
/** Round status dot with optional mono label; pulse = live activity. */
export interface StatusDotProps {
  /** @default "ok" */
  tone?: "ok" | "warn" | "danger" | "accent" | "neutral";
  /** verge-pulse blink for live processes (running scans). @default false */
  pulse?: boolean;
  /** Mono 11px label beside the dot, colored to match. */
  label?: React.ReactNode;
  /** Dot diameter px. @default 8 */
  size?: number;
  style?: React.CSSProperties;
}
export declare function StatusDot(props: StatusDotProps): React.ReactElement;
