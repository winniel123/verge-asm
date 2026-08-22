import * as React from "react";
export interface LogLine {
  /** Mono timestamp, e.g. "14:02:11" */
  time: string;
  level?: "info" | "warn" | "error";
  text: string;
}
export interface LogViewerProps {
  lines: LogLine[];
  /** Micro-label bar, e.g. the batch id */
  title?: string;
  /** Pulsing indicator + auto-follow */
  live?: boolean;
  /** Scroll area height px. Default 220 */
  height?: number;
  style?: React.CSSProperties;
}
export function LogViewer(props: LogViewerProps): JSX.Element;
