import * as React from "react";
export interface CoverageMeterProps {
  /** Micro-label, e.g. the scope */
  label?: string;
  counted: number;
  /** Omit for the census (no-denominator) state — name scopes and custody extensions */
  total?: number | null;
  /** e.g. "addresses", "names" */
  unit?: string;
  /** Muted line under the bar */
  detail?: string;
  size?: "sm" | "md";
  style?: React.CSSProperties;
}
export function CoverageMeter(props: CoverageMeterProps): JSX.Element;
