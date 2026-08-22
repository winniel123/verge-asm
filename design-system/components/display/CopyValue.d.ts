import * as React from "react";
export interface CopyValueProps {
  /** The copied text */
  value: string;
  /** Optional shortened display (value still copied in full) */
  display?: string;
  /** Mono font size px. Default 12.5 */
  size?: number;
  style?: React.CSSProperties;
}
export function CopyValue(props: CopyValueProps): JSX.Element;
