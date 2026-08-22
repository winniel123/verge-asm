import * as React from "react";
export interface ReportCardProps {
  /** Micro-label heading, e.g. "Open signals" */
  title: string;
  /** Mono period readout, e.g. "Aug 8 \u2013 Aug 22" */
  period?: string;
  /** Pre-formatted KPI value */
  value: string;
  delta?: string;
  deltaTone?: "good" | "bad" | "neutral";
  caption?: string;
  /** Right-corner node (e.g. DropdownMenu) */
  action?: React.ReactNode;
  /** Chart slot (Sparkline / BarChart), bottom-aligned */
  children?: React.ReactNode;
  style?: React.CSSProperties;
}
export function ReportCard(props: ReportCardProps): JSX.Element;
