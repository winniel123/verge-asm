import * as React from "react";
export interface TooltipProps {
  content: React.ReactNode;
  side?: "top" | "bottom" | "left" | "right";
  /** Mono content — e.g. the ISO 8601 timestamp behind a relative time */
  mono?: boolean;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Tooltip(props: TooltipProps): JSX.Element;
