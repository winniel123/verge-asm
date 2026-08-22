import * as React from "react";
export interface TimeSeries {
  label: string;
  data: number[];
  /** Default var(--chart-1..4) by index */
  color?: string;
}
export interface TimeSeriesChartProps {
  series: TimeSeries[];
  /** Sparse x-axis tick labels, one slot per data index ("" = no tick) */
  labels?: string[];
  /** Dense labels for the hover readout (falls back to labels) */
  hoverLabels?: string[];
  /** Default 220 */
  height?: number;
  yFormat?: (v: number) => string;
  /** aria-label for the svg */
  label?: string;
  /** Default: true when series.length > 1 */
  showLegend?: boolean;
  style?: React.CSSProperties;
}
export function TimeSeriesChart(props: TimeSeriesChartProps): JSX.Element;
