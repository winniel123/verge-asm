import * as React from "react";
export interface BarChartProps {
  data: number[];
  /** Sparse axis labels — non-empty entries are distributed across the width (first flush left, last flush right) */
  labels?: string[];
  /** Bar area height px. Default 72 */
  height?: number;
  /** Default var(--chart-1) */
  color?: string;
  /** Dim all but the newest bar (default true) */
  emphasizeLast?: boolean;
  showBaseline?: boolean;
  style?: React.CSSProperties;
}
export function BarChart(props: BarChartProps): JSX.Element;
