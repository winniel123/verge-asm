import * as React from "react";
export interface ProgressProps {
  /** 0\u2013100; omit for the indeterminate scan sweep */
  value?: number | null;
  /** Micro-label above the bar */
  label?: string;
  /** Mono right-aligned readout, e.g. "142/214 subjects" */
  detail?: string;
  tone?: "accent" | "ok" | "warn" | "danger";
  /** md 6px / sm 4px */
  size?: "sm" | "md";
  style?: React.CSSProperties;
}
export function Progress(props: ProgressProps): JSX.Element;
