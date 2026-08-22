import * as React from "react";
export interface StatProps {
  /** Micro-label caption, e.g. "Open signals" */
  label: string;
  /** Pre-formatted: thousands separators, e.g. "1,284" */
  value: string;
  /** Signed with a true minus: "+12" or "\u22125" */
  delta?: string;
  /** Colors the delta — in security, up is not always good */
  deltaTone?: "good" | "bad" | "neutral";
  /** Muted line under the number */
  caption?: string;
  /** Pulsing dot next to the label */
  live?: boolean;
  style?: React.CSSProperties;
}
export function Stat(props: StatProps): JSX.Element;
