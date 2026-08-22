import * as React from "react";
export interface SkeletonProps {
  shape?: "line" | "block" | "circle";
  width?: number | string;
  height?: number | string;
  /** For shape="line": number of stacked lines (last one shortens) */
  lines?: number;
  /** 12px line height to match mono values */
  mono?: boolean;
  style?: React.CSSProperties;
}
export function Skeleton(props: SkeletonProps): JSX.Element;
