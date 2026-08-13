import * as React from "react";
/** Hover tooltip: ink panel, mono 11px. For technical detail (absolute timestamps, full hashes). */
export interface TooltipProps {
  label: React.ReactNode;
  /** @default "top" */
  side?: "top" | "bottom";
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export declare function Tooltip(props: TooltipProps): React.ReactElement;
