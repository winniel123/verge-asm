import * as React from "react";
export interface SparklineProps {
  data: number[];
  /** Default 140\u00d736 */
  width?: number;
  height?: number;
  /** Default var(--chart-1). Use --chart-2..4 for additional series */
  color?: string;
  /** Soft area fill under the line (default true) */
  area?: boolean;
  strokeWidth?: number;
  style?: React.CSSProperties;
}
export function Sparkline(props: SparklineProps): JSX.Element;
