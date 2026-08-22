import * as React from "react";
export interface DeltaChipProps {
  /** e.g. "+3", "\u22120.6d" — the leading sign draws the arrow */
  value: string | number;
  /** Whether the direction is good news; "bad" for a rising open count */
  tone?: "good" | "bad" | "neutral";
  size?: "sm" | "md";
  style?: React.CSSProperties;
}
export function DeltaChip(props: DeltaChipProps): JSX.Element;
